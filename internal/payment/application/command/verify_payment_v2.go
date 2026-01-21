package command

import (
	"context"
	"strconv"

	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/service"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

// PaymentRequirementsV2 represents x402 v2 payment requirements
type PaymentRequirementsV2 struct {
	Scheme            string
	Network           string // CAIP-2 format
	Amount            string
	Asset             string
	PayTo             string
	MaxTimeoutSeconds int
	Extra             map[string]interface{}
}

// PaymentPayloadV2 represents x402 v2 payment payload
type PaymentPayloadV2 struct {
	X402Version int
	Payload     map[string]interface{}
	Accepted    PaymentRequirementsV2
}

// VerifyPaymentCommandV2 represents a v2 request to verify a payment
type VerifyPaymentCommandV2 struct {
	X402Version         int
	PaymentPayload      PaymentPayloadV2
	PaymentRequirements PaymentRequirementsV2
}

// VerifyPaymentResultV2 represents the v2 result of a verification
type VerifyPaymentResultV2 struct {
	IsValid       bool
	InvalidReason string
	Payer         string
}

// VerifyPaymentHandlerV2 handles v2 verify payment commands
type VerifyPaymentHandlerV2 struct {
	blockchainClient BlockchainClient
	verificationSvc  *service.VerificationService
}

// NewVerifyPaymentHandlerV2 creates a new VerifyPaymentHandlerV2
func NewVerifyPaymentHandlerV2(client BlockchainClient, verificationSvc *service.VerificationService) *VerifyPaymentHandlerV2 {
	return &VerifyPaymentHandlerV2{
		blockchainClient: client,
		verificationSvc:  verificationSvc,
	}
}

// Handle processes the v2 verify payment command
func (h *VerifyPaymentHandlerV2) Handle(ctx context.Context, cmd VerifyPaymentCommandV2) (VerifyPaymentResultV2, error) {
	// Validate x402 version
	if cmd.X402Version != 2 {
		return VerifyPaymentResultV2{
			IsValid:       false,
			InvalidReason: valueobject.ErrInvalidX402Version,
		}, nil
	}

	// Validate scheme
	if cmd.PaymentRequirements.Scheme != "exact" {
		return VerifyPaymentResultV2{
			IsValid:       false,
			InvalidReason: valueobject.ErrUnsupportedScheme,
		}, nil
	}

	// Parse network from CAIP-2 format
	network, err := valueobject.NewNetworkFromCAIP2(cmd.PaymentRequirements.Network)
	if err != nil {
		return VerifyPaymentResultV2{
			IsValid:       false,
			InvalidReason: valueobject.ErrInvalidNetwork,
		}, nil
	}

	// Extract transaction from payload
	txHex, ok := cmd.PaymentPayload.Payload["transaction"].(string)
	if !ok || txHex == "" {
		return VerifyPaymentResultV2{
			IsValid:       false,
			InvalidReason: valueobject.ErrInvalidPayload,
		}, nil
	}

	// Determine token type from asset
	tokenType := assetToTokenType(cmd.PaymentRequirements.Asset)

	// Parse amount
	amount, err := strconv.ParseUint(cmd.PaymentRequirements.Amount, 10, 64)
	if err != nil {
		return VerifyPaymentResultV2{
			IsValid:       false,
			InvalidReason: valueobject.ErrInvalidPaymentRequirements,
		}, nil
	}

	// Parse recipient address
	expectedRecipient, err := valueobject.NewStacksAddress(cmd.PaymentRequirements.PayTo)
	if err != nil {
		return VerifyPaymentResultV2{
			IsValid:       false,
			InvalidReason: valueobject.ErrInvalidPaymentRequirements,
		}, nil
	}

	// For v2 verify, we need to decode the transaction and extract sender
	// Since we don't have the tx on chain yet (it's just signed, not broadcast),
	// we need to decode and validate the transaction hex
	sender, err := extractSenderFromTransaction(txHex, network)
	if err != nil {
		return VerifyPaymentResultV2{
			IsValid:       false,
			InvalidReason: valueobject.ErrInvalidPayload,
			Payer:         "",
		}, nil
	}

	// Validate transaction structure and extract details
	valid, invalidReason := validateTransactionV2(txHex, expectedRecipient, valueobject.NewAmount(amount), tokenType, network)

	return VerifyPaymentResultV2{
		IsValid:       valid,
		InvalidReason: invalidReason,
		Payer:         sender,
	}, nil
}

// assetToTokenType converts asset identifier to TokenType
func assetToTokenType(asset string) valueobject.TokenType {
	switch asset {
	case "STX":
		return valueobject.TokenSTX
	case "SBTC":
		return valueobject.TokenSBTC
	case "USDCX":
		return valueobject.TokenUSDCX
	default:
		// Check if it's a contract identifier (address.name)
		if len(asset) > 0 && asset[0] == 'S' {
			// Could be a contract, try to determine from known contracts
			return valueobject.TokenSTX // Default to STX for unknown
		}
		return valueobject.TokenSTX
	}
}

// extractSenderFromTransaction extracts the sender address from a signed transaction hex
// This is a placeholder - actual implementation requires Stacks transaction deserialization
func extractSenderFromTransaction(txHex string, network valueobject.Network) (string, error) {
	// TODO: Implement proper transaction deserialization
	// For now, this would require a Stacks transaction parsing library in Go
	// or calling an external service to decode the transaction

	// Placeholder: return empty string, actual implementation needed
	// The sender would be extracted from the transaction's auth field
	return "", nil
}

// validateTransactionV2 validates a signed transaction without broadcasting
// Returns (isValid, invalidReason)
func validateTransactionV2(txHex string, expectedRecipient valueobject.StacksAddress, minAmount valueobject.Amount, tokenType valueobject.TokenType, network valueobject.Network) (bool, string) {
	// TODO: Implement proper transaction validation
	// This would require:
	// 1. Deserialize the transaction
	// 2. Verify signature is valid
	// 3. Check recipient matches
	// 4. Check amount is sufficient
	// 5. Check token type matches

	// For now, we'll do basic validation and defer full validation to settle
	if txHex == "" {
		return false, valueobject.ErrInvalidPayload
	}

	// Basic hex validation
	if len(txHex) < 10 {
		return false, valueobject.ErrInvalidPayload
	}

	// Transaction appears valid (will be fully validated on broadcast)
	return true, ""
}
