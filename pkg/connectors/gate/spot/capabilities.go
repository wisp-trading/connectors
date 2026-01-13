package spot

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
)

// SupportsTradingOperations returns whether trading operations are supported
func (g *gateSpot) SupportsTradingOperations() bool {
	return true
}

// SupportsRealTimeData returns whether real-time data is supported
func (g *gateSpot) SupportsRealTimeData() bool {
	return true
}

// SupportsHistoricalData returns whether historical data is supported
func (g *gateSpot) SupportsHistoricalData() bool {
	return true
}

// SupportsPerpetuals returns whether perpetual contracts are supported
func (g *gateSpot) SupportsPerpetuals() bool {
	return false
}

// SupportsSpot returns whether spot trading is supported
func (g *gateSpot) SupportsSpot() bool {
	return true
}

// SupportsWebSocket returns whether WebSocket is supported
func (g *gateSpot) SupportsWebSocket() bool {
	return true
}

// GetConnectorInfo returns metadata about the exchange
func (g *gateSpot) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             types.GateSpot,
		TradingEnabled:   g.SupportsTradingOperations(),
		WebSocketEnabled: true,
		MaxLeverage:      numerical.NewFromFloat(0),
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USDT",
	}
}
