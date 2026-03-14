package blockchain

import (
	"context"
	"log"
	"time"

	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/service"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
	"github.com/x402stacks/stacks-facilitator/internal/stacks"
)

// StacksClientAdapter adapts the Stacks client for use in the domain layer
type StacksClientAdapter struct {
	mainnetClient *stacks.Client
	testnetClient *stacks.Client
}

// NewStacksClientAdapter creates a new StacksClientAdapter
func NewStacksClientAdapter() *StacksClientAdapter {
	return &StacksClientAdapter{
		mainnetClient: stacks.NewClientForNetwork(valueobject.NetworkMainnet),
		testnetClient: stacks.NewClientForNetwork(valueobject.NetworkTestnet),
	}
}

// GetTransaction fetches a transaction from the blockchain
func (a *StacksClientAdapter) GetTransaction(ctx context.Context, txID valueobject.TransactionID, tokenType valueobject.TokenType, network valueobject.Network) (service.BlockchainTransaction, error) {
	client := a.getClientForNetwork(network)
	return client.GetTransactionWithTokenType(ctx, txID, tokenType, network)
}

// GetTransactionWithRetry fetches a transaction with retry logic
func (a *StacksClientAdapter) GetTransactionWithRetry(ctx context.Context, txID valueobject.TransactionID, tokenType valueobject.TokenType, network valueobject.Network, maxRetries int, retryDelay time.Duration) (service.BlockchainTransaction, error) {
	client := a.getClientForNetwork(network)

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		tx, err := client.GetTransactionWithTokenType(ctx, txID, tokenType, network)
		if err == nil {
			return tx, nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return service.BlockchainTransaction{}, ctx.Err()
		case <-time.After(retryDelay):
		}
	}

	return service.BlockchainTransaction{}, lastErr
}

// WaitForConfirmation waits for a transaction to be confirmed
func (a *StacksClientAdapter) WaitForConfirmation(ctx context.Context, txID valueobject.TransactionID, tokenType valueobject.TokenType, network valueobject.Network, maxRetries int, retryDelay time.Duration) (service.BlockchainTransaction, error) {
	client := a.getClientForNetwork(network)

	for i := 0; i < maxRetries; i++ {
		log.Printf("[Adapter] Polling for confirmation: tx=%s attempt=%d/%d", txID.String(), i+1, maxRetries)
		tx, err := client.GetTransactionWithTokenType(ctx, txID, tokenType, network)
		if err == nil {
			if tx.IsConfirmed {
				log.Printf("[Adapter] Transaction confirmed: tx=%s status=%s block=%d", txID.String(), tx.Status, tx.BlockHeight)
				return tx, nil
			}
			if stacks.IsTransactionFailed(tx.Status) {
				log.Printf("[Adapter] Transaction failed on chain: tx=%s status=%s", txID.String(), tx.Status)
				return tx, nil
			}
			log.Printf("[Adapter] Transaction not yet confirmed: tx=%s status=%s", txID.String(), tx.Status)
		} else {
			log.Printf("[Adapter] Error fetching transaction: tx=%s attempt=%d error=%v", txID.String(), i+1, err)
		}

		select {
		case <-ctx.Done():
			log.Printf("[Adapter] Context cancelled while waiting for confirmation: tx=%s", txID.String())
			return service.BlockchainTransaction{}, ctx.Err()
		case <-time.After(retryDelay):
		}
	}

	log.Printf("[Adapter] Max retries reached, fetching final state: tx=%s", txID.String())
	// Return last fetched transaction even if not confirmed
	return client.GetTransactionWithTokenType(ctx, txID, tokenType, network)
}

// BroadcastTransaction broadcasts a signed transaction
func (a *StacksClientAdapter) BroadcastTransaction(ctx context.Context, signedTx string, network valueobject.Network) (valueobject.TransactionID, error) {
	log.Printf("[Adapter] Broadcasting transaction on %s (hex length=%d)", network, len(signedTx))
	client := a.getClientForNetwork(network)
	txID, err := client.BroadcastTransaction(ctx, signedTx)
	if err != nil {
		log.Printf("[Adapter] Broadcast failed on %s: %v", network, err)
		return txID, err
	}
	log.Printf("[Adapter] Broadcast successful: tx=%s network=%s", txID.String(), network)
	return txID, nil
}

// getClientForNetwork returns the appropriate client for the network
func (a *StacksClientAdapter) getClientForNetwork(network valueobject.Network) *stacks.Client {
	if network.IsMainnet() {
		return a.mainnetClient
	}
	return a.testnetClient
}
