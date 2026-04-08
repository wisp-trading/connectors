package deribit

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// Config holds the configuration for Deribit options connector
type Config struct {
	ClientID       string `json:"client_id"`       // API Client ID
	ClientSecret   string `json:"client_secret"`   // API Client Secret
	BaseURL        string `json:"base_url,omitempty"`        // REST API URL
	WebSocketURL   string `json:"websocket_url,omitempty"`   // WebSocket URL
	UseTestnet     bool   `json:"use_testnet,omitempty"`     // Use testnet environment
	DefaultSlippage float64 `json:"default_slippage,omitempty"` // Default slippage for market orders
}

var _ connector.Config = (*Config)(nil)

func (c Config) ExchangeName() connector.ExchangeName {
	return types.DeribitOptions
}

func (c *Config) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("client_secret is required")
	}

	// Set default REST API URL
	if c.BaseURL == "" {
		if c.UseTestnet {
			c.BaseURL = "https://test.deribit.com/api/v2"
		} else {
			c.BaseURL = "https://www.deribit.com/api/v2"
		}
	}

	// Set default WebSocket URL
	if c.WebSocketURL == "" {
		if c.UseTestnet {
			c.WebSocketURL = "wss://test.deribit.com/ws/api/v2"
		} else {
			c.WebSocketURL = "wss://www.deribit.com/ws/api/v2"
		}
	}

	// Set default slippage if not specified (0.5%)
	if c.DefaultSlippage == 0 {
		c.DefaultSlippage = 0.005
	}

	// Validate slippage is reasonable (0-10%)
	if c.DefaultSlippage < 0 || c.DefaultSlippage > 0.1 {
		return fmt.Errorf("default_slippage must be between 0 and 0.1 (0-10%%), got: %f", c.DefaultSlippage)
	}

	return nil
}
