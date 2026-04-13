package hyperliquid

import (
	"time"

	"github.com/wisp-trading/connectors/pkg/websocket/security"
)

// HyperliquidWSConfig holds Hyperliquid-specific WebSocket configuration
type HyperliquidWSConfig struct {
	WSURL             string
	ReconnectDelay    time.Duration
	MaxReconnects     int
	PingInterval      time.Duration
	PongTimeout       time.Duration
	MaxMessageSize    int
	RateLimitCapacity int
	RateLimitRefill   time.Duration
}

// DefaultWSConfig returns sensible Hyperliquid WebSocket defaults
func DefaultWSConfig() HyperliquidWSConfig {
	return HyperliquidWSConfig{
		WSURL:             "wss://api.hyperliquid.xyz/ws",
		ReconnectDelay:    5 * time.Second,
		MaxReconnects:     10,
		PingInterval:      30 * time.Second,
		PongTimeout:       10 * time.Second,
		MaxMessageSize:    1024 * 1024,
		RateLimitCapacity: 10000,
		RateLimitRefill:   time.Second,
	}
}

// ValidationConfig returns Hyperliquid's specific validation rules
func (c HyperliquidWSConfig) ValidationConfig() security.ValidationConfig {
	return security.ValidationConfig{
		MaxMessageSize: c.MaxMessageSize,
		TypeField:      "channel",
		AllowedTypes: map[string]bool{
			"l2Book":   true,
			"candle":   true,
			"trades":   true,
			"webData2": true,
		},
	}
}
