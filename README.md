# Stacks Facilitator

A stateless payment verification and settlement service for the Stacks blockchain, implementing the x402 protocol. Built with Go, following Domain-Driven Design (DDD) principles.

## Features

-**Verify** existing blockchain transactions against specified criteria
- **Settle** payments by broadcasting signed transactions and confirming them on-chain
- **Multi-token support**: STX, sBTC, USDCx
- **Multi-network support**: Mainnet and Testnet
- **Stateless**: No database required
- **Retry logic**: Built-in retry mechanism for blockchain operations
- **x402 v2 support**: Coinbase-compatible payment protocol endpoints

## Requirements

- Go 1.24+ (for local development)
- Docker (for containerized deployment)

## Quick Start

### Run with Docker (Recommended)

```bash
# Build and run
docker-compose up -d

# Check health
curl http://localhost:8089/health
```

### Run Locally

```bash
# Install dependencies
go mod download

# Run the server
go run ./cmd/server/main.go

# Or build and run
go build -o server ./cmd/server
./server
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server listening port |

### Network Configuration

The facilitator connects to Hiro API endpoints automatically based on the network specified in each request:

| Network | API Endpoint |
|---------|--------------|
| `mainnet` | `https://api.mainnet.hiro.so` |
| `testnet` | `https://api.testnet.hiro.so` |

These endpoints are hardcoded and cannot be configured via environment variables.

---

## API Reference

### Base URL

```
http://localhost:8080
```

### Authentication

No authentication required. The service is designed to run behind a reverse proxy or API gateway that handles authentication.

---

## Endpoints

### Health Check

Check if the service is running.

**Request:**
```
GET /health
```

**Response:**
```json
{
  "status": "ok"
}
```

**Example:**
```bash
curl http://localhost:8080/health
```

---

### Verify Payment (V1 - Legacy)

Verify an existing blockchain transaction against specified criteria.

**Request:**
```
POST /api/v1/verify
Content-Type: application/json
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tx_id` | string | Yes | Transaction ID (with orwithout `0x` prefix) |
| `expected_recipient` | string | Yes | Expected recipient Stacks address |
| `min_amount` | integer | Yes | Minimum amount in base units (microSTX) |
| `network` | string | Yes | Network: `mainnet` or `testnet` |
| `token_type` | string | No | Token type: `STX`, `SBTC`, `USDCX` (default: `STX`) |
| `expected_sender` | string | No | Optional sender address to validate |
| `expected_memo` | string | No | Optional memo to validate |

**Example Request:**
```bash
curl -X POST http://localhost:8080/api/v1/verify \
  -H "Content-Type: application/json" \
  -d '{
    "tx_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
    "expected_recipient": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
    "min_amount": 1000000,
    "network": "testnet",
    "token_type": "STX"
  }'
```

**Success Response (200 OK):**
```json
{
  "valid": true,
  "tx_id": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
  "sender_address": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
  "recipient_address": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
  "amount": 1000000,
  "fee": 180,
  "nonce": 5,
  "status": "success",
  "block_height": 12345,
  "token_type": "STX",
  "memo": "payment for service",
  "network": "testnet"
}
```

**Validation Failed Response (200 OK):**
```json
{
  "valid": false,
  "tx_id": "0x...",
  "sender_address": "ST...",
  "recipient_address": "ST...",
  "amount": 500000,
  "status": "success",
  "errors": [
    "insufficient amount: expected at least 1000000, got 500000"
  ]
}
```

---

### Settle Payment (V1 - Legacy)

Broadcast a signed transaction and wait for confirmation.

**Request:**
```
POST /api/v1/settle
Content-Type: application/json
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `signed_transaction` | string | Yes | Hex-encoded signed transaction |
| `expected_recipient` | string | Yes | Expected recipient Stacks address |
| `min_amount` | integer | Yes | Minimum amount in base units |
| `network` | string | Yes | Network: `mainnet` or `testnet` |
| `token_type` | string | No | Token type: `STX`, `SBTC`, `USDCX` (default: `STX`) |
| `expected_sender` | string | No | Optional sender address to validate |

**Example Request:**
```bash
curl -X POST http://localhost:8080/api/v1/settle \
  -H "Content-Type: application/json" \
  -d '{
    "signed_transaction": "0x00000001038...",
    "expected_recipient": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
    "min_amount": 1000000,
    "network": "testnet"
  }'
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "tx_id": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
  "sender_address": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
  "recipient_address": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
  "amount": 1000000,
  "fee": 180,
  "status": "success",
  "block_height": 12346,
  "token_type": "STX",
  "network": "testnet"
}
```

**Settlement Failed Response (400 Bad Request):**
```json
{
  "success": false,
  "tx_id": "0x...",
  "status": "success",
  "errors": [
    "recipient mismatch: expected ST1..., got ST2..."
  ]
}
```

---

### Verify Payment (V2 - x402 Coinbase-Compatible)

Verify a payment using the x402 v2 protocol format (Coinbase-compatible).

**Request:**
```
POST /verify
Content-Type: application/json
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `x402Version` | integer | No | Protocol version (default: 2) |
| `paymentPayload` | object | Yes | Client's payment authorization |
| `paymentRequirements` | object | Yes | Server's payment requirements |

