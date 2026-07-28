package uniswap_v3

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// Config is the EVM UniV3 connector configuration.
// Works for any EVM chain with UniV3-compatible router/quoter (Robinhood pilot default).
type Config struct {
	// RPCURL is the HTTPS JSON-RPC endpoint for the chain.
	RPCURL string `json:"rpc_url"`
	// PrivateKey is the hex-encoded signing key (no 0x). Required for live swaps.
	// Never exposed to agents — bot-only credential store.
	PrivateKey string `json:"private_key"`
	// AccountAddress is the 0x wallet address (checksum optional).
	AccountAddress string `json:"account_address"`
	// ChainID is the EVM chain id (required). Stored as string in connectors.yml often;
	// JSON number also works when written by Settings.
	ChainID uint64 `json:"chain_id"`
	// SwapRouter is the UniV3 SwapRouter02 (or compatible) address.
	SwapRouter string `json:"swap_router"`
	// Quoter is the QuoterV2 address.
	Quoter string `json:"quoter"`
	// WETH is the wrapped native token address used as default quote.
	WETH string `json:"weth"`
	// DefaultFeeTier is the UniV3 fee in hundredths of a bip (10000 = 1%).
	DefaultFeeTier uint32 `json:"default_fee_tier,omitempty"`
	// DefaultSlippage is fraction (0.05 = 5%). Higher for meme chips.
	DefaultSlippage float64 `json:"default_slippage,omitempty"`
	// DryRun when true never broadcasts — returns simulated order ids.
	// Also forced when private_key is empty.
	DryRun bool `json:"dry_run,omitempty"`
	// Network label for Settings UI (mainnet / robinhood / …).
	Network string `json:"network,omitempty"`
}

var _ connector.Config = (*Config)(nil)

func (c Config) ExchangeName() connector.ExchangeName {
	return types.UniswapV3
}

func (c *Config) Validate() error {
	if c.RPCURL == "" {
		return fmt.Errorf("rpc_url is required")
	}
	if c.ChainID == 0 {
		return fmt.Errorf("chain_id is required")
	}
	if c.DefaultFeeTier == 0 {
		c.DefaultFeeTier = 10000 // 1% — common meme pool fee
	}
	if c.DefaultSlippage == 0 {
		c.DefaultSlippage = 0.05 // 5%
	}
	if c.DefaultSlippage < 0 || c.DefaultSlippage > 0.5 {
		return fmt.Errorf("default_slippage must be between 0 and 0.5, got %f", c.DefaultSlippage)
	}
	if c.PrivateKey == "" {
		c.DryRun = true
	}
	if !c.DryRun && c.AccountAddress == "" {
		return fmt.Errorf("account_address is required when not dry_run")
	}
	return nil
}
