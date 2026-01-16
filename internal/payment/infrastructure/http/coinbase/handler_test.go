package coinbase

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/x402stacks/stacks-facilitator/internal/payment/application/command"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

// MockSettleHandler for testing
type MockSettleHandler struct {
	HandleFn func(ctx context.Context, cmd command.SettlePaymentCommand) (command.SettlePaymentResult, error)
}

func (m *MockSettleHandler) Handle(ctx context.Context, cmd command.SettlePaymentCommand) (command.SettlePaymentResult, error) {
	return m.HandleFn(ctx, cmd)
}

// MockBalanceChecker for testing
type MockBalanceChecker struct {
	STXBalance   uint64
	TokenBalance uint64
	Error        error
}

func (m *MockBalanceChecker) GetSTXBalance(ctx context.Context, address string, network valueobject.Network) (uint64, error) {
	if m.Error != nil {
		return 0, m.Error
	}
	return m.STXBalance, nil
}

func (m *MockBalanceChecker) GetTokenBalance(ctx context.Context, address string, contractID string, network valueobject.Network) (uint64, error) {
	if m.Error != nil {
		return 0, m.Error
	}
	return m.TokenBalance, nil
}

func TestHandler_Supported(t *testing.T) {
	mockBalance := &MockBalanceChecker{STXBalance: 1000000}
	handler := NewHandler(nil, mockBalance)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/supported", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Supported(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response SupportedResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Kinds, 2)
	assert.Equal(t, "exact", response.Kinds[0].Scheme)
	assert.Equal(t, "stacks-mainnet", response.Kinds[0].Network)
	assert.Equal(t, 1, response.Kinds[0].X402Version)
	assert.Equal(t, "stacks-testnet", response.Kinds[1].Network)
}

