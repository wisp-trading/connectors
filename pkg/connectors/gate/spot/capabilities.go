package spot

import (
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// SupportsTradingOperations returns whether trading operations are supported
func (g *gateSpot) SupportsTradingOperations() bool {
	return true
}

// SupportsRealTimeData returns whether real-time data is supported
func (g *gateSpot) SupportsRealTimeData() bool {
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
