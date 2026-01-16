package stacks

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/service"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

// TransactionResponse represents the API response for a transaction
type TransactionResponse struct {
	TxID          string             `json:"tx_id"`
	TxStatus      string             `json:"tx_status"`
	TxType        string             `json:"tx_type"`
	BlockHeight   uint64             `json:"block_height"`
	Fee           string             `json:"fee_rate"`
	Nonce         uint64             `json:"nonce"`
	SenderAddress string             `json:"sender_address"`
	TokenTransfer *TokenTransferData `json:"token_transfer,omitempty"`
	ContractCall  *ContractCallData  `json:"contract_call,omitempty"`
}

// TokenTransferData represents STX transfer data
type TokenTransferData struct {
	RecipientAddress string `json:"recipient_address"`
	Amount           string `json:"amount"`
	Memo             string `json:"memo"`
}

// ContractCallData represents a contract call (for SIP-010 tokens)
type ContractCallData struct {
	ContractID   string                   `json:"contract_id"`
	FunctionName string                   `json:"function_name"`
	FunctionArgs []ContractFunctionArgRaw `json:"function_args"`
}

// ContractFunctionArgRaw represents a raw function argument
type ContractFunctionArgRaw struct {
	Hex  string `json:"hex"`
	Repr string `json:"repr"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// Client is a Stacks blockchain API client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Stacks client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientForNetwork creates a client for the specified network
func NewClientForNetwork(network valueobject.Network) *Client {
	return NewClient(network.APIBaseURL())
}

// GetTransaction fetches a transaction by ID
func (c *Client) GetTransaction(ctx context.Context, txID valueobject.TransactionID) (service.BlockchainTransaction, error) {
	url := fmt.Sprintf("%s/extended/v1/tx/%s", c.baseURL, txID.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return service.BlockchainTransaction{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return service.BlockchainTransaction{}, fmt.Errorf("failed to fetch transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return service.BlockchainTransaction{}, errors.New("transaction not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return service.BlockchainTransaction{}, fmt.Errorf("API error: %s", string(body))
	}

	var txResp TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return service.BlockchainTransaction{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.parseTransactionResponse(txResp, valueobject.TokenSTX)
}

// GetTransactionWithTokenType fetches a transaction and parses it for a specific token type
func (c *Client) GetTransactionWithTokenType(ctx context.Context, txID valueobject.TransactionID, tokenType valueobject.TokenType, network valueobject.Network) (service.BlockchainTransaction, error) {
	url := fmt.Sprintf("%s/extended/v1/tx/%s", c.baseURL, txID.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return service.BlockchainTransaction{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return service.BlockchainTransaction{}, fmt.Errorf("failed to fetch transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return service.BlockchainTransaction{}, errors.New("transaction not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return service.BlockchainTransaction{}, fmt.Errorf("API error: %s", string(body))
	}

	var txResp TransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return service.BlockchainTransaction{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.parseTransactionResponse(txResp, tokenType)
}

// BroadcastTransaction broadcasts a signed transaction to the network
func (c *Client) BroadcastTransaction(ctx context.Context, signedTx string) (valueobject.TransactionID, error) {
	url := fmt.Sprintf("%s/v2/transactions", c.baseURL)

	// Remove 0x prefix if present
	txHex := strings.TrimPrefix(signedTx, "0x")

	// Decode hex to bytes
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return valueobject.TransactionID{}, fmt.Errorf("invalid transaction hex: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(txBytes))
	if err != nil {
		return valueobject.TransactionID{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return valueobject.TransactionID{}, fmt.Errorf("failed to broadcast transaction: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return valueobject.TransactionID{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return valueobject.TransactionID{}, fmt.Errorf("broadcast failed: %s", string(body))
	}

	// Parse transaction ID from response (comes as JSON string with quotes)
	var txIDStr string
	if err := json.Unmarshal(body, &txIDStr); err != nil {
		// Try without quotes
		txIDStr = strings.Trim(string(body), "\"")
	}

	return valueobject.NewTransactionID(txIDStr)
}

// parseTransactionResponse converts API response to domain model
func (c *Client) parseTransactionResponse(resp TransactionResponse, tokenType valueobject.TokenType) (service.BlockchainTransaction, error) {
	txID, err := valueobject.NewTransactionID(resp.TxID)
	if err != nil {
		return service.BlockchainTransaction{}, fmt.Errorf("invalid tx ID: %w", err)
	}

	sender, err := valueobject.NewStacksAddress(resp.SenderAddress)
	if err != nil {
		return service.BlockchainTransaction{}, fmt.Errorf("invalid sender address: %w", err)
	}

	var recipient valueobject.StacksAddress
	var amount valueobject.Amount
	var memo string

	// Parse based on transaction type
	if resp.TxType == "token_transfer" && resp.TokenTransfer != nil {
		recipient, err = valueobject.NewStacksAddress(resp.TokenTransfer.RecipientAddress)
		if err != nil {
			return service.BlockchainTransaction{}, fmt.Errorf("invalid recipient address: %w", err)
		}

		amountVal, err := strconv.ParseUint(resp.TokenTransfer.Amount, 10, 64)
		if err != nil {
			return service.BlockchainTransaction{}, fmt.Errorf("invalid amount: %w", err)
		}
		amount = valueobject.NewAmount(amountVal)
		memo = resp.TokenTransfer.Memo
	} else if resp.TxType == "contract_call" && resp.ContractCall != nil {
		// Parse SIP-010 transfer
		parsedRecipient, parsedAmount, parsedMemo, err := parseSIP010Transfer(resp.ContractCall)
		if err != nil {
			return service.BlockchainTransaction{}, err
		}
		recipient = parsedRecipient
		amount = parsedAmount
		memo = parsedMemo
	} else {
		return service.BlockchainTransaction{}, errors.New("unsupported transaction type")
	}

	fee, _ := strconv.ParseUint(resp.Fee, 10, 64)

	return service.BlockchainTransaction{
		TxID:        txID,
		TokenType:   tokenType,
		Sender:      sender,
		Recipient:   recipient,
		Amount:      amount,
		Fee:         valueobject.NewAmount(fee),
		Nonce:       resp.Nonce,
		BlockHeight: resp.BlockHeight,
		Memo:        memo,
		Status:      resp.TxStatus,
		IsConfirmed: IsTransactionConfirmed(resp.TxStatus, resp.BlockHeight),
	}, nil
}

// parseSIP010Transfer parses a SIP-010 contract call (sBTC, USDCx)
func parseSIP010Transfer(call *ContractCallData) (valueobject.StacksAddress, valueobject.Amount, string, error) {
	if call.FunctionName != "transfer" {
		return valueobject.StacksAddress{}, valueobject.Amount{}, "", errors.New("not a transfer function")
	}

	var recipient valueobject.StacksAddress
	var amount valueobject.Amount
	var memo string

	for _, arg := range call.FunctionArgs {
		switch arg.Name {
		case "amount":
			amountVal, err := strconv.ParseUint(strings.TrimPrefix(arg.Repr, "u"), 10, 64)
			if err != nil {
				return valueobject.StacksAddress{}, valueobject.Amount{}, "", fmt.Errorf("invalid amount: %w", err)
			}
			amount = valueobject.NewAmount(amountVal)
		case "recipient", "to":
			addr := strings.TrimPrefix(arg.Repr, "'")
			var err error
			recipient, err = valueobject.NewStacksAddress(addr)
			if err != nil {
				return valueobject.StacksAddress{}, valueobject.Amount{}, "", fmt.Errorf("invalid recipient: %w", err)
			}
		case "memo":
			memo = arg.Repr
		}
	}

	if recipient.IsZero() {
		return valueobject.StacksAddress{}, valueobject.Amount{}, "", errors.New("recipient not found in contract call")
	}

	return recipient, amount, memo, nil
}

// IsTransactionConfirmed checks if a transaction is confirmed
func IsTransactionConfirmed(status string, blockHeight uint64) bool {
	return status == "success" && blockHeight > 0
}

// IsTransactionFailed checks if a transaction has failed
func IsTransactionFailed(status string) bool {
	failedStatuses := []string{
		"failed",
		"abort_by_response",
		"abort_by_post_condition",
	}
	for _, s := range failedStatuses {
		if status == s {
			return true
		}
	}
	return false
}

// AccountBalanceResponse represents the account balance API response
type AccountBalanceResponse struct {
	STX            STXBalance            `json:"stx"`
	FungibleTokens map[string]TokenBalance `json:"fungible_tokens"`
}

// STXBalance represents STX balance info
type STXBalance struct {
	Balance       string `json:"balance"`
	LockedBalance string `json:"locked"`
}

// TokenBalance represents a fungible token balance
type TokenBalance struct {
	Balance string `json:"balance"`
}

// GetSTXBalance returns the STX balance for an address
func (c *Client) GetSTXBalance(ctx context.Context, address string) (uint64, error) {
	url := fmt.Sprintf("%s/extended/v1/address/%s/stx", c.baseURL, address)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API error: %s", string(body))
	}

	var balanceResp STXBalance
	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	balance, err := strconv.ParseUint(balanceResp.Balance, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid balance: %w", err)
	}

	return balance, nil
}

// GetTokenBalance returns the balance for a specific fungible token
func (c *Client) GetTokenBalance(ctx context.Context, address string, contractID string) (uint64, error) {
	url := fmt.Sprintf("%s/extended/v1/address/%s/balances", c.baseURL, address)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API error: %s", string(body))
	}

	var balanceResp AccountBalanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&balanceResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	tokenBalance, exists := balanceResp.FungibleTokens[contractID]
	if !exists {
		return 0, nil // No balance for this token
	}

	balance, err := strconv.ParseUint(tokenBalance.Balance, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid balance: %w", err)
	}

	return balance, nil
}
