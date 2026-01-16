package coinbase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

// BalanceChecker defines the interface for checking account balances
type BalanceChecker interface {
	GetSTXBalance(ctx context.Context, address string, network valueobject.Network) (uint64, error)
	GetTokenBalance(ctx context.Context, address string, contractID string, network valueobject.Network) (uint64, error)
}

// PreSubmitVerifier validates signed transactions before submission
type PreSubmitVerifier struct {
	balanceChecker BalanceChecker
}

// NewPreSubmitVerifier creates a new PreSubmitVerifier
func NewPreSubmitVerifier(balanceChecker BalanceChecker) *PreSubmitVerifier {
	return &PreSubmitVerifier{
		balanceChecker: balanceChecker,
	}
}

// VerifyRequirements contains the expected values for verification
type VerifyRequirements struct {
	ExpectedRecipient string
	MinAmount         uint64
	Network           valueobject.Network
	TokenType         valueobject.TokenType
	Asset             string // Contract ID for SIP-010 tokens
}

// PreSubmitResult contains the verification result
type PreSubmitResult struct {
	Valid     bool
	Payer     string
	ErrorCode string
	Errors    []string
}

// Verify validates a signed transaction against requirements without broadcasting
func (v *PreSubmitVerifier) Verify(ctx context.Context, payload StacksPayload, requirements VerifyRequirements) *PreSubmitResult {
	result := &PreSubmitResult{
		Valid: true,
		Payer: payload.Authorization.From,
	}

	// Validate authorization fields are present
	if payload.Authorization.From == "" {
		result.Valid = false
		result.ErrorCode = ErrorCodeMalformedPayload
		result.Errors = append(result.Errors, "missing sender address in authorization")
		return result
	}

	if payload.Authorization.To == "" {
		result.Valid = false
		result.ErrorCode = ErrorCodeMalformedPayload
		result.Errors = append(result.Errors, "missing recipient address in authorization")
		return result
	}

	if payload.Authorization.Value == "" {
		result.Valid = false
		result.ErrorCode = ErrorCodeMalformedPayload
		result.Errors = append(result.Errors, "missing value in authorization")
		return result
	}

	// Parse and validate sender address
	_, err := valueobject.NewStacksAddress(payload.Authorization.From)
	if err != nil {
		result.Valid = false
		result.ErrorCode = ErrorCodeMalformedPayload
		result.Errors = append(result.Errors, fmt.Sprintf("invalid sender address: %s", err.Error()))
		return result
	}

	// Parse and validate recipient address
	_, err = valueobject.NewStacksAddress(payload.Authorization.To)
	if err != nil {
		result.Valid = false
		result.ErrorCode = ErrorCodeMalformedPayload
		result.Errors = append(result.Errors, fmt.Sprintf("invalid recipient address: %s", err.Error()))
		return result
	}

	// Validate recipient matches requirement
	if !addressesMatch(payload.Authorization.To, requirements.ExpectedRecipient) {
		result.Valid = false
		result.ErrorCode = ErrorCodeRecipientMismatch
		result.Errors = append(result.Errors, fmt.Sprintf(
			"recipient mismatch: got %s, expected %s",
			payload.Authorization.To, requirements.ExpectedRecipient,
		))
		return result
	}

	// Parse and validate amount
	amount, err := strconv.ParseUint(payload.Authorization.Value, 10, 64)
	if err != nil {
		result.Valid = false
		result.ErrorCode = ErrorCodeMalformedPayload
		result.Errors = append(result.Errors, fmt.Sprintf("invalid amount: %s", err.Error()))
		return result
	}

	// Validate amount meets minimum
	if amount < requirements.MinAmount {
		result.Valid = false
		result.ErrorCode = ErrorCodeAmountTooLow
		result.Errors = append(result.Errors, fmt.Sprintf(
			"amount too low: got %d, minimum required %d",
			amount, requirements.MinAmount,
		))
		return result
	}

	// Validate signed transaction is present
	if payload.SignedTransaction == "" {
		result.Valid = false
		result.ErrorCode = ErrorCodeMissingPayload
		result.Errors = append(result.Errors, "missing signed transaction")
		return result
	}

	// Check sender has sufficient balance
	var balance uint64
	if requirements.TokenType.IsNative() {
		balance, err = v.balanceChecker.GetSTXBalance(ctx, payload.Authorization.From, requirements.Network)
	} else {
		balance, err = v.balanceChecker.GetTokenBalance(ctx, payload.Authorization.From, requirements.Asset, requirements.Network)
	}

	if err != nil {
		// Balance check failed - we allow transaction to proceed but log the issue
		// The blockchain will reject if balance is actually insufficient
		result.Errors = append(result.Errors, fmt.Sprintf("balance check warning: %s", err.Error()))
	} else if balance < amount {
		result.Valid = false
		result.ErrorCode = ErrorCodeInsufficientFunds
		result.Errors = append(result.Errors, fmt.Sprintf(
			"insufficient balance: has %d, needs %d",
			balance, amount,
		))
		return result
	}

	return result
}

// addressesMatch compares two Stacks addresses for equality
// Handles case-insensitivity for the prefix (SP/ST) and hash portion
func addressesMatch(a, b string) bool {
	return strings.EqualFold(a, b)
}