**PaymentRequirements Object:**

| Field | Type | Description |
|-------|------|-------------|
| `scheme` | string | Payment scheme: `exact` |
| `network` | string | CAIP-2 network ID: `stacks:1` (mainnet) or `stacks:2147483648` (testnet) |
| `amount` | string | Amount in microSTX |
| `asset` | string | Asset identifier: `STX` or contract address |
| `payTo` | string | Recipient Stacks address |
| `maxTimeoutSeconds` | integer | Maximum payment timeout |
| `extra` | object | Optional additional parameters |

**Example Request:**
```bash
curl -X POST http://localhost:8080/verify \
  -H "Content-Type: application/json" \
  -d '{
    "x402Version": 2,
    "paymentPayload": {
      "x402Version": 2,"accepted": {
        "scheme": "exact",
        "network": "stacks:2147483648",
        "amount": "1000000",
        "asset": "STX",
        "payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
      },
      "payload": {
        "transaction": "0x1234567890abcdef..."
      }
    },
    "paymentRequirements": {
      "scheme": "exact",
      "network": "stacks:2147483648",
      "amount": "1000000",
      "asset": "STX",
      "payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
    }
  }'
```

**Success Response (200 OK):**
```json
{
  "isValid": true
}
```

**Invalid Payment Response (200 OK):**
```json
{
  "isValid": false,
  "invalidReason": "insufficient_amount",
  "payer": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7"
}
```

---

### Settle Payment (V2 - x402 Coinbase-Compatible)

Settle a payment using the x402 v2 protocol format (Coinbase-compatible).

**Request:**
```
POST /settle
Content-Type: application/json
```

**Request Body:**

Same structure as verify endpoint.

**Example Request:**
```bash
curl -X POST http://localhost:8080/settle \
  -H "Content-Type: application/json" \
  -d '{
    "x402Version": 2,
    "paymentPayload": {
      "x402Version": 2,
      "accepted": {
        "scheme": "exact",
        "network": "stacks:2147483648",
        "amount": "1000000",
        "asset": "STX",
        "payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
      },
      "payload": {
        "transaction": "0x00000001038..."
      }
    },
    "paymentRequirements": {
      "scheme": "exact",
      "network": "stacks:2147483648",
      "amount": "1000000",
      "asset": "STX",
      "payTo": "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM"
    }
  }'
```

**Success Response (200 OK):**
```json
{
  "success": true,
  "payer": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
  "transaction": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
  "network": "stacks:2147483648"
}
```

**Failed Response (400 Bad Request):**
```json
{
  "success": false,
  "errorReason": "broadcast_failed",
  "payer": "ST2J6ZY48GV1EZ5V2V5RB9MP66SW86PYKKNRV9EJ7",
  "transaction": "",
  "network": "stacks:2147483648"
}
```

---

### Supported Payment Methods

Get the list of supported payment methods and networks.

**Request:**
```
GET /supported
```

**Response (200 OK):**
```json
{
  "kinds": [
    {
      "x402Version": 2,
      "scheme": "exact",
      "network": "stacks:1"
    },
    {
      "x402Version": 2,
      "scheme": "exact",
      "network": "stacks:2147483648"
    }
  ],
  "extensions": [],
  "signers": {
    "stacks:*": []
  }
}
```

**Example:**
```bash
curl http://localhost:8080/supported
```

---

## Token Types

| Token | Description | Transaction Type |
|-------|-------------|------------------|
| `STX` | Native Stacks token | `token_transfer` |
| `SBTC` | Bitcoin on Stacks (SIP-010) | `contract_call` |
| `USDCX` | USDC on Stacks (SIP-010) | `contract_call` |

