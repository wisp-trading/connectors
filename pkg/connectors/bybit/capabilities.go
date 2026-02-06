package bybit

import (
	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// SupportsTradingOperations returns whether trading operations are supported
func (b *bybit) SupportsTradingOperations() bool {
	return b.trading != nil
}

// SupportsRealTimeData returns whether real-time data is supported
func (b *bybit) SupportsRealTimeData() bool {
	return true
}

// GetConnectorInfo returns metadata about the exchange
func (b *bybit) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             types.Bybit,
		TradingEnabled:   b.SupportsTradingOperations(),
		WebSocketEnabled: true,
		MaxLeverage:      numerical.NewFromFloat(125.0),
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USDT",
	}
}
