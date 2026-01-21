package command

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/service"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

// SettlePaymentCommandV2 represents a v2 request to settle a payment
type SettlePaymentCommandV2 struct {
	X402Version         int
	PaymentPayload      PaymentPayloadV2
	PaymentRequirements PaymentRequirementsV2
}

// SettlePaymentResultV2 represents the v2 result of a settlement
type SettlePaymentResultV2 struct {
	Success     bool
	ErrorReason string
	Payer       string
	Transaction string
	Network     string // CAIP-2 format
}

// SettlePaymentHandlerV2 handles v2 settle payment commands
type SettlePaymentHandlerV2 struct {
	broadcaster     TransactionBroadcaster
	verificationSvc *service.VerificationService
	maxRetries      int
	retryDelay      time.Duration
}

// NewSettlePaymentHandlerV2 creates a new SettlePaymentHandlerV2
func NewSettlePaymentHandlerV2(broadcaster TransactionBroadcaster, verificationSvc *service.VerificationService) *SettlePaymentHandlerV2 {
	return &SettlePaymentHandlerV2{
		broadcaster:     broadcaster,
		verificationSvc: verificationSvc,
		maxRetries:      15,
		retryDelay:      2 * time.Second,
	}
}

// Handle processes the v2 settle payment command
func (h *SettlePaymentHandlerV2) Handle(ctx context.Context, cmd SettlePaymentCommandV2) (SettlePaymentResultV2, error) {
	// Validate x402 version
	if cmd.X402Version != 2 {
		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: valueobject.ErrInvalidX402Version,
			Network:     cmd.PaymentRequirements.Network,
		}, nil
	}

	// Validate scheme
	if cmd.PaymentRequirements.Scheme != "exact" {
		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: valueobject.ErrUnsupportedScheme,
			Network:     cmd.PaymentRequirements.Network,
		}, nil
	}

	// Parse network from CAIP-2 format
	network, err := valueobject.NewNetworkFromCAIP2(cmd.PaymentRequirements.Network)
	if err != nil {
		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: valueobject.ErrInvalidNetwork,
			Network:     cmd.PaymentRequirements.Network,
		}, nil
	}

	// Extract transaction from payload
	txHex, ok := cmd.PaymentPayload.Payload["transaction"].(string)
	if !ok || txHex == "" {
		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: valueobject.ErrInvalidPayload,
			Network:     cmd.PaymentRequirements.Network,
		}, nil
	}

	// Determine token type from asset
	tokenType := assetToTokenType(cmd.PaymentRequirements.Asset)

	// Parse amount
	amount, err := strconv.ParseUint(cmd.PaymentRequirements.Amount, 10, 64)
	if err != nil {
		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: valueobject.ErrInvalidPaymentRequirements,
			Network:     cmd.PaymentRequirements.Network,
		}, nil
	}

	// Parse expected recipient
	expectedRecipient, err := valueobject.NewStacksAddress(cmd.PaymentRequirements.PayTo)
	if err != nil {
		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: valueobject.ErrInvalidPaymentRequirements,
			Network:     cmd.PaymentRequirements.Network,
		}, nil
	}

	// Broadcast the transaction
	txID, err := h.broadcaster.BroadcastTransaction(ctx, txHex, network)
	if err != nil {
		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: valueobject.ErrBroadcastFailed,
			Network:     cmd.PaymentRequirements.Network,
		}, fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Wait for transaction to be confirmed
	tx, err := h.broadcaster.WaitForConfirmation(ctx, txID, tokenType, network, h.maxRetries, h.retryDelay)
	if err != nil {
		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: valueobject.ErrInvalidTransactionState,
			Transaction: txID.String(),
			Network:     cmd.PaymentRequirements.Network,
		}, fmt.Errorf("failed to confirm transaction: %w", err)
	}

	// Build verification criteria
	criteria := service.VerificationCriteria{
		ExpectedRecipient: expectedRecipient,
		MinAmount:         valueobject.NewAmount(amount),
		AcceptUnconfirmed: false,
	}

	// Verify transaction
	verificationResult := h.verificationSvc.Verify(tx, criteria)

	if !verificationResult.Valid {
		// Determine specific error reason
		errorReason := valueobject.ErrUnexpectedSettleError
		if len(verificationResult.Errors) > 0 {
			// Map error to v2 error code
			errorReason = mapVerificationErrorToV2(verificationResult.Errors[0])
		}

		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: errorReason,
			Payer:       tx.Sender.String(),
			Transaction: tx.TxID.String(),
			Network:     cmd.PaymentRequirements.Network,
		}, nil
	}

	// Check transaction status
	if tx.Status == "failed" || tx.Status == "abort_by_response" || tx.Status == "abort_by_post_condition" {
		return SettlePaymentResultV2{
			Success:     false,
			ErrorReason: valueobject.ErrTransactionFailed,
			Payer:       tx.Sender.String(),
			Transaction: tx.TxID.String(),
			Network:     cmd.PaymentRequirements.Network,
		}, nil
	}

	return SettlePaymentResultV2{
		Success:     true,
		Payer:       tx.Sender.String(),
		Transaction: tx.TxID.String(),
		Network:     cmd.PaymentRequirements.Network,
	}, nil
}

// mapVerificationErrorToV2 maps verification error messages to v2 error codes
func mapVerificationErrorToV2(errorMsg string) string {
	// Map common error patterns to v2 error codes
	switch {
	case contains(errorMsg, "recipient"):
		return valueobject.ErrRecipientMismatch
	case contains(errorMsg, "amount") || contains(errorMsg, "insufficient"):
		return valueobject.ErrAmountInsufficient
	case contains(errorMsg, "sender"):
		return valueobject.ErrSenderMismatch
	case contains(errorMsg, "not found"):
		return valueobject.ErrTransactionNotFound
	case contains(errorMsg, "pending"):
		return valueobject.ErrTransactionPending
	case contains(errorMsg, "failed"):
		return valueobject.ErrTransactionFailed
	default:
		return valueobject.ErrUnexpectedSettleError
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, i, substr) {
			return true
		}
	}
	return false
}

func equalFoldAt(s string, start int, substr string) bool {
	for i := 0; i < len(substr); i++ {
		c1 := s[start+i]
		c2 := substr[i]
		if c1 != c2 && toLower(c1) != toLower(c2) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
