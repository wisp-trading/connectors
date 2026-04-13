package hyperliquid

import (
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

// SupportsTradingOperations returns whether trading operations are supported
func (h *hyperliquidSpot) SupportsTradingOperations() bool {
	return h.trading != nil
}

// SupportsRealTimeData returns whether real-time data is supported
func (h *hyperliquidSpot) SupportsRealTimeData() bool {
	return true
}

// GetConnectorInfo returns metadata about the exchange
func (h *hyperliquidSpot) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             types.HyperliquidSpot,
		TradingEnabled:   h.SupportsTradingOperations(),
		WebSocketEnabled: true,
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USDC",
	}
}

// GetSpotSymbol returns the Hyperliquid symbol for a spot pair.
// On Hyperliquid, spot pairs use the format "@N" (asset index based),
// but the API accepts the token name directly (e.g. "HYPE", "PURR").
func (h *hyperliquidSpot) GetSpotSymbol(pair portfolio.Pair) string {
	return h.normaliseAssetName(pair.Base())
}
