package http

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/x402stacks/stacks-facilitator/internal/payment/application/command"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

// VerifyPaymentHandler interface for verify payment use case (v1)
type VerifyPaymentHandler interface {
	Handle(ctx context.Context, cmd command.VerifyPaymentCommand) (command.VerifyPaymentResult, error)
}

// SettlePaymentHandler interface for settle payment use case (v1)
type SettlePaymentHandler interface {
	Handle(ctx context.Context, cmd command.SettlePaymentCommand) (command.SettlePaymentResult, error)
}

// VerifyPaymentHandlerV2 interface for verify payment use case (v2)
type VerifyPaymentHandlerV2 interface {
	Handle(ctx context.Context, cmd command.VerifyPaymentCommandV2) (command.VerifyPaymentResultV2, error)
}

// SettlePaymentHandlerV2 interface for settle payment use case (v2)
type SettlePaymentHandlerV2 interface {
	Handle(ctx context.Context, cmd command.SettlePaymentCommandV2) (command.SettlePaymentResultV2, error)
}

// Handler handles HTTP requests for payments
type Handler struct {
	verifyHandler   VerifyPaymentHandler
	settleHandler   SettlePaymentHandler
	verifyHandlerV2 VerifyPaymentHandlerV2
	settleHandlerV2 SettlePaymentHandlerV2
}

// NewHandler creates a new Handler
func NewHandler(verifyHandler VerifyPaymentHandler, settleHandler SettlePaymentHandler) *Handler {
	return &Handler{
		verifyHandler: verifyHandler,
		settleHandler: settleHandler,
	}
}

// NewHandlerWithV2 creates a new Handler with V2 support
func NewHandlerWithV2(
	verifyHandler VerifyPaymentHandler,
	settleHandler SettlePaymentHandler,
	verifyHandlerV2 VerifyPaymentHandlerV2,
	settleHandlerV2 SettlePaymentHandlerV2,
) *Handler {
	return &Handler{
		verifyHandler:   verifyHandler,
		settleHandler:   settleHandler,
		verifyHandlerV2: verifyHandlerV2,
		settleHandlerV2: settleHandlerV2,
	}
}

// VerifyV1 handles POST /api/v1/verify (legacy v1 format)
func (h *Handler) VerifyV1(c echo.Context) error {
	var req VerifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
	}

	// Validate required fields
	if req.TxID == "" || req.ExpectedRecipient == "" || req.Network == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_required_fields",
			Message: "tx_id, expected_recipient, and network are required",
		})
	}

	// Default token type to STX
	if req.TokenType == "" {
		req.TokenType = "STX"
	}

	cmd := command.VerifyPaymentCommand{
		TxID:              req.TxID,
		TokenType:         req.TokenType,
		ExpectedRecipient: req.ExpectedRecipient,
		MinAmount:         req.MinAmount,
		ExpectedSender:    req.ExpectedSender,
		ExpectedMemo:      req.ExpectedMemo,
		Network:           req.Network,
	}

	result, err := h.verifyHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "verification_failed",
			Message: err.Error(),
		})
	}

	response := VerifyResponse{
		Valid:            result.Valid,
		TxID:             result.TxID,
		SenderAddress:    result.SenderAddress,
		RecipientAddress: result.RecipientAddress,
		Amount:           result.Amount,
		Fee:              result.Fee,
		Nonce:            result.Nonce,
		Status:           result.Status,
		BlockHeight:      result.BlockHeight,
		TokenType:        result.TokenType,
		Memo:             result.Memo,
		Network:          result.Network,
		Errors:           result.Errors,
	}

	return c.JSON(http.StatusOK, response)
}

