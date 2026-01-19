package valueobject

import (
	"errors"
	"strings"
)

// StacksAddress represents a Stacks blockchain address
type StacksAddress struct {
	value string
}

// NewStacksAddress creates a new StacksAddress from a string
func NewStacksAddress(addr string) (StacksAddress, error) {
	if addr == "" {
		return StacksAddress{}, errors.New("address cannot be empty")
	}

	// Normalize to uppercase (Stacks addresses are canonically uppercase)
	normalized := strings.ToUpper(addr)

	// Validate prefix (ST for testnet, SP for mainnet, SM for mainnet multisig, SN for testnet multisig)
	if !strings.HasPrefix(normalized, "ST") && !strings.HasPrefix(normalized, "SP") &&
		!strings.HasPrefix(normalized, "SM") && !strings.HasPrefix(normalized, "SN") {
		return StacksAddress{}, errors.New("invalid Stacks address prefix: must start with ST, SP, SM, or SN")
	}

	// Validate length (Stacks addresses are typically 41 characters)
	if len(normalized) < 30 || len(normalized) > 50 {
		return StacksAddress{}, errors.New("invalid Stacks address length")
	}

	return StacksAddress{value: normalized}, nil
}

// String returns the address as a string
func (a StacksAddress) String() string {
	return a.value
}

// Equals checks if two StacksAddresses are equal
func (a StacksAddress) Equals(other StacksAddress) bool {
	return a.value == other.value
}

// IsTestnet checks if the address is for testnet
func (a StacksAddress) IsTestnet() bool {
	return strings.HasPrefix(a.value, "ST") || strings.HasPrefix(a.value, "SN")
}

// IsMainnet checks if the address is for mainnet
func (a StacksAddress) IsMainnet() bool {
	return strings.HasPrefix(a.value, "SP") || strings.HasPrefix(a.value, "SM")
}

// IsZero checks if the address is empty
func (a StacksAddress) IsZero() bool {
	return a.value == ""
}