func TestHandler_Verify_Success(t *testing.T) {
	mockBalance := &MockBalanceChecker{STXBalance: 2000000}
	handler := NewHandler(nil, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "stacks-testnet",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "stacks-testnet",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Verify(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response VerifyResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.IsValid)
	assert.Equal(t, "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7", response.Payer)
	assert.Nil(t, response.InvalidReason)
}

func TestHandler_Verify_InsufficientBalance(t *testing.T) {
	mockBalance := &MockBalanceChecker{STXBalance: 500000} // Less than required
	handler := NewHandler(nil, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "stacks-testnet",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "stacks-testnet",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Verify(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response VerifyResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.IsValid)
	assert.NotNil(t, response.InvalidReason)
	assert.Contains(t, *response.InvalidReason, "insufficient balance")
}

func TestHandler_Verify_RecipientMismatch(t *testing.T) {
	mockBalance := &MockBalanceChecker{STXBalance: 2000000}
	handler := NewHandler(nil, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "stacks-testnet",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST3NBRSFKX28FQ2ZJ1MAKX58HKHSDTV2KJ3DKZE1B",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "stacks-testnet",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Verify(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response VerifyResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.IsValid)
	assert.NotNil(t, response.InvalidReason)
	assert.Contains(t, *response.InvalidReason, "recipient mismatch")
}

func TestHandler_Verify_AmountTooLow(t *testing.T) {
	mockBalance := &MockBalanceChecker{STXBalance: 2000000}
	handler := NewHandler(nil, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "stacks-testnet",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "500000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "stacks-testnet",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Verify(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response VerifyResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.IsValid)
	assert.NotNil(t, response.InvalidReason)
	assert.Contains(t, *response.InvalidReason, "amount too low")
}

func TestHandler_Verify_InvalidNetwork(t *testing.T) {
	mockBalance := &MockBalanceChecker{STXBalance: 2000000}
	handler := NewHandler(nil, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "invalid-network",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "invalid-network",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Verify(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response VerifyResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.IsValid)
	assert.NotNil(t, response.InvalidReason)
	assert.Contains(t, *response.InvalidReason, ErrorCodeInvalidNetwork)
}

func TestHandler_Verify_InvalidRequest(t *testing.T) {
	handler := NewHandler(nil, &MockBalanceChecker{})

	e := echo.New()
	reqBody := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Verify(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Settle_Success(t *testing.T) {
	mockSettle := &MockSettleHandler{
		HandleFn: func(ctx context.Context, cmd command.SettlePaymentCommand) (command.SettlePaymentResult, error) {
			return command.SettlePaymentResult{
				Success:          true,
				TxID:             "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				SenderAddress:    "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
				RecipientAddress: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
				Amount:           1000000,
				Fee:              180,
				Status:           "success",
				BlockHeight:      12345,
				TokenType:        "STX",
				Network:          "testnet",
			}, nil
		},
	}
	mockBalance := &MockBalanceChecker{STXBalance: 2000000}
	handler := NewHandler(mockSettle, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "stacks-testnet",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "stacks-testnet",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/settle", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Settle(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response SettleResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7", response.Payer)
	assert.Equal(t, "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", response.Transaction)
	assert.Equal(t, "stacks-testnet", response.Network)
	assert.Nil(t, response.ErrorReason)
}

func TestHandler_Settle_BroadcastError(t *testing.T) {
	mockSettle := &MockSettleHandler{
		HandleFn: func(ctx context.Context, cmd command.SettlePaymentCommand) (command.SettlePaymentResult, error) {
			return command.SettlePaymentResult{}, errors.New("broadcast failed: network error")
		},
	}
	mockBalance := &MockBalanceChecker{STXBalance: 2000000}
	handler := NewHandler(mockSettle, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "stacks-testnet",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "stacks-testnet",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/settle", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Settle(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response SettleResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotNil(t, response.ErrorReason)
	assert.Contains(t, *response.ErrorReason, ErrorCodeBroadcastFailed)
}

func TestHandler_Settle_VerificationFailed(t *testing.T) {
	mockSettle := &MockSettleHandler{
		HandleFn: func(ctx context.Context, cmd command.SettlePaymentCommand) (command.SettlePaymentResult, error) {
			return command.SettlePaymentResult{
				Success:          false,
				TxID:             "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				SenderAddress:    "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
				RecipientAddress: "ST3NBRSFKX28FQ2ZJ1MAKX58HKHSDTV2KJ3DKZE1B",
				Amount:           1000000,
				Status:           "success",
				Network:          "testnet",
				Errors:           []string{"recipient mismatch"},
			}, nil
		},
	}
	mockBalance := &MockBalanceChecker{STXBalance: 2000000}
	handler := NewHandler(mockSettle, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "stacks-testnet",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "stacks-testnet",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/settle", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Settle(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response SettleResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.NotNil(t, response.ErrorReason)
	assert.Equal(t, "recipient mismatch", *response.ErrorReason)
}

func TestHandler_Settle_InvalidNetwork(t *testing.T) {
	handler := NewHandler(nil, &MockBalanceChecker{})

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "invalid-network",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "invalid-network",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/settle", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Settle(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Settle_InvalidRequest(t *testing.T) {
	handler := NewHandler(nil, &MockBalanceChecker{})

	e := echo.New()
	reqBody := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/settle", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Settle(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestParseNetwork(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"stacks-mainnet", "mainnet", false},
		{"stacks-testnet", "testnet", false},
		{"mainnet", "mainnet", false},
		{"testnet", "testnet", false},
		{"STACKS-MAINNET", "mainnet", false},
		{"MAINNET", "mainnet", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			network, err := parseNetwork(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, network.String())
			}
		})
	}
}

func TestFormatNetworkForResponse(t *testing.T) {
	tests := []struct {
		input    valueobject.Network
		expected string
	}{
		{valueobject.NetworkMainnet, "stacks-mainnet"},
		{valueobject.NetworkTestnet, "stacks-testnet"},
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result := formatNetworkForResponse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseAsset(t *testing.T) {
	tests := []struct {
		asset         string
		expectedType  valueobject.TokenType
		expectedAsset string
	}{
		{"", valueobject.TokenSTX, ""},
		{"STX", valueobject.TokenSTX, ""},
		{"stx", valueobject.TokenSTX, ""},
		{"SM3VDXK3WZZSA84XXFKAFAF15NNZX32CTSG82JFQ4.sbtc-token", valueobject.TokenSBTC, "SM3VDXK3WZZSA84XXFKAFAF15NNZX32CTSG82JFQ4.sbtc-token"},
		{"SP123.usdc-token", valueobject.TokenUSDCX, "SP123.usdc-token"},
	}

	for _, tt := range tests {
		t.Run(tt.asset, func(t *testing.T) {
			tokenType, asset := parseAsset(tt.asset, valueobject.NetworkMainnet)
			assert.Equal(t, tt.expectedType, tokenType)
			assert.Equal(t, tt.expectedAsset, asset)
		})
	}
}

func TestHandler_Verify_MissingSignedTransaction(t *testing.T) {
	mockBalance := &MockBalanceChecker{STXBalance: 2000000}
	handler := NewHandler(nil, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "stacks-testnet",
			"payload": {
				"signedTransaction": "",
				"authorization": {
					"from": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "stacks-testnet",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Verify(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response VerifyResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.IsValid)
	assert.NotNil(t, response.InvalidReason)
	assert.Contains(t, *response.InvalidReason, "signed transaction")
}

func TestHandler_Verify_MissingAuthorization(t *testing.T) {
	mockBalance := &MockBalanceChecker{STXBalance: 2000000}
	handler := NewHandler(nil, mockBalance)

	e := echo.New()
	reqBody := `{
		"paymentPayload": {
			"x402Version": 1,
			"scheme": "exact",
			"network": "stacks-testnet",
			"payload": {
				"signedTransaction": "0x00000001deadbeef",
				"authorization": {
					"from": "",
					"to": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
					"value": "1000000"
				}
			}
		},
		"paymentRequirements": {
			"scheme": "exact",
			"network": "stacks-testnet",
			"maxAmountRequired": "1000000",
			"payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Verify(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response VerifyResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response.IsValid)
	assert.NotNil(t, response.InvalidReason)
	assert.Contains(t, *response.InvalidReason, "sender address")
}
