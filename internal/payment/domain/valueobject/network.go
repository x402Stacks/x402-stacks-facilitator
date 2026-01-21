package valueobject

import (
	"errors"
	"strings"
)

// Network represents a Stacks blockchain network
type Network string

const (
	NetworkMainnet Network = "mainnet"
	NetworkTestnet Network = "testnet"
)

// CAIP-2 network identifiers for Stacks
const (
	StacksMainnetCAIP2 = "stacks:1"
	StacksTestnetCAIP2 = "stacks:2147483648"
)

// NewNetwork creates a new Network from a string (supports both v1 names and CAIP-2)
func NewNetwork(s string) (Network, error) {
	if s == "" {
		return "", errors.New("network cannot be empty")
	}

	// Check for CAIP-2 format first
	switch s {
	case StacksMainnetCAIP2:
		return NetworkMainnet, nil
	case StacksTestnetCAIP2:
		return NetworkTestnet, nil
	}

	// Check for v1 format (simple names)
	normalized := strings.ToLower(s)
	switch normalized {
	case "mainnet":
		return NetworkMainnet, nil
	case "testnet":
		return NetworkTestnet, nil
	default:
		return "", errors.New("unsupported network: " + s)
	}
}

// NewNetworkFromCAIP2 creates a Network from a CAIP-2 identifier
func NewNetworkFromCAIP2(caip2 string) (Network, error) {
	switch caip2 {
	case StacksMainnetCAIP2:
		return NetworkMainnet, nil
	case StacksTestnetCAIP2:
		return NetworkTestnet, nil
	default:
		if strings.HasPrefix(caip2, "stacks:") {
			return "", errors.New("unsupported stacks network: " + caip2)
		}
		return "", errors.New("invalid CAIP-2 format: " + caip2)
	}
}

// String returns the network as a string
func (n Network) String() string {
	return string(n)
}

// APIBaseURL returns the Hiro API base URL for this network
func (n Network) APIBaseURL() string {
	switch n {
	case NetworkMainnet:
		return "https://api.mainnet.hiro.so"
	case NetworkTestnet:
		return "https://api.testnet.hiro.so"
	default:
		return ""
	}
}

// IsMainnet returns true if this is mainnet
func (n Network) IsMainnet() bool {
	return n == NetworkMainnet
}

// IsTestnet returns true if this is testnet
func (n Network) IsTestnet() bool {
	return n == NetworkTestnet
}

// ToCAIP2 returns the CAIP-2 identifier for this network
func (n Network) ToCAIP2() string {
	switch n {
	case NetworkMainnet:
		return StacksMainnetCAIP2
	case NetworkTestnet:
		return StacksTestnetCAIP2
	default:
		return ""
	}
}
