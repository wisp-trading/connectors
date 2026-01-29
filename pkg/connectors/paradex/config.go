package paradex

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

type Config struct {
	BaseURL        string `json:"base_url,omitempty"`
	WebSocketURL   string `json:"websocket_url,omitempty"`
	StarknetRPC    string `json:"starknet_rpc,omitempty"`
	AccountAddress string `json:"account_address"`
	EthPrivateKey  string `json:"eth_private_key"`
	L2PrivateKey   string `json:"l2_private_key,omitempty"`
	Network        string `json:"network,omitempty"`
}

var _ connector.Config = (*Config)(nil)

func (p *paradex) NewConfig() connector.Config {
	return &Config{}
}

func (c Config) Validate() error {
	if c.EthPrivateKey == "" {
		return fmt.Errorf("eth_private_key is required")
	}
	if c.AccountAddress == "" {
		return fmt.Errorf("account_address is required")
	}
	if c.Network == "" {
		c.Network = "mainnet"
	}

	// Validate network value
	if c.Network != "mainnet" && c.Network != "testnet" {
		return fmt.Errorf("network must be 'mainnet' or 'testnet', got: %s", c.Network)
	}

	// Set defaults based on network
	if c.BaseURL == "" {
		if c.Network == "testnet" {
			c.BaseURL = "https://api.testnet.paradex.trade/consumer"
		} else {
			c.BaseURL = "https://api.paradex.trade/consumer"
		}
	}

	if c.StarknetRPC == "" {
		if c.Network == "testnet" {
			c.StarknetRPC = "https://starknet-sepolia.public.blastapi.io"
		} else {
			c.StarknetRPC = "https://starknet-mainnet.public.blastapi.io"
		}
	}

	if c.WebSocketURL == "" {
		if c.Network == "testnet" {
			c.WebSocketURL = "wss://ws.testnet.paradex.trade/v1"
		} else {
			c.WebSocketURL = "wss://ws.paradex.trade/v1"
		}
	}

	return nil
}

func (c Config) ExchangeName() connector.ExchangeName {
	return types.Paradex
}
