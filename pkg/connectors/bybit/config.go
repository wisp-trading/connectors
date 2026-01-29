package bybit

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// Config holds the configuration for the Bybit connector
type Config struct {
	APIKey          string  `json:"api_key"`
	APISecret       string  `json:"api_secret"`
	BaseURL         string  `json:"base_url,omitempty"`
	IsTestnet       bool    `json:"is_testnet,omitempty"`
	DefaultSlippage float64 `json:"default_slippage,omitempty"` // Default 0.005 (0.5%)
}

var _ connector.Config = (*Config)(nil)

func (b *bybit) NewConfig() connector.Config {
	return &Config{}
}

func (c Config) ExchangeName() connector.ExchangeName {
	return types.Bybit
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if c.APISecret == "" {
		return fmt.Errorf("api_secret is required")
	}

	// Set default slippage if not provided
	if c.DefaultSlippage == 0 {
		c.DefaultSlippage = 0.005
	}

	// Set base URL based on testnet flag if not explicitly provided
	if c.BaseURL == "" {
		if c.IsTestnet {
			c.BaseURL = "https://api-testnet.bybit.com"
		} else {
			c.BaseURL = "https://api.bybit.com"
		}
	}

	return nil
}
