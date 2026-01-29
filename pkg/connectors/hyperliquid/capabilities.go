package hyperliquid

import (
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// SupportsTradingOperations returns whether trading operations are supported
func (h *hyperliquid) SupportsTradingOperations() bool {
	return h.trading != nil
}

// SupportsRealTimeData returns whether real-time data is supported
func (h *hyperliquid) SupportsRealTimeData() bool {
	return true
}

// SupportsHistoricalData returns whether historical data is supported
func (h *hyperliquid) SupportsHistoricalData() bool {
	return h.marketData != nil
}

func (h *hyperliquid) SupportsPerpetuals() bool {
	return true
}

func (h *hyperliquid) SupportsSpot() bool {
	return false
}

// GetConnectorInfo returns metadata about the exchange
func (h *hyperliquid) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             types.Hyperliquid,
		TradingEnabled:   h.SupportsTradingOperations(),
		WebSocketEnabled: true,
		MaxLeverage:      numerical.NewFromFloat(50.0),
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USD",
	}
}
