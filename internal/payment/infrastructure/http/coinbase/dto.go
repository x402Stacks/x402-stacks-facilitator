// Package coinbase provides Coinbase x402 format compatibility adapter
package coinbase

// VerifyRequest represents a Coinbase-format verify request
type VerifyRequest struct {
	PaymentPayload      PaymentPayload      `json:"paymentPayload"`
	PaymentRequirements PaymentRequirements `json:"paymentRequirements"`
}

// SettleRequest represents a Coinbase-format settle request
type SettleRequest struct {
	PaymentPayload      PaymentPayload      `json:"paymentPayload"`
	PaymentRequirements PaymentRequirements `json:"paymentRequirements"`
}

// PaymentPayload represents the payment payload in Coinbase format
type PaymentPayload struct {
	X402Version int            `json:"x402Version"`
	Scheme      string         `json:"scheme"`
	Network     string         `json:"network"`
	Payload     StacksPayload  `json:"payload"`
}

// StacksPayload contains Stacks-specific transaction data
// Mirrors Coinbase EVM pattern: pre-parsed authorization fields + signed tx
type StacksPayload struct {
	SignedTransaction string        `json:"signedTransaction"`
	Authorization     Authorization `json:"authorization"`
}

// Authorization contains pre-parsed transaction fields for validation
// This mirrors Coinbase's EVM approach where clients provide parsed fields
type Authorization struct {
	From  string `json:"from"`  // Sender address (e.g., "SP..." or "ST...")
	To    string `json:"to"`    // Recipient address
	Value string `json:"value"` // Amount in base units (string for large numbers)
}

// PaymentRequirements specifies what the payment must satisfy
type PaymentRequirements struct {
	Scheme            string `json:"scheme"`
	Network           string `json:"network"`
	MaxAmountRequired string `json:"maxAmountRequired"`
	Asset             string `json:"asset,omitempty"`
	PayTo             string `json:"payTo"`
	Resource          string `json:"resource,omitempty"`
	Description       string `json:"description,omitempty"`
}

// VerifyResponse represents a Coinbase-format verify response
type VerifyResponse struct {
	IsValid       bool    `json:"isValid"`
	Payer         string  `json:"payer"`
	InvalidReason *string `json:"invalidReason,omitempty"`
}

// SettleResponse represents a Coinbase-format settle response
type SettleResponse struct {
	Success     bool    `json:"success"`
	Payer       string  `json:"payer"`
	Transaction string  `json:"transaction"`
	Network     string  `json:"network"`
	ErrorReason *string `json:"errorReason,omitempty"`
}

// SupportedKind represents a supported payment kind
type SupportedKind struct {
	X402Version int    `json:"x402Version"`
	Scheme      string `json:"scheme"`
	Network     string `json:"network"`
}

// SupportedResponse represents the /supported endpoint response
type SupportedResponse struct {
	Kinds      []SupportedKind   `json:"kinds"`
	Extensions []string          `json:"extensions"`
	Signers    map[string]string `json:"signers,omitempty"`
}

// ErrorCode constants for Coinbase-compatible error responses
const (
	ErrorCodeInsufficientFunds        = "insufficient_funds"
	ErrorCodeInvalidNetwork           = "invalid_network"
	ErrorCodeRecipientMismatch        = "invalid_exact_stacks_payload_recipient_mismatch"
	ErrorCodeAmountTooLow             = "invalid_exact_stacks_payload_authorization_value"
	ErrorCodeInvalidSignature         = "invalid_exact_stacks_payload_signature"
	ErrorCodeMissingPayload           = "invalid_exact_stacks_payload_missing"
	ErrorCodeMalformedPayload         = "invalid_exact_stacks_payload_malformed"
	ErrorCodeBroadcastFailed          = "broadcast_failed"
	ErrorCodeConfirmationFailed       = "confirmation_failed"
)
