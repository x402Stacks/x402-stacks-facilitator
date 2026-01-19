package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/x402stacks/stacks-facilitator/internal/payment/application/command"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/service"
	"github.com/x402stacks/stacks-facilitator/internal/payment/infrastructure/blockchain"
	"github.com/x402stacks/stacks-facilitator/internal/payment/infrastructure/http"
	"github.com/x402stacks/stacks-facilitator/internal/payment/infrastructure/http/coinbase"
)

func main() {
	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Create infrastructure
	stacksAdapter := blockchain.NewStacksClientAdapter()
	verificationSvc := service.NewVerificationService()

	// Create command handlers
	verifyHandler := command.NewVerifyPaymentHandler(stacksAdapter, verificationSvc)
	settleHandler := command.NewSettlePaymentHandler(stacksAdapter, verificationSvc)

	// Register original API routes (/api/v1/*)
	handler := http.NewHandler(verifyHandler, settleHandler)
	handler.RegisterRoutes(e)

	// Register Coinbase-compatible routes (/, /verify, /settle, /supported)
	coinbaseHandler := coinbase.NewHandler(settleHandler, stacksAdapter)
	coinbaseHandler.RegisterRoutes(e)

	// Get port from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Starting server on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
