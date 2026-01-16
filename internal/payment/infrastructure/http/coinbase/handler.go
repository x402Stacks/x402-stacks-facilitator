package coinbase

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/x402stacks/stacks-facilitator/internal/payment/application/command"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

// SettlePaymentHandler interface for settlement operations
type SettlePaymentHandler interface {
	Handle(ctx context.Context, cmd command.SettlePaymentCommand) (command.SettlePaymentResult, error)
}

// Handler handles Coinbase-format HTTP requests
type Handler struct {
	settleHandler SettlePaymentHandler
	verifier      *PreSubmitVerifier
}

// NewHandler creates a new Coinbase-format Handler
func NewHandler(settleHandler SettlePaymentHandler, balanceChecker BalanceChecker) *Handler {
	return &Handler{
		settleHandler: settleHandler,
		verifier:      NewPreSubmitVerifier(balanceChecker),
	}
}

// Verify handles POST /verify (Coinbase format)
// Validates signed transaction WITHOUT broadcasting
func (h *Handler) Verify(c echo.Context) error {
	var req VerifyRequest
	if err := c.Bind(&req); err != nil {
		reason := "invalid request body: " + err.Error()
		return c.JSON(http.StatusBadRequest, VerifyResponse{
			IsValid:       false,
			InvalidReason: &reason,
		})
	}

	// Parse network
	network, err := parseNetwork(req.PaymentPayload.Network)
	if err != nil {
		reason := ErrorCodeInvalidNetwork + ": " + err.Error()
		return c.JSON(http.StatusBadRequest, VerifyResponse{
			IsValid:       false,
			InvalidReason: &reason,
		})
	}

	// Parse token type from asset
	tokenType, asset := parseAsset(req.PaymentRequirements.Asset, network)

	// Parse minimum amount
	minAmount, err := strconv.ParseUint(req.PaymentRequirements.MaxAmountRequired, 10, 64)
	if err != nil {
		reason := "invalid maxAmountRequired: " + err.Error()
		return c.JSON(http.StatusBadRequest, VerifyResponse{
			IsValid:       false,
			InvalidReason: &reason,
		})
	}

	// Build verification requirements
	requirements := VerifyRequirements{
		ExpectedRecipient: req.PaymentRequirements.PayTo,
		MinAmount:         minAmount,
		Network:           network,
		TokenType:         tokenType,
		Asset:             asset,
	}

	// Run pre-submit verification
	result := h.verifier.Verify(c.Request().Context(), req.PaymentPayload.Payload, requirements)

	response := VerifyResponse{
		IsValid: result.Valid,
		Payer:   result.Payer,
	}

	if !result.Valid && len(result.Errors) > 0 {
		errorMsg := result.Errors[0]
		response.InvalidReason = &errorMsg
	}

	return c.JSON(http.StatusOK, response)
}

// Settle handles POST /settle (Coinbase format)
// Broadcasts signed transaction and returns result
func (h *Handler) Settle(c echo.Context) error {
	var req SettleRequest
	if err := c.Bind(&req); err != nil {
		reason := "invalid request body: " + err.Error()
		return c.JSON(http.StatusBadRequest, SettleResponse{
			Success:     false,
			ErrorReason: &reason,
		})
	}

	// Parse network
	network, err := parseNetwork(req.PaymentPayload.Network)
	if err != nil {
		reason := ErrorCodeInvalidNetwork + ": " + err.Error()
		return c.JSON(http.StatusBadRequest, SettleResponse{
			Success:     false,
			ErrorReason: &reason,
		})
	}

	// Parse token type
	tokenType, _ := parseAsset(req.PaymentRequirements.Asset, network)

	// Parse minimum amount
	minAmount, err := strconv.ParseUint(req.PaymentRequirements.MaxAmountRequired, 10, 64)
	if err != nil {
		reason := "invalid maxAmountRequired: " + err.Error()
		return c.JSON(http.StatusBadRequest, SettleResponse{
			Success:     false,
			ErrorReason: &reason,
		})
	}

	// Build settle command for existing handler
	cmd := command.SettlePaymentCommand{
		SignedTransaction: req.PaymentPayload.Payload.SignedTransaction,
		TokenType:         tokenType.String(),
		ExpectedRecipient: req.PaymentRequirements.PayTo,
		MinAmount:         minAmount,
		ExpectedSender:    stringPtrOrNil(req.PaymentPayload.Payload.Authorization.From),
		Network:           network.String(),
	}

	// Execute settlement
	result, err := h.settleHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		reason := ErrorCodeBroadcastFailed + ": " + err.Error()
		return c.JSON(http.StatusInternalServerError, SettleResponse{
			Success:     false,
			Network:     formatNetworkForResponse(network),
			ErrorReason: &reason,
		})
	}

	response := SettleResponse{
		Success:     result.Success,
		Payer:       result.SenderAddress,
		Transaction: result.TxID,
		Network:     formatNetworkForResponse(network),
	}

	if !result.Success {
		errorMsg := ErrorCodeConfirmationFailed
		if len(result.Errors) > 0 {
			errorMsg = result.Errors[0]
		}
		response.ErrorReason = &errorMsg
	}

	if result.Success {
		return c.JSON(http.StatusOK, response)
	}
	return c.JSON(http.StatusBadRequest, response)
}

// Supported handles GET /supported
// Returns supported payment kinds and capabilities
func (h *Handler) Supported(c echo.Context) error {
	response := SupportedResponse{
		Kinds: []SupportedKind{
			{
				X402Version: 1,
				Scheme:      "exact",
				Network:     "stacks-mainnet",
			},
			{
				X402Version: 1,
				Scheme:      "exact",
				Network:     "stacks-testnet",
			},
		},
		Extensions: []string{},
		Signers:    map[string]string{},
	}

	return c.JSON(http.StatusOK, response)
}

// RegisterRoutes registers the Coinbase-format routes at root level
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.POST("/verify", h.Verify)
	e.POST("/settle", h.Settle)
	e.GET("/supported", h.Supported)
}

// parseNetwork converts Coinbase network format to internal Network
// Coinbase format: "stacks-mainnet", "stacks-testnet"
// Internal format: "mainnet", "testnet"
func parseNetwork(network string) (valueobject.Network, error) {
	normalized := strings.ToLower(network)

	// Handle Coinbase format
	if strings.HasPrefix(normalized, "stacks-") {
		normalized = strings.TrimPrefix(normalized, "stacks-")
	}

	return valueobject.NewNetwork(normalized)
}

// formatNetworkForResponse converts internal network to Coinbase format
func formatNetworkForResponse(network valueobject.Network) string {
	return "stacks-" + network.String()
}

// parseAsset converts asset identifier to TokenType and contract ID
// Returns (tokenType, contractID)
func parseAsset(asset string, network valueobject.Network) (valueobject.TokenType, string) {
	if asset == "" || strings.ToUpper(asset) == "STX" {
		return valueobject.TokenSTX, ""
	}

	// Check for sBTC contract
	if strings.Contains(strings.ToLower(asset), "sbtc") {
		return valueobject.TokenSBTC, asset
	}

	// Check for USDCx contract
	if strings.Contains(strings.ToLower(asset), "usdc") {
		return valueobject.TokenUSDCX, asset
	}

	// Unknown asset - treat as SIP-010 token, default to STX behavior
	// The contract ID will be used for balance checks
	return valueobject.TokenSTX, asset
}

// stringPtrOrNil returns a pointer to s if non-empty, otherwise nil
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
