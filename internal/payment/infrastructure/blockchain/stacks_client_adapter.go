package blockchain

import (
	"context"
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
		tx, err := client.GetTransactionWithTokenType(ctx, txID, tokenType, network)
		if err == nil {
			if tx.IsConfirmed || stacks.IsTransactionFailed(tx.Status) {
				return tx, nil
			}
		}

		select {
		case <-ctx.Done():
			return service.BlockchainTransaction{}, ctx.Err()
		case <-time.After(retryDelay):
		}
	}

	// Return last fetched transaction even if not confirmed
	return client.GetTransactionWithTokenType(ctx, txID, tokenType, network)
}

// BroadcastTransaction broadcasts a signed transaction
func (a *StacksClientAdapter) BroadcastTransaction(ctx context.Context, signedTx string, network valueobject.Network) (valueobject.TransactionID, error) {
	client := a.getClientForNetwork(network)
	return client.BroadcastTransaction(ctx, signedTx)
}

// getClientForNetwork returns the appropriate client for the network
func (a *StacksClientAdapter) getClientForNetwork(network valueobject.Network) *stacks.Client {
	if network.IsMainnet() {
		return a.mainnetClient
	}
	return a.testnetClient
}

// GetSTXBalance returns the STX balance for an address on the specified network
func (a *StacksClientAdapter) GetSTXBalance(ctx context.Context, address string, network valueobject.Network) (uint64, error) {
	client := a.getClientForNetwork(network)
	return client.GetSTXBalance(ctx, address)
}

// GetTokenBalance returns the token balance for an address on the specified network
func (a *StacksClientAdapter) GetTokenBalance(ctx context.Context, address string, contractID string, network valueobject.Network) (uint64, error) {
	client := a.getClientForNetwork(network)
	return client.GetTokenBalance(ctx, address, contractID)
}
