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
		QuoteCurrency: "USDC",
	}
}

func (p *paradex) GetPerpSymbol(pair portfolio.Pair) string {
	return fmt.Sprintf("%s-USD-PERP", pair.Base().Symbol())
}

func (p *paradex) PerpSymbolToPair(symbol string) (portfolio.Pair, error) {
	var base string

	_, err := fmt.Sscanf(symbol, "%s-USD-PERP", &base)
	if err != nil {
		return portfolio.Pair{}, fmt.Errorf("invalid perp symbol format: %w", err)
	}

	return portfolio.NewPair(
		portfolio.NewAsset(base),
		portfolio.NewAsset("USDC"),
	), nil
}

func (p *paradex) SupportsTradingOperations() bool {
	return true
}

func (p *paradex) SupportsRealTimeData() bool {
	return true
}
