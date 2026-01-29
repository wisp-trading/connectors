package paradex

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *paradex) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             types.Paradex,
		TradingEnabled:   p.SupportsTradingOperations(),
		WebSocketEnabled: p.SupportsRealTimeData(),
		MaxLeverage:      numerical.NewFromFloat(10.0),
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USD",
	}
}

func (p *paradex) GetPerpSymbol(symbol portfolio.Asset) string {
	return fmt.Sprintf("%s-USD-PERP", symbol.Symbol())
}

func (p *paradex) SupportsTradingOperations() bool {
	return true
}

func (p *paradex) SupportsRealTimeData() bool {
	return true
}
