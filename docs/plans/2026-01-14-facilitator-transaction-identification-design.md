# Facilitator Transaction Identification Design

**Date:** 2026-01-14
**Status:** Approved

## Problem

We need to identify on-chain whether a transaction was settled through the x402 facilitator vs. submitted directly, for analytics purposes.

**Constraints:**
- Must be provable purely from blockchain data (on-chain only)
- Facilitator must remain stateless (no database)
- Analytics will be handled by a separate backend service

## Solution

Use a **memo prefix convention** - all facilitator-routed transactions include `x402:` prefix in the memo field.

## Memo Format

```
x402:<24-char-base64url-nonce>
```

| Component | Size | Description |
|-----------|------|-------------|
| Prefix | 5 bytes | `x402:` - identifies facilitator-routed transactions |
| Nonce | 24 chars | 18 random bytes, base64url encoded (144 bits entropy) |
| **Total** | **29 bytes** | Within 34-byte Stacks memo limit |

**Example:** `x402:Ab3Kx9mPqR2sT5vW8yZ1aB3K`

## Changes Required

### SDK (x402-stacks)

| File | Change |
|------|--------|
| `middleware.ts` | Change nonce generation from `randomBytes(16).toString('hex')` to `randomBytes(18).toString('base64url')` |
| `client.ts` | Prepend `x402:` to memo in `signSTXTransfer()`, `signSBTCTransfer()`, `signUSDCxTransfer()` |
| `interceptor.ts` | Prepend `x402:` to memo in `signPayment()` |

### Facilitator (stacks-facilitator)

No changes required - remains stateless.

## Analytics Integration (Separate Backend)

The analytics backend (out of scope for this design) will:

1. **Set up Chainhook** to capture all STX/sBTC/USDCx transfers:

```json
{
  "chain": "stacks",
  "uuid": "x402-analytics",
  "name": "X402 Transfer Events",
  "version": 1,
  "networks": {
    "mainnet": {
      "if_this": {
        "scope": "stx_event",
        "actions": ["transfer"]
      },
      "then_that": {
        "http_post": {
          "url": "https://analytics-api.example.com/webhook/stx-transfers",
          "authorization_header": "Bearer <token>"
        }
      }
    }
  }
}
```

2. **Filter by memo prefix** in webhook handler:

```typescript
function handleWebhook(tx: StacksTransaction) {
  if (tx.memo?.startsWith('x402:')) {
    // Facilitator-settled transaction - store in analytics DB
    await db.insert('x402_transactions', {
      txId: tx.tx_id,
      nonce: tx.memo.slice(5), // Extract nonce after prefix
      sender: tx.sender_address,
      recipient: tx.token_transfer.recipient_address,
      amount: tx.token_transfer.amount,
      timestamp: tx.block_time,
    });
  }
}
```

**Note:** Chainhook does not currently support memo filtering at the predicate level ([GitHub issue #488](https://github.com/hirosystems/chainhook/issues/488)), so filtering must happen application-side.

## Verification Logic

To check if a transaction was facilitator-routed:

```typescript
function isX402Transaction(memo: string): boolean {
  return memo?.startsWith('x402:');
}
```

Prefix-only check is sufficient for analytics purposes - no need to validate nonce format.

## Security Considerations

- **No cryptographic proof** - Anyone could manually craft a transaction with `x402:` prefix
- **Acceptable for analytics** - No incentive for users to fake the prefix
- **Not suitable for billing/compliance** - If stronger proof is needed later, consider sponsored transactions

## Alternatives Considered

| Approach | Pros | Cons | Decision |
|----------|------|------|----------|
| **Memo prefix (chosen)** | Simple, no cost, SDK-only change | Convention-based, not cryptographic | Selected |
| Sponsored transactions | Cryptographic proof, native feature | Facilitator pays fees | Rejected (cost) |
| Facilitator contract | Explicit on-chain record | Complex, higher gas | Rejected (complexity) |

## Implementation Scope

- **In scope:** SDK memo changes
- **Out of scope:** Facilitator changes, analytics backend
