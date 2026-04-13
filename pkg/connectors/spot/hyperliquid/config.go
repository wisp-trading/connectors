package hyperliquid

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// Config holds the configuration for the Hyperliquid spot connector.
// Fields mirror the perps config — both share the same API credentials.
type Config struct {
	BaseURL        string  `json:"base_url,omitempty"`
	WebsocketURL   string  `json:"websocket_url,omitempty"`
	PrivateKey     string  `json:"private_key"`
	AccountAddress string  `json:"account_address"`
	VaultAddress   string  `json:"vault_address,omitempty"`
	UseTestnet     bool    `json:"use_testnet,omitempty"`
	DefaultSlippage float64 `json:"default_slippage,omitempty"`
}

var _ connector.Config = (*Config)(nil)

func (h *hyperliquidSpot) NewConfig() connector.Config {
	return &Config{}
}

func (c Config) ExchangeName() connector.ExchangeName {
	return types.HyperliquidSpot
}

func (c *Config) Validate() error {
	if c.PrivateKey == "" {
		return fmt.Errorf("private_key is required")
	}
	if c.AccountAddress == "" {
		return fmt.Errorf("account_address is required")
	}

	if c.UseTestnet {
		if c.BaseURL == "" {
			c.BaseURL = "https://api.hyperliquid-testnet.xyz"
		}
		if c.WebsocketURL == "" {
			c.WebsocketURL = "wss://api.hyperliquid-testnet.xyz/ws"
		}
	} else {
		if c.BaseURL == "" {
			c.BaseURL = "https://api.hyperliquid.xyz"
		}
		if c.WebsocketURL == "" {
			c.WebsocketURL = "wss://api.hyperliquid.xyz/ws"
		}
	}

	if c.DefaultSlippage == 0 {
		c.DefaultSlippage = 0.005
	}

	if c.DefaultSlippage < 0 || c.DefaultSlippage > 0.1 {
		return fmt.Errorf("default_slippage must be between 0 and 0.1 (0-10%%), got: %f", c.DefaultSlippage)
	}

	return nil
}
