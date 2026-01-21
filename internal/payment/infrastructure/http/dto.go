package http

// VerifyRequest represents a verify payment request
type VerifyRequest struct {
	TxID              string  `json:"tx_id"`
	TokenType         string  `json:"token_type,omitempty"`
	ExpectedRecipient string  `json:"expected_recipient"`
	MinAmount         uint64  `json:"min_amount"`
	ExpectedSender    *string `json:"expected_sender,omitempty"`
	ExpectedMemo      *string `json:"expected_memo,omitempty"`
	Network           string  `json:"network"`
}

// VerifyResponse represents a verify payment response
type VerifyResponse struct {
	Valid            bool     `json:"valid"`
	TxID             string   `json:"tx_id"`
	SenderAddress    string   `json:"sender_address"`
	RecipientAddress string   `json:"recipient_address"`
	Amount           uint64   `json:"amount"`
	Fee              uint64   `json:"fee"`
	Nonce            uint64   `json:"nonce,omitempty"`
	Status           string   `json:"status"`
	BlockHeight      uint64   `json:"block_height"`
	TokenType        string   `json:"token_type"`
	Memo             string   `json:"memo,omitempty"`
	Network          string   `json:"network"`
	Errors           []string `json:"errors,omitempty"`
}

// SettleRequest represents a settle payment request
type SettleRequest struct {
	SignedTransaction string  `json:"signed_transaction"`
	TokenType         string  `json:"token_type,omitempty"`
	ExpectedRecipient string  `json:"expected_recipient"`
	MinAmount         uint64  `json:"min_amount"`
	ExpectedSender    *string `json:"expected_sender,omitempty"`
	Network           string  `json:"network"`
}

// SettleResponse represents a settle payment response
type SettleResponse struct {
	Success          bool     `json:"success"`
	TxID             string   `json:"tx_id"`
	SenderAddress    string   `json:"sender_address"`
	RecipientAddress string   `json:"recipient_address"`
	Amount           uint64   `json:"amount"`
	Fee              uint64   `json:"fee"`
	Status           string   `json:"status"`
	BlockHeight      uint64   `json:"block_height"`
	TokenType        string   `json:"token_type"`
	Network          string   `json:"network"`
	Errors           []string `json:"errors,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// ===== V2 Types (Coinbase x402 compatible) =====

// ResourceInfo describes the protected resource
type ResourceInfo struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// PaymentRequirementsV2 defines acceptable payment method (x402 v2)
type PaymentRequirementsV2 struct {
	Scheme            string                 `json:"scheme"`
	Network           string                 `json:"network"` // CAIP-2 format: stacks:1, stacks:2147483648
	Amount            string                 `json:"amount"`
	Asset             string                 `json:"asset"` // "STX" or contract identifier
	PayTo             string                 `json:"payTo"`
	MaxTimeoutSeconds int                    `json:"maxTimeoutSeconds"`
	Extra             map[string]interface{} `json:"extra,omitempty"`
}

// PaymentPayloadV2 is the client's payment authorization (x402 v2)
type PaymentPayloadV2 struct {
	X402Version int                    `json:"x402Version"`
	Resource    *ResourceInfo          `json:"resource,omitempty"`
	Accepted    PaymentRequirementsV2  `json:"accepted"`
	Payload     map[string]interface{} `json:"payload"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`
}

// VerifyRequestV2 for POST /verify (x402 v2)
type VerifyRequestV2 struct {
	X402Version         int                   `json:"x402Version"`
	PaymentPayload      PaymentPayloadV2      `json:"paymentPayload"`
	PaymentRequirements PaymentRequirementsV2 `json:"paymentRequirements"`
}

// VerifyResponseV2 for POST /verify (x402 v2)
type VerifyResponseV2 struct {
	IsValid       bool   `json:"isValid"`
	InvalidReason string `json:"invalidReason,omitempty"`
	Payer         string `json:"payer,omitempty"`
}

// SettleRequestV2 for POST /settle (x402 v2) - same structure as verify
type SettleRequestV2 struct {
	X402Version         int                   `json:"x402Version"`
	PaymentPayload      PaymentPayloadV2      `json:"paymentPayload"`
	PaymentRequirements PaymentRequirementsV2 `json:"paymentRequirements"`
}

// SettleResponseV2 for POST /settle (x402 v2 SettlementResponse)
type SettleResponseV2 struct {
	Success     bool   `json:"success"`
	ErrorReason string `json:"errorReason,omitempty"`
	Payer       string `json:"payer,omitempty"`
	Transaction string `json:"transaction"`
	Network     string `json:"network"`
}

// SupportedKind represents a supported payment kind
type SupportedKind struct {
	X402Version int                    `json:"x402Version"`
	Scheme      string                 `json:"scheme"`
	Network     string                 `json:"network"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// SupportedResponse for GET /supported
type SupportedResponse struct {
	Kinds      []SupportedKind     `json:"kinds"`
	Extensions []string            `json:"extensions"`
	Signers    map[string][]string `json:"signers"`
}
