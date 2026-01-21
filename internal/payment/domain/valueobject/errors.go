package valueobject

// V2 Error codes per x402 specification
const (
	ErrInsufficientFunds         = "insufficient_funds"
	ErrInvalidNetwork            = "invalid_network"
	ErrInvalidPayload            = "invalid_payload"
	ErrInvalidPaymentRequirements = "invalid_payment_requirements"
	ErrInvalidScheme             = "invalid_scheme"
	ErrUnsupportedScheme         = "unsupported_scheme"
	ErrInvalidX402Version        = "invalid_x402_version"
	ErrInvalidTransactionState   = "invalid_transaction_state"
	ErrUnexpectedVerifyError     = "unexpected_verify_error"
	ErrUnexpectedSettleError     = "unexpected_settle_error"
	ErrRecipientMismatch         = "recipient_mismatch"
	ErrAmountInsufficient        = "amount_insufficient"
	ErrSenderMismatch            = "sender_mismatch"
	ErrTransactionNotFound       = "transaction_not_found"
	ErrTransactionPending        = "transaction_pending"
	ErrTransactionFailed         = "transaction_failed"
	ErrBroadcastFailed           = "broadcast_failed"
)
