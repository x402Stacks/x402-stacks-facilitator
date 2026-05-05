# Stacks Facilitator

A stateless payment verification and settlement service for the Stacks blockchain, implementing the x402 protocol. Built with Go, following Domain-Driven Design (DDD) principles.

## Features

- **Verify** existing blockchain transactions against specified criteria
- **Settle** payments by broadcasting signed transactions and confirming them on-chain
- **Multi-token support**: STX, sBTC, USDCx
- **Multi-network support**: Mainnet and Testnet
- **Stateless**: No database required
- **Retry logic**: Built-in retry mechanism for blockchain operations

## Requirements

- Go 1.24+ (for local development)
- Docker (for containerized deployment)

## Quick Start

### Run with Docker (Recommended)

```bash
# Build and run
docker-compose up -d

# Check health
curl http://localhost:8080/health
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

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `HIRO_API_KEY` | unset | Optional Hiro API key sent as `x-api-key` on Hiro API requests |

## API Reference

### Base URLs

- **Mainnet API**: `https://api.mainnet.hiro.so`
- **Testnet API**: `https://api.testnet.hiro.so`

---

### Health Check

Check if the service is running.

```
GET /health
```

**Response:**
```json
{
  "status": "ok"
}
```

---

### Verify Payment

Verify an existing blockchain transaction against specified criteria.

```
POST /api/v1/verify
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tx_id` | string | Yes | Transaction ID (with or without `0x` prefix) |
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
  "status": "confirmed",
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
  "status": "confirmed",
  "errors": [
    "insufficient amount: expected at least 1000000, got 500000"
  ]
}
```

---

### Settle Payment

Broadcast a signed transaction and wait for confirmation.

```
POST /api/v1/settle
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
    "signed_transaction": "0x00000001...",
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
  "status": "confirmed",
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
  "status": "confirmed",
  "errors": [
    "recipient mismatch: expected ST1..., got ST2..."
  ]
}
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

## Verification Rules

The service validates transactions against these criteria:

1. **Transaction Status**: Must not be `failed`, `abort_by_response`, or `abort_by_post_condition`
2. **Confirmation**: Transaction must be confirmed (block_height > 0)
3. **Recipient**: Must match `expected_recipient` exactly
4. **Amount**: Must be >= `min_amount`
5. **Sender** (optional): If specified, must match exactly
6. **Memo** (optional): If specified, must match exactly

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

## License

MIT
