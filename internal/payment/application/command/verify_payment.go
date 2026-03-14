package command

import (
	"context"
	"fmt"
	"time"

	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/service"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

// Verify retry defaults: 10 retries × 2s = 20s window.
// Verify only fetches an existing transaction, so a shorter window is acceptable.
const (
	DefaultVerifyMaxRetries = 10
	DefaultVerifyRetryDelay = 2 * time.Second
)

// BlockchainClient interface for fetching transactions
type BlockchainClient interface {
	GetTransactionWithRetry(ctx context.Context, txID valueobject.TransactionID, tokenType valueobject.TokenType, network valueobject.Network, maxRetries int, retryDelay time.Duration) (service.BlockchainTransaction, error)
}

// VerifyPaymentCommand represents a request to verify a payment
type VerifyPaymentCommand struct {
	TxID              string
	TokenType         string
	ExpectedRecipient string
	MinAmount         uint64
	ExpectedSender    *string
	ExpectedMemo      *string
	Network           string
}

// VerifyPaymentResult represents the result of a verification
type VerifyPaymentResult struct {
	Valid            bool
	TxID             string
	SenderAddress    string
	RecipientAddress string
	Amount           uint64
	Fee              uint64
	Nonce            uint64
	Status           string
	BlockHeight      uint64
	TokenType        string
	Memo             string
	Network          string
	Errors           []string
}

// VerifyPaymentHandler handles verify payment commands
type VerifyPaymentHandler struct {
	blockchainClient  BlockchainClient
	verificationSvc   *service.VerificationService
	maxRetries        int
	retryDelay        time.Duration
}

// NewVerifyPaymentHandler creates a new VerifyPaymentHandler
func NewVerifyPaymentHandler(client BlockchainClient, verificationSvc *service.VerificationService) *VerifyPaymentHandler {
	return &VerifyPaymentHandler{
		blockchainClient:  client,
		verificationSvc:   verificationSvc,
		maxRetries:        DefaultVerifyMaxRetries,
		retryDelay:        DefaultVerifyRetryDelay,
	}
}

// Handle processes the verify payment command
func (h *VerifyPaymentHandler) Handle(ctx context.Context, cmd VerifyPaymentCommand) (VerifyPaymentResult, error) {
	// Parse and validate inputs
	txID, err := valueobject.NewTransactionID(cmd.TxID)
	if err != nil {
		return VerifyPaymentResult{}, fmt.Errorf("invalid transaction ID: %w", err)
	}

	tokenType, err := valueobject.NewTokenType(cmd.TokenType)
	if err != nil {
		tokenType = valueobject.TokenSTX // Default to STX
	}

	network, err := valueobject.NewNetwork(cmd.Network)
	if err != nil {
		return VerifyPaymentResult{}, fmt.Errorf("invalid network: %w", err)
	}

	expectedRecipient, err := valueobject.NewStacksAddress(cmd.ExpectedRecipient)
	if err != nil {
		return VerifyPaymentResult{}, fmt.Errorf("invalid expected recipient: %w", err)
	}

	// Fetch transaction from blockchain
	tx, err := h.blockchainClient.GetTransactionWithRetry(ctx, txID, tokenType, network, h.maxRetries, h.retryDelay)
	if err != nil {
		return VerifyPaymentResult{}, fmt.Errorf("failed to fetch transaction: %w", err)
	}

	// Build verification criteria
	criteria := service.VerificationCriteria{
		ExpectedRecipient: expectedRecipient,
		MinAmount:         valueobject.NewAmount(cmd.MinAmount),
		AcceptUnconfirmed: false, // Always require confirmation
	}

	// Optional sender
	if cmd.ExpectedSender != nil {
		sender, err := valueobject.NewStacksAddress(*cmd.ExpectedSender)
		if err == nil {
			criteria.ExpectedSender = &sender
		}
	}

	// Optional memo
	if cmd.ExpectedMemo != nil {
		criteria.ExpectedMemo = cmd.ExpectedMemo
	}

	// Verify transaction
	verificationResult := h.verificationSvc.Verify(tx, criteria)

	// Determine status
	status := determinePaymentStatus(tx)

	return VerifyPaymentResult{
		Valid:            verificationResult.Valid,
		TxID:             tx.TxID.String(),
		SenderAddress:    tx.Sender.String(),
		RecipientAddress: tx.Recipient.String(),
		Amount:           tx.Amount.Value(),
		Fee:              tx.Fee.Value(),
		Nonce:            tx.Nonce,
		Status:           status,
		BlockHeight:      tx.BlockHeight,
		TokenType:        tx.TokenType.String(),
		Memo:             tx.Memo,
		Network:          network.String(),
		Errors:           verificationResult.Errors,
	}, nil
}

// determinePaymentStatus converts blockchain status to payment status
func determinePaymentStatus(tx service.BlockchainTransaction) string {
	if tx.IsConfirmed {
		return "confirmed"
	}
	if tx.Status == "failed" || tx.Status == "abort_by_response" || tx.Status == "abort_by_post_condition" {
		return "failed"
	}
	return "pending"
}