// SettleV1 handles POST /api/v1/settle (legacy v1 format)
func (h *Handler) SettleV1(c echo.Context) error {
	var req SettleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
	}

	// Validate required fields
	if req.SignedTransaction == "" || req.ExpectedRecipient == "" || req.Network == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_required_fields",
			Message: "signed_transaction, expected_recipient, and network are required",
		})
	}

	// Default token type to STX
	if req.TokenType == "" {
		req.TokenType = "STX"
	}

	cmd := command.SettlePaymentCommand{
		SignedTransaction: req.SignedTransaction,
		TokenType:         req.TokenType,
		ExpectedRecipient: req.ExpectedRecipient,
		MinAmount:         req.MinAmount,
		ExpectedSender:    req.ExpectedSender,
		Network:           req.Network,
	}

	result, err := h.settleHandler.Handle(c.Request().Context(), cmd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "settlement_failed",
			Message: err.Error(),
		})
	}

	response := SettleResponse{
		Success:          result.Success,
		TxID:             result.TxID,
		SenderAddress:    result.SenderAddress,
		RecipientAddress: result.RecipientAddress,
		Amount:           result.Amount,
		Fee:              result.Fee,
		Status:           result.Status,
		BlockHeight:      result.BlockHeight,
		TokenType:        result.TokenType,
		Network:          result.Network,
		Errors:           result.Errors,
	}

	if !result.Success {
		return c.JSON(http.StatusBadRequest, response)
	}

	return c.JSON(http.StatusOK, response)
}

// Health handles GET /health
func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}

// Verify handles POST /verify (x402 v2 Coinbase-compatible)
func (h *Handler) Verify(c echo.Context) error {
	var req VerifyRequestV2
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, VerifyResponseV2{
			IsValid:       false,
			InvalidReason: "invalid_request: " + err.Error(),
		})
	}

	// Check if V2 handler is available
	if h.verifyHandlerV2 == nil {
		return c.JSON(http.StatusNotImplemented, VerifyResponseV2{
			IsValid:       false,
			InvalidReason: "v2_not_implemented",
		})
	}

	// Validate required fields
	if req.PaymentRequirements.PayTo == "" || req.PaymentRequirements.Network == "" {
		return c.JSON(http.StatusBadRequest, VerifyResponseV2{
			IsValid:       false,
			InvalidReason: valueobject.ErrInvalidPaymentRequirements,
		})
	}

	// Build V2 command
	cmd := command.VerifyPaymentCommandV2{
		X402Version: req.X402Version,
		PaymentPayload: command.PaymentPayloadV2{
			X402Version: req.PaymentPayload.X402Version,
			Payload:     req.PaymentPayload.Payload,
			Accepted: command.PaymentRequirementsV2{
				Scheme:            req.PaymentPayload.Accepted.Scheme,
				Network:           req.PaymentPayload.Accepted.Network,
				Amount:            req.PaymentPayload.Accepted.Amount,
				Asset:             req.PaymentPayload.Accepted.Asset,
				PayTo:             req.PaymentPayload.Accepted.PayTo,
				MaxTimeoutSeconds: req.PaymentPayload.Accepted.MaxTimeoutSeconds,
				Extra:             req.PaymentPayload.Accepted.Extra,
			},
		},
		PaymentRequirements: command.PaymentRequirementsV2{
			Scheme:            req.PaymentRequirements.Scheme,
			Network:           req.PaymentRequirements.Network,
			Amount:            req.PaymentRequirements.Amount,
			Asset:             req.PaymentRequirements.Asset,
			PayTo:             req.PaymentRequirements.PayTo,
			MaxTimeoutSeconds: req.PaymentRequirements.MaxTimeoutSeconds,
			Extra:             req.PaymentRequirements.Extra,
		},
	}

	result, err := h.verifyHandlerV2.Handle(c.Request().Context(), cmd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, VerifyResponseV2{
			IsValid:       false,
			InvalidReason: valueobject.ErrUnexpectedVerifyError,
			Payer:         result.Payer,
		})
	}

	response := VerifyResponseV2{
		IsValid:       result.IsValid,
		InvalidReason: result.InvalidReason,
		Payer:         result.Payer,
	}

	if !result.IsValid {
		return c.JSON(http.StatusOK, response)
	}

	return c.JSON(http.StatusOK, response)
}

