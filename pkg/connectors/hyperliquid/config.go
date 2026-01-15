package hyperliquid

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
)

type Config struct {
	BaseURL         string  `json:"base_url,omitempty"`
	WebsocketURL    string  `json:"websocket_url,omitempty"`
	PrivateKey      string  `json:"private_key"`
	AccountAddress  string  `json:"account_address"`
	VaultAddress    string  `json:"vault_address,omitempty"`
	UseTestnet      bool    `json:"use_testnet,omitempty"`
	DefaultSlippage float64 `json:"default_slippage,omitempty"` // Default slippage for market orders (0.005 = 0.5%)
}

var _ connector.Config = (*Config)(nil)

func (h *hyperliquid) NewConfig() connector.Config {
	return &Config{}
}

func (c Config) ExchangeName() connector.ExchangeName {
	return types.Hyperliquid
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
