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
	PolymarketAddress string `json:"polymarket_address"` // Safe proxy wallet address (only used when signature_type != 0)

	// SignatureType controls which address the CLOB uses as the order maker/funder:
	//   0 = EOA (default) — maker is derived from private_key; on-chain ops (split/merge) use EOA
	//   1 = Proxy / magic.link
	//   2 = GnosisSafe — maker is polymarket_address (Safe); incompatible with on-chain ops
	//
	// IMPORTANT: when polygon_rpc_url is set (on-chain ops enabled), signature_type MUST be 0.
	// SplitPosition and MergePositions are always submitted by the EOA private key (msg.sender = EOA).
	// Using signature_type 2 causes CLOB fills to credit the Safe while on-chain ops run from the EOA
	// — positions end up at different addresses and MergePositions will revert.
	SignatureType int `json:"signature_type,omitempty"`

	// On-chain — required for SplitPosition / MergePositions (NegRisk arb).
	// The CTF client is initialised with this Polygon RPC backend; without it
	// all on-chain calls fail at runtime with [CTF-002].
	// Example: "https://polygon-mainnet.g.alchemy.com/v2/<key>"
	PolygonRPCURL string `json:"polygon_rpc_url"`
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

	// When on-chain operations are enabled (polygon_rpc_url is set), signature_type must be 0 (EOA).
	// SplitPosition/MergePositions are submitted by the EOA private key (msg.sender = EOA address).
	// Using signature_type 2 (Safe) would credit CLOB fills to the Safe while on-chain ops run from
	// the EOA — positions end up at different addresses and MergePositions will revert on-chain.
	if c.PolygonRPCURL != "" && c.SignatureType != 0 {
		return fmt.Errorf(
			"signature_type %d is incompatible with on-chain operations: "+
				"SplitPosition/MergePositions always run from the EOA (private key); "+
				"CLOB fills would be credited to the Safe instead — set signature_type: 0",
			c.SignatureType,
		)
	}

	// Polygon RPC is required for on-chain CTF operations (SplitPosition / MergePositions).
	// Add polygon_rpc_url to your wisp.yml credentials section.
	if c.PolygonRPCURL == "" {
		return fmt.Errorf("polygon_rpc_url is required (needed for on-chain CTF split/merge operations)")
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
