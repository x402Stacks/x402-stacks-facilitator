package stacks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/x402stacks/stacks-facilitator/internal/payment/domain/valueobject"
)

func TestClient_GetTransaction_STXTransfer(t *testing.T) {
	// Mock server returning an STX transfer transaction
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/extended/v1/tx/0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", r.URL.Path)

		response := TransactionResponse{
			TxID:          "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			TxStatus:      "success",
			TxType:        "token_transfer",
			BlockHeight:   12345,
			Fee:           "180",
			Nonce:         5,
			SenderAddress: "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			TokenTransfer: &TokenTransferData{
				RecipientAddress: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
				Amount:           "1000000",
				Memo:             "test payment",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	txID, _ := valueobject.NewTransactionID("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	tx, err := client.GetTransaction(ctx, txID)

	require.NoError(t, err)
	assert.Equal(t, txID.String(), tx.TxID.String())
	assert.Equal(t, "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7", tx.Sender.String())
	assert.Equal(t, "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM", tx.Recipient.String())
	assert.Equal(t, uint64(1000000), tx.Amount.Value())
	assert.Equal(t, uint64(180), tx.Fee.Value())
	assert.Equal(t, uint64(12345), tx.BlockHeight)
	assert.Equal(t, "test payment", tx.Memo)
	assert.True(t, tx.IsConfirmed)
	assert.Equal(t, valueobject.TokenSTX, tx.TokenType)
}

func TestClient_GetTransaction_SendsAPIKeyWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-hiro-key", r.Header.Get("x-api-key"))

		response := TransactionResponse{
			TxID:          "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			TxStatus:      "success",
			TxType:        "token_transfer",
			BlockHeight:   12345,
			Fee:           "180",
			Nonce:         5,
			SenderAddress: "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			TokenTransfer: &TokenTransferData{
				RecipientAddress: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
				Amount:           "1000000",
				Memo:             "test payment",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClientWithAPIKey(server.URL, "test-hiro-key")
	ctx := context.Background()
	txID, _ := valueobject.NewTransactionID("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	_, err := client.GetTransaction(ctx, txID)

	require.NoError(t, err)
}

func TestClient_GetTransaction_PendingTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TransactionResponse{
			TxID:          "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			TxStatus:      "pending",
			TxType:        "token_transfer",
			BlockHeight:   0,
			Fee:           "180",
			Nonce:         5,
			SenderAddress: "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			TokenTransfer: &TokenTransferData{
				RecipientAddress: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
				Amount:           "1000000",
				Memo:             "",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	txID, _ := valueobject.NewTransactionID("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	tx, err := client.GetTransaction(ctx, txID)

	require.NoError(t, err)
	assert.False(t, tx.IsConfirmed)
	assert.Equal(t, "pending", tx.Status)
}

func TestClient_GetTransaction_FailedTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := TransactionResponse{
			TxID:          "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			TxStatus:      "abort_by_response",
			TxType:        "token_transfer",
			BlockHeight:   12345,
			Fee:           "180",
			Nonce:         5,
			SenderAddress: "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
			TokenTransfer: &TokenTransferData{
				RecipientAddress: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
				Amount:           "1000000",
				Memo:             "",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	txID, _ := valueobject.NewTransactionID("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	tx, err := client.GetTransaction(ctx, txID)

	require.NoError(t, err)
	assert.Equal(t, "abort_by_response", tx.Status)
}

func TestClient_GetTransaction_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	txID, _ := valueobject.NewTransactionID("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	_, err := client.GetTransaction(ctx, txID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestClient_BroadcastTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/transactions", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))

		// Return transaction ID
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	txID, err := client.BroadcastTransaction(ctx, "0x00000001deadbeef")

	require.NoError(t, err)
	assert.Equal(t, "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", txID.String())
}

func TestClient_BroadcastTransaction_SendsAPIKeyWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-hiro-key", r.Header.Get("x-api-key"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`"0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"`))
	}))
	defer server.Close()

	client := NewClientWithAPIKey(server.URL, "test-hiro-key")
	ctx := context.Background()

	_, err := client.BroadcastTransaction(ctx, "0x00000001deadbeef")

	require.NoError(t, err)
}

func TestClient_BroadcastTransaction_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid transaction"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	_, err := client.BroadcastTransaction(ctx, "invalid")

	assert.Error(t, err)
}

func TestClient_IsTransactionConfirmed(t *testing.T) {
	assert.True(t, IsTransactionConfirmed("success", 12345))
	assert.False(t, IsTransactionConfirmed("success", 0))
	assert.False(t, IsTransactionConfirmed("pending", 0))
	assert.False(t, IsTransactionConfirmed("failed", 12345))
}

func TestClient_IsTransactionFailed(t *testing.T) {
	assert.False(t, IsTransactionFailed("success"))
	assert.False(t, IsTransactionFailed("pending"))
	assert.True(t, IsTransactionFailed("failed"))
	assert.True(t, IsTransactionFailed("abort_by_response"))
	assert.True(t, IsTransactionFailed("abort_by_post_condition"))
}