## Amount Units

All amounts are in **base units**:

| Token | Base Unit | Conversion |
|-------|-----------|------------|
| STX | microSTX | 1 STX = 1,000,000 microSTX |
| SBTC | satoshis | 1 sBTC = 100,000,000 satoshis |
| USDCX | micro USDC | 1 USDC = 1,000,000 micro USDC |

## Network Identifiers

The V2 API uses CAIP-2 format for network identifiers:

| Network | V1 Format | V2 Format (CAIP-2) |
|---------|-----------|---------------------|
| Mainnet | `mainnet` | `stacks:1` |
| Testnet | `testnet` | `stacks:2147483648` |

Both formats are accepted inV1 endpoints. V2 endpoints require CAIP-2 format.

## Verification Rules

The service validates transactions against these criteria:

1. **Transaction Status**: Must not be `failed`, `abort_by_response`, or `abort_by_post_condition`
2. **Confirmation**: Transaction must be confirmed (block_height > 0)
3. **Recipient**: Must match `expected_recipient` exactly
4. **Amount**: Must be >= `min_amount`
5. **Sender** (optional): If specified, must match exactly
6. **Memo** (optional): If specified, must match exactly

## Error Responses

### V1 Error Format

```json
{
  "error": "error_code",
  "message": "Human readable error message"
}
```

### V2 Error Format

V1 endpoints return HTTP 200 with `valid: false`. Errors are in the `errors` array:

```json
{
  "valid": false,
  "errors": ["insufficient amount: expected 1000000, got 500000"]
}
```

V2 endpoints return structured responses with `isValid` and `invalidReason`:

```json
{
  "isValid": false,
  "invalidReason": "insufficient_amount"
}
```

## Project Structure

```
stacks-facilitator/
├── cmd/server/main.go                 # Application entry point
├── internal/
│   ├── payment/
│   │   ├── domain/                    # Business logic (DDD)
│   │   │   ├── valueobject/           # Value objects
│   │   │   └── service/               # Domain services
│   │   ├── application/command/       # Use cases
│   │   └── infrastructure/            # External concerns
│   │       ├── blockchain/            # Stacks client adapter
│   │       └── http/                  # HTTP handlers
│   └── stacks/                        # Hiro API client
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

## Testing

```bash
# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package tests
go test ./internal/payment/domain/valueobject/... -v
```

## Usage Examples

### JavaScript/TypeScript

```typescript
// Verify a payment (V1)
async function verifyPayment(txId: string, recipient: string, amount: number) {
  const response = await fetch('http://localhost:8080/api/v1/verify', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      tx_id: txId,
      expected_recipient: recipient,
      min_amount: amount,
      network: 'mainnet',
      token_type: 'STX'
    })
  });
  return response.json();
}

// Check health
async function checkHealth() {
  const response = await fetch('http://localhost:8080/health');
  return response.json();
}
```

### Python

```python
import requests

# Verify a payment (V1)
def verify_payment(tx_id: str, recipient: str, amount: int):
    response = requests.post('http://localhost:8080/api/v1/verify', json={
        'tx_id': tx_id,
        'expected_recipient': recipient,
        'min_amount': amount,
        'network': 'mainnet',
        'token_type': 'STX'
    })
    return response.json()

# Settle a payment (V1)
def settle_payment(signed_tx: str, recipient: str, amount: int):
    response = requests.post('http://localhost:8080/api/v1/settle', json={
        'signed_transaction': signed_tx,
        'expected_recipient': recipient,
        'min_amount': amount,
        'network': 'mainnet'
    })
    return response.json()
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type VerifyRequest struct {
    TxID              string `json:"tx_id"`
    ExpectedRecipient string `json:"expected_recipient"`
    MinAmount         uint64 `json:"min_amount"`
    Network           string `json:"network"`
    TokenType         string `json:"token_type"`
}

func main() {
    req := VerifyRequest{
        TxID:              "0x1234...",
        ExpectedRecipient: "ST1PQHQKV0RJXZFY1DGX8MNSNYVE3VGZJSRTPGZGM",
        MinAmount:         1000000,
        Network:           "mainnet",
        TokenType:         "STX",
    }
    
    body, _ := json.Marshal(req)
    resp, err := http.Post(
        "http://localhost:8080/api/v1/verify",
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Printf("Result: %+v\n", result)
}
```

## License

MIT