// Settle handles POST /settle (x402 v2 Coinbase-compatible)
func (h *Handler) Settle(c echo.Context) error {
	var req SettleRequestV2
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, SettleResponseV2{
			Success:     false,
			ErrorReason: "invalid_request: " + err.Error(),
			Transaction: "",
			Network:     "",
		})
	}

	// Check if V2 handler is available
	if h.settleHandlerV2 == nil {
		return c.JSON(http.StatusNotImplemented, SettleResponseV2{
			Success:     false,
			ErrorReason: "v2_not_implemented",
			Transaction: "",
			Network:     req.PaymentRequirements.Network,
		})
	}

	// Validate required fields
	if req.PaymentRequirements.PayTo == "" || req.PaymentRequirements.Network == "" {
		return c.JSON(http.StatusBadRequest, SettleResponseV2{
			Success:     false,
			ErrorReason: valueobject.ErrInvalidPaymentRequirements,
			Transaction: "",
			Network:     req.PaymentRequirements.Network,
		})
	}

	// Build V2 command
	cmd := command.SettlePaymentCommandV2{
		X402Version: req.X402Version,
		PaymentPayload: command.PaymentPayloadV2{
			X402Version: req.PaymentPayload.X402Version,
			Payload:     req.PaymentPayload.Payload,
			Accepted: command.PaymentRequirementsV2{
				Scheme:            req.PaymentPayload.Accepted.Scheme,
				Network:           req.PaymentPayload.Accepted.Network,
				Amount:            req.PaymentPayload.Accepted.Amount,
				Asset:             req.PaymentPayload.Accepted.Asset,
				PayTo:             req.PaymentPayload.Accepted.PayTo,
				MaxTimeoutSeconds: req.PaymentPayload.Accepted.MaxTimeoutSeconds,
				Extra:             req.PaymentPayload.Accepted.Extra,
			},
		},
		PaymentRequirements: command.PaymentRequirementsV2{
			Scheme:            req.PaymentRequirements.Scheme,
			Network:           req.PaymentRequirements.Network,
			Amount:            req.PaymentRequirements.Amount,
			Asset:             req.PaymentRequirements.Asset,
			PayTo:             req.PaymentRequirements.PayTo,
			MaxTimeoutSeconds: req.PaymentRequirements.MaxTimeoutSeconds,
			Extra:             req.PaymentRequirements.Extra,
		},
	}

	result, err := h.settleHandlerV2.Handle(c.Request().Context(), cmd)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, SettleResponseV2{
			Success:     false,
			ErrorReason: valueobject.ErrUnexpectedSettleError,
			Payer:       result.Payer,
			Transaction: result.Transaction,
			Network:     result.Network,
		})
	}

	response := SettleResponseV2{
		Success:     result.Success,
		ErrorReason: result.ErrorReason,
		Payer:       result.Payer,
		Transaction: result.Transaction,
		Network:     result.Network,
	}

	if !result.Success {
		return c.JSON(http.StatusBadRequest, response)
	}

	return c.JSON(http.StatusOK, response)
}

// Supported handles GET /supported (x402 v2 Coinbase-compatible)
func (h *Handler) Supported(c echo.Context) error {
	response := SupportedResponse{
		Kinds: []SupportedKind{
			{
				X402Version: 2,
				Scheme:      "exact",
				Network:     valueobject.StacksMainnetCAIP2,
			},
			{
				X402Version: 2,
				Scheme:      "exact",
				Network:     valueobject.StacksTestnetCAIP2,
			},
		},
		Extensions: []string{},
		Signers: map[string][]string{
			"stacks:*": {}, // Empty for now - would contain sponsor addresses if sponsoring enabled
		},
	}

	return c.JSON(http.StatusOK, response)
}

// RegisterRoutes registers the HTTP routes
func (h *Handler) RegisterRoutes(e *echo.Echo) {
	// V1 routes (legacy - for backward compatibility with existing Stacks SDK clients)
	apiV1 := e.Group("/api/v1")
	apiV1.POST("/verify", h.VerifyV1)
	apiV1.POST("/settle", h.SettleV1)

	// V2 routes (Coinbase x402 compatible - at root level)
	e.POST("/verify", h.Verify)
	e.POST("/settle", h.Settle)
	e.GET("/supported", h.Supported)

	// Health check
	e.GET("/health", h.Health)
}
