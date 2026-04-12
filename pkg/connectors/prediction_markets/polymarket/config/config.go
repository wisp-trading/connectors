package config

import (
	"fmt"
	"strings"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// SignatureType represents the wallet signature type for Polymarket orders
type SignatureType int

const (
	SignatureTypeEOA         SignatureType = 0
	SignatureTypeProxy       SignatureType = 1
	SignatureTypeGnosisSafe  SignatureType = 2
)

// UnmarshalJSON parses signature type from string (e.g., "EOA", "GNOSIS_SAFE")
func (s *SignatureType) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)
	switch str {
	case "EOA", "eoa":
		*s = SignatureTypeEOA
	case "PROXY", "proxy":
		*s = SignatureTypeProxy
	case "GNOSIS_SAFE", "gnosis_safe":
		*s = SignatureTypeGnosisSafe
	default:
		return fmt.Errorf("invalid signature_type: %q (expected EOA, PROXY, or GNOSIS_SAFE)", str)
	}
	return nil
}

func NewConfig() connector.Config {
	return &Config{}
}

// Config holds the configuration for Polymarket connector
type Config struct {
	// Authentication
	PrivateKey        string        `json:"private_key"`        // Ethereum private key for signing orders
	PolymarketAddress string        `json:"polymarket_address"` // Safe proxy wallet address (only used when signature_type != EOA)
	SignatureType     SignatureType `json:"signature_type,omitempty"` // EOA (default), PROXY, or GNOSIS_SAFE

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
	// polymarket_address is optional for Safe mode (auto-derived from EOA).
	// For non-Safe modes, it's recommended but not strictly required.
	// CTF operations will use it if provided.

	// Validate private key format (should be hex string, typically 64 chars + optional 0x prefix)
	privateKey := strings.TrimPrefix(c.PrivateKey, "0x")
	if len(privateKey) < 64 || !isHexString(privateKey) {
		return fmt.Errorf("private_key must be a valid hex string (64+ hex characters)")
	}

	// Validate funder address format if provided (for Safe mode, it's auto-derived if empty)
	if c.PolymarketAddress != "" {
		if !strings.HasPrefix(c.PolymarketAddress, "0x") || len(c.PolymarketAddress) != 42 || !isHexString(c.PolymarketAddress[2:]) {
			return fmt.Errorf("polymarket_address must be a valid Ethereum address (0x followed by 40 hex characters)")
		}
	}

	// When signature_type is 2 (Safe) and on-chain operations are enabled, we'll automatically
	// derive the Safe address from the EOA and configure both CLOB and on-chain ops to use it.
	// If polymarket_address is not set, it will be derived deterministically.

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
