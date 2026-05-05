package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/x402stacks/stacks-facilitator/internal/payment/application/command"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/service"
	"github.com/x402stacks/stacks-facilitator/internal/payment/infrastructure/blockchain"
	httpHandler "github.com/x402stacks/stacks-facilitator/internal/payment/infrastructure/http"
)

func main() {
	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	hiroAPIKey := os.Getenv("HIRO_API_KEY")

	// Initialize infrastructure
	stacksAdapter := blockchain.NewStacksClientAdapterWithAPIKey(hiroAPIKey)

	// Initialize domain services
	verificationService := service.NewVerificationService()

	// Initialize V1 application handlers
	verifyHandler := command.NewVerifyPaymentHandler(stacksAdapter, verificationService)
	settleHandler := command.NewSettlePaymentHandler(stacksAdapter, verificationService)

	// Initialize V2 application handlers (Coinbase-compatible)
	verifyHandlerV2 := command.NewVerifyPaymentHandlerV2(stacksAdapter, verificationService)
	settleHandlerV2 := command.NewSettlePaymentHandlerV2(stacksAdapter, verificationService)

	// Initialize HTTP handler with V2 support
	handler := httpHandler.NewHandlerWithV2(verifyHandler, settleHandler, verifyHandlerV2, settleHandlerV2)

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RequestID())

	// Register routes
	handler.RegisterRoutes(e)

	// Start server
	log.Printf("Starting server on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
