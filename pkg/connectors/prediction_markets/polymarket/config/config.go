package config

import (
	"fmt"
	"strings"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

func NewConfig() connector.Config {
	return &Config{}
}

// Config holds the configuration for Polymarket connector
type Config struct {
	// Authentication
	PrivateKey        string `json:"private_key"`        // Ethereum private key for signing orders
	PolymarketAddress string `json:"polymarket_address"` // Safe proxy wallet address

	SignatureType int `json:"signature_type,omitempty"` // Signature type: 0=EOA, 1=Proxy/magic.link, 2=GnosisSafe (default: 1)
}

var _ connector.Config = (*Config)(nil)

// ExchangeName returns the exchange name for Polymarket
func (c Config) ExchangeName() connector.ExchangeName {
	return types.Polymarket
}

// Validate checks if the configuration is valid and sets defaults
func (c *Config) Validate() error {
	if c.PrivateKey == "" {
		return fmt.Errorf("private_key is required")
	}
	if c.PolymarketAddress == "" {
		return fmt.Errorf("funder_address is required")
	}

	// Validate private key format (should be hex string, typically 64 chars + optional 0x prefix)
	privateKey := strings.TrimPrefix(c.PrivateKey, "0x")
	if len(privateKey) < 64 || !isHexString(privateKey) {
		return fmt.Errorf("private_key must be a valid hex string (64+ hex characters)")
	}

	// Validate funder address format (Ethereum address: 0x + 40 hex chars)
	if !strings.HasPrefix(c.PolymarketAddress, "0x") || len(c.PolymarketAddress) != 42 || !isHexString(c.PolymarketAddress[2:]) {
		return fmt.Errorf("funder_address must be a valid Ethereum address (0x followed by 40 hex characters)")
	}

	// Set default signature type (Proxy/magic.link wallet)
	if c.SignatureType == 0 {
		c.SignatureType = 1
	}

	return nil
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
