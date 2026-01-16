package coinbase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

type mockBalanceChecker struct {
	stxBalance   uint64
	tokenBalance uint64
	err          error
}

func (m *mockBalanceChecker) GetSTXBalance(ctx context.Context, address string, network valueobject.Network) (uint64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.stxBalance, nil
}

func (m *mockBalanceChecker) GetTokenBalance(ctx context.Context, address string, contractID string, network valueobject.Network) (uint64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.tokenBalance, nil
}

func TestPreSubmitVerifier_Verify_Success(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.True(t, result.Valid)
	assert.Equal(t, "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7", result.Payer)
	assert.Empty(t, result.ErrorCode)
}

func TestPreSubmitVerifier_Verify_MissingSender(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "",
			To:    "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.False(t, result.Valid)
	assert.Equal(t, ErrorCodeMalformedPayload, result.ErrorCode)
	assert.Contains(t, result.Errors[0], "missing sender address")
}

func TestPreSubmitVerifier_Verify_MissingRecipient(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "",
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.False(t, result.Valid)
	assert.Equal(t, ErrorCodeMalformedPayload, result.ErrorCode)
	assert.Contains(t, result.Errors[0], "missing recipient address")
}

func TestPreSubmitVerifier_Verify_MissingValue(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
			Value: "",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.False(t, result.Valid)
	assert.Equal(t, ErrorCodeMalformedPayload, result.ErrorCode)
	assert.Contains(t, result.Errors[0], "missing value")
}

func TestPreSubmitVerifier_Verify_InvalidSenderAddress(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "invalid-address",
			To:    "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.False(t, result.Valid)
	assert.Equal(t, ErrorCodeMalformedPayload, result.ErrorCode)
	assert.Contains(t, result.Errors[0], "invalid sender address")
}

func TestPreSubmitVerifier_Verify_RecipientMismatch(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "ST3NBRSFKX28FQ2ZJ1MAKX58HKHSDTV2KJ3DKZE1B",
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.False(t, result.Valid)
	assert.Equal(t, ErrorCodeRecipientMismatch, result.ErrorCode)
	assert.Contains(t, result.Errors[0], "recipient mismatch")
}

func TestPreSubmitVerifier_Verify_AmountTooLow(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
			Value: "500000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.False(t, result.Valid)
	assert.Equal(t, ErrorCodeAmountTooLow, result.ErrorCode)
	assert.Contains(t, result.Errors[0], "amount too low")
}

func TestPreSubmitVerifier_Verify_MissingSignedTransaction(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.False(t, result.Valid)
	assert.Equal(t, ErrorCodeMissingPayload, result.ErrorCode)
	assert.Contains(t, result.Errors[0], "missing signed transaction")
}

func TestPreSubmitVerifier_Verify_InsufficientBalance(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 500000} // Less than required
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.False(t, result.Valid)
	assert.Equal(t, ErrorCodeInsufficientFunds, result.ErrorCode)
	assert.Contains(t, result.Errors[0], "insufficient balance")
}

func TestPreSubmitVerifier_Verify_BalanceCheckError(t *testing.T) {
	checker := &mockBalanceChecker{err: errors.New("network error")}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	// Balance check errors should result in valid=true with a warning
	// (blockchain will validate actual balance on broadcast)
	result := verifier.Verify(context.Background(), payload, requirements)

	assert.True(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "balance check warning")
}

func TestPreSubmitVerifier_Verify_TokenBalance(t *testing.T) {
	checker := &mockBalanceChecker{tokenBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSBTC,
		Asset:             "SM3VDXK3WZZSA84XXFKAFAF15NNZX32CTSG82JFQ4.sbtc-token",
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.True(t, result.Valid)
	assert.Empty(t, result.ErrorCode)
}

func TestPreSubmitVerifier_Verify_RecipientCaseInsensitive(t *testing.T) {
	checker := &mockBalanceChecker{stxBalance: 2000000}
	verifier := NewPreSubmitVerifier(checker)

	payload := StacksPayload{
		SignedTransaction: "0x00000001deadbeef",
		Authorization: Authorization{
			From:  "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			To:    "st1pqhqkv0rjxzfy1dgx8mnsnyve3vgzjsrtpgzgm", // lowercase
			Value: "1000000",
		},
	}

	requirements := VerifyRequirements{
		ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM", // uppercase
		MinAmount:         1000000,
		Network:           valueobject.NetworkTestnet,
		TokenType:         valueobject.TokenSTX,
	}

	result := verifier.Verify(context.Background(), payload, requirements)

	assert.True(t, result.Valid)
}
