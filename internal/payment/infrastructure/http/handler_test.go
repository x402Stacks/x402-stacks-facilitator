package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/x402stacks/stacks-facilitator/internal/payment/application/command"
)

// MockVerifyHandler for testing
type MockVerifyHandler struct {
	HandleFn func(ctx context.Context, cmd command.VerifyPaymentCommand) (command.VerifyPaymentResult, error)
}

func (m *MockVerifyHandler) Handle(ctx context.Context, cmd command.VerifyPaymentCommand) (command.VerifyPaymentResult, error) {
	return m.HandleFn(ctx, cmd)
}

// MockSettleHandler for testing
type MockSettleHandler struct {
	HandleFn func(ctx context.Context, cmd command.SettlePaymentCommand) (command.SettlePaymentResult, error)
}

func (m *MockSettleHandler) Handle(ctx context.Context, cmd command.SettlePaymentCommand) (command.SettlePaymentResult, error) {
	return m.HandleFn(ctx, cmd)
}

func TestHandler_Verify_Success(t *testing.T) {
	mockVerify := &MockVerifyHandler{
		HandleFn: func(ctx context.Context, cmd command.VerifyPaymentCommand) (command.VerifyPaymentResult, error) {
			return command.VerifyPaymentResult{
				Valid:            true,
				TxID:             "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				SenderAddress:    "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
				RecipientAddress: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
				Amount:           1000000,
				Fee:              180,
				Status:           "confirmed",
				BlockHeight:      12345,
				TokenType:        "STX",
				Network:          "testnet",
			}, nil
		},
	}

	handler := NewHandler(mockVerify, nil)

	e := echo.New()
	reqBody := `{
		"tx_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		"expected_recipient": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		"min_amount": 500000,
		"network": "testnet"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.VerifyV1(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response VerifyResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Valid)
	assert.Equal(t, "confirmed", response.Status)
}

func TestHandler_Verify_InvalidRequest(t *testing.T) {
	handler := NewHandler(nil, nil)

	e := echo.New()
	reqBody := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.VerifyV1(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Verify_MissingRequiredFields(t *testing.T) {
	handler := NewHandler(nil, nil)

	e := echo.New()
	reqBody := `{"tx_id": "0x1234"}` // Missing required fields
	req := httptest.NewRequest(http.MethodPost, "/api/v1/verify", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.VerifyV1(c)

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
				Status:           "confirmed",
				BlockHeight:      12345,
				TokenType:        "STX",
				Network:          "testnet",
			}, nil
		},
	}

	handler := NewHandler(nil, mockSettle)

	e := echo.New()
	reqBody := `{
		"signed_transaction": "0x00000001deadbeef",
		"expected_recipient": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
		"min_amount": 500000,
		"network": "testnet"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settle", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.SettleV1(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response SettleResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "confirmed", response.Status)
}

func TestHandler_Settle_InvalidRequest(t *testing.T) {
	handler := NewHandler(nil, nil)

	e := echo.New()
	reqBody := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settle", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.SettleV1(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Settle_MissingRequiredFields(t *testing.T) {
	handler := NewHandler(nil, nil)

	e := echo.New()
	reqBody := `{"signed_transaction": "0x1234"}` // Missing required fields
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settle", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.SettleV1(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_Health(t *testing.T) {
	handler := NewHandler(nil, nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.Health(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
