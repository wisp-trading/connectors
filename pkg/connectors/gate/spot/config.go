package spot

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
)

// Config holds the configuration for Gate.io Spot connector
type Config struct {
	APIKey          string  `json:"api_key"`
	APISecret       string  `json:"api_secret"`
	BaseURL         string  `json:"base_url,omitempty"`
	UseTestnet      bool    `json:"use_testnet,omitempty"`
	DefaultSlippage float64 `json:"default_slippage,omitempty"` // Default slippage for market orders (0.005 = 0.5%)
}

var _ connector.Config = (*Config)(nil)

func (c Config) ExchangeName() connector.ExchangeName {
	return types.GateSpot
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if c.APISecret == "" {
		return fmt.Errorf("api_secret is required")
	}

	// Set default base URL
	if c.BaseURL == "" {
		if c.UseTestnet {
			c.BaseURL = "https://fx-api-testnet.gateio.ws/api/v4"
		} else {
			c.BaseURL = "https://api.gateio.ws/api/v4"
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
