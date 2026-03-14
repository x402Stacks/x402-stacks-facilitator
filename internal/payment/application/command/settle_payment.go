package command

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/service"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

// Settle retry defaults: 25 retries × 3s = 75s window.
// sBTC/SIP-010 contract calls can take 60+ seconds to confirm.
const (
	DefaultSettleMaxRetries = 25
	DefaultSettleRetryDelay = 3 * time.Second
)

// TransactionBroadcaster interface for broadcasting and confirming transactions
type TransactionBroadcaster interface {
	BroadcastTransaction(ctx context.Context, signedTx string, network valueobject.Network) (valueobject.TransactionID, error)
	WaitForConfirmation(ctx context.Context, txID valueobject.TransactionID, tokenType valueobject.TokenType, network valueobject.Network, maxRetries int, retryDelay time.Duration) (service.BlockchainTransaction, error)
}

// SettlePaymentCommand represents a request to settle a payment
type SettlePaymentCommand struct {
	SignedTransaction string
	TokenType         string
	ExpectedRecipient string
	MinAmount         uint64
	ExpectedSender    *string
	Network           string
}

// SettlePaymentResult represents the result of a settlement
type SettlePaymentResult struct {
	Success          bool
	TxID             string
	SenderAddress    string
	RecipientAddress string
	Amount           uint64
	Fee              uint64
	Status           string
	BlockHeight      uint64
	TokenType        string
	Network          string
	Errors           []string
}

// SettlePaymentHandler handles settle payment commands
type SettlePaymentHandler struct {
	broadcaster     TransactionBroadcaster
	verificationSvc *service.VerificationService
	maxRetries      int
	retryDelay      time.Duration
}

// NewSettlePaymentHandler creates a new SettlePaymentHandler
func NewSettlePaymentHandler(broadcaster TransactionBroadcaster, verificationSvc *service.VerificationService) *SettlePaymentHandler {
	return &SettlePaymentHandler{
		broadcaster:     broadcaster,
		verificationSvc: verificationSvc,
		maxRetries:      DefaultSettleMaxRetries,
		retryDelay:      DefaultSettleRetryDelay,
	}
}

// Handle processes the settle payment command
func (h *SettlePaymentHandler) Handle(ctx context.Context, cmd SettlePaymentCommand) (SettlePaymentResult, error) {
	log.Printf("[Settle] Processing settle command: network=%s recipient=%s token=%s amount=%d",
		cmd.Network, cmd.ExpectedRecipient, cmd.TokenType, cmd.MinAmount)

	// Parse and validate inputs
	tokenType, err := valueobject.NewTokenType(cmd.TokenType)
	if err != nil {
		log.Printf("[Settle] Unknown token type %q, defaulting to STX", cmd.TokenType)
		tokenType = valueobject.TokenSTX // Default to STX
	}

	network, err := valueobject.NewNetwork(cmd.Network)
	if err != nil {
		log.Printf("[Settle] Invalid network %q: %v", cmd.Network, err)
		return SettlePaymentResult{}, fmt.Errorf("invalid network: %w", err)
	}

	expectedRecipient, err := valueobject.NewStacksAddress(cmd.ExpectedRecipient)
	if err != nil {
		log.Printf("[Settle] Invalid recipient address %q: %v", cmd.ExpectedRecipient, err)
		return SettlePaymentResult{}, fmt.Errorf("invalid expected recipient: %w", err)
	}

	// Broadcast the transaction
	log.Printf("[Settle] Broadcasting transaction on %s", network)
	txID, err := h.broadcaster.BroadcastTransaction(ctx, cmd.SignedTransaction, network)
	if err != nil {
		log.Printf("[Settle] Broadcast failed: %v", err)
		return SettlePaymentResult{}, fmt.Errorf("failed to broadcast transaction: %w", err)
	}
	log.Printf("[Settle] Transaction broadcast successful: tx=%s", txID.String())

	// Wait for transaction to be confirmed
	log.Printf("[Settle] Waiting for confirmation: tx=%s maxRetries=%d retryDelay=%s", txID.String(), h.maxRetries, h.retryDelay)
	tx, err := h.broadcaster.WaitForConfirmation(ctx, txID, tokenType, network, h.maxRetries, h.retryDelay)
	if err != nil {
		log.Printf("[Settle] Confirmation failed: tx=%s error=%v", txID.String(), err)
		return SettlePaymentResult{}, fmt.Errorf("failed to confirm transaction: %w", err)
	}
	log.Printf("[Settle] Transaction confirmed: tx=%s status=%s block=%d", tx.TxID.String(), tx.Status, tx.BlockHeight)

	// Build verification criteria (always require confirmation for settlement)
	criteria := service.VerificationCriteria{
		ExpectedRecipient: expectedRecipient,
		MinAmount:         valueobject.NewAmount(cmd.MinAmount),
		AcceptUnconfirmed: false,
	}

	// Optional sender
	if cmd.ExpectedSender != nil {
		sender, err := valueobject.NewStacksAddress(*cmd.ExpectedSender)
		if err == nil {
			criteria.ExpectedSender = &sender
			log.Printf("[Settle] Verifying with expected sender: %s", sender.String())
		}
	}

	// Verify transaction
	log.Printf("[Settle] Verifying transaction against criteria: recipient=%s minAmount=%d",
		expectedRecipient.String(), cmd.MinAmount)
	verificationResult := h.verificationSvc.Verify(tx, criteria)

	// Determine status
	status := determinePaymentStatus(tx)

	if !verificationResult.Valid {
		log.Printf("[Settle] Verification failed: tx=%s errors=%v", tx.TxID.String(), verificationResult.Errors)
	} else {
		log.Printf("[Settle] Verification passed: tx=%s sender=%s recipient=%s amount=%d fee=%d",
			tx.TxID.String(), tx.Sender.String(), tx.Recipient.String(), tx.Amount.Value(), tx.Fee.Value())
	}

	return SettlePaymentResult{
		Success:          verificationResult.Valid,
		TxID:             tx.TxID.String(),
		SenderAddress:    tx.Sender.String(),
		RecipientAddress: tx.Recipient.String(),
		Amount:           tx.Amount.Value(),
		Fee:              tx.Fee.Value(),
		Status:           status,
		BlockHeight:      tx.BlockHeight,
		TokenType:        tx.TokenType.String(),
		Network:          network.String(),
		Errors:           verificationResult.Errors,
	}, nil
}
