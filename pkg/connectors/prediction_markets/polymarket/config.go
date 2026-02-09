package polymarket

import (
	"fmt"
	"strings"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// Config holds the configuration for Polymarket connector
type Config struct {
	// Authentication
	APIKey        string `json:"api_key"`
	APISecret     string `json:"api_secret"`
	Passphrase    string `json:"passphrase"`
	PrivateKey    string `json:"private_key"`    // Ethereum private key for signing orders
	FunderAddress string `json:"funder_address"` // Safe proxy wallet address

	// Endpoints
	BaseURL      string `json:"base_url,omitempty"`      // CLOB REST API URL
	GammaURL     string `json:"gamma_url,omitempty"`     // Gamma Markets API URL
	WebSocketURL string `json:"websocket_url,omitempty"` // WebSocket URL

	// Trading Configuration
	ChainID       int `json:"chain_id,omitempty"`       // Polygon chain ID (default: 137)
	SignatureType int `json:"signature_type,omitempty"` // Signature type (default: 2 for Safe proxy wallet)
}

var _ connector.Config = (*Config)(nil)

// ExchangeName returns the exchange name for Polymarket
func (c Config) ExchangeName() connector.ExchangeName {
	return types.Polymarket
}

// Validate checks if the configuration is valid and sets defaults
func (c *Config) Validate() error {
	// Validate required authentication fields
	if c.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if c.APISecret == "" {
		return fmt.Errorf("api_secret is required")
	}
	if c.Passphrase == "" {
		return fmt.Errorf("passphrase is required")
	}
	if c.PrivateKey == "" {
		return fmt.Errorf("private_key is required")
	}
	if c.FunderAddress == "" {
		return fmt.Errorf("funder_address is required")
	}

	// Validate private key format (should be hex string, typically 64 chars + optional 0x prefix)
	privateKey := strings.TrimPrefix(c.PrivateKey, "0x")
	if len(privateKey) < 64 || !isHexString(privateKey) {
		return fmt.Errorf("private_key must be a valid hex string (64+ hex characters)")
	}

	// Validate funder address format (Ethereum address: 0x + 40 hex chars)
	if !strings.HasPrefix(c.FunderAddress, "0x") || len(c.FunderAddress) != 42 || !isHexString(c.FunderAddress[2:]) {
		return fmt.Errorf("funder_address must be a valid Ethereum address (0x followed by 40 hex characters)")
	}

	// Set default URLs if not provided
	if c.BaseURL == "" {
		c.BaseURL = "https://clob.polymarket.com"
	}
	if c.GammaURL == "" {
		c.GammaURL = "https://gamma-api.polymarket.com"
	}
	if c.WebSocketURL == "" {
		c.WebSocketURL = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
	}

	// Set default chain ID (Polygon mainnet)
	if c.ChainID == 0 {
		c.ChainID = 137
	}

	// Set default signature type (Safe proxy wallet)
	if c.SignatureType == 0 {
		c.SignatureType = 2
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
