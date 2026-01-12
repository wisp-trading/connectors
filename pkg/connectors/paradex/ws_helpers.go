package paradex

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/live-trading/pkg/connectors/paradex/websocket"
	padexmodel "github.com/trishtzy/go-paradex/models"

	"strings"
)

func (p *paradex) convertOrderBookUpdates(output chan<- connector.OrderBook) {
	defer close(output)

	for wsUpdate := range p.wsService.OrderbookUpdates() {
		asset := p.parseAssetFromSymbol(wsUpdate.Symbol)

		connectorOrderBook := connector.OrderBook{
			Asset:     asset,
			Bids:      p.convertWSPriceLevels(wsUpdate.Bids),
			Asks:      p.convertWSPriceLevels(wsUpdate.Asks),
			Timestamp: wsUpdate.Timestamp,
		}

		select {
		case output <- connectorOrderBook:
		case <-p.wsContext.Done():
			return
		}
	}
}

func (p *paradex) convertWSPriceLevels(wsLevels []websockets.PriceLevel) []connector.PriceLevel {
	result := make([]connector.PriceLevel, len(wsLevels))
	for i, wsLevel := range wsLevels {
		result[i] = connector.PriceLevel{
			Price:    numerical.NewFromFloat(wsLevel.Price),
			Quantity: numerical.NewFromFloat(wsLevel.Quantity),
		}
	}
	return result
}

func (p *paradex) convertTradeUpdates(output chan<- connector.Trade) {
	defer close(output)

	for wsUpdate := range p.wsService.TradeUpdates() {
		side := padexmodel.ResponsesOrderSide(wsUpdate.Side)

		connectorTrade := connector.Trade{
			ID:        wsUpdate.TradeID,
			Symbol:    wsUpdate.Symbol,
			Price:     numerical.NewFromFloat(wsUpdate.Price),
			Quantity:  numerical.NewFromFloat(wsUpdate.Quantity),
			Side:      p.convertOrderSide(side),
			IsMaker:   false,            // paradex doesn't provide this in trade updates
			Fee:       numerical.Zero(), // Not available in WebSocket updates
			Timestamp: wsUpdate.Timestamp,
		}

		select {
		case output <- connectorTrade:
		case <-p.wsContext.Done():
			return
		}
	}
}

func (p *paradex) parseAssetFromSymbol(symbol string) portfolio.Asset {
	// Parse symbols like "BTC-USD-PERP" to extract "BTC"
	parts := strings.Split(symbol, "-")
	if len(parts) > 0 {
		return portfolio.NewAsset(parts[0])
	}
	return portfolio.NewAsset(symbol)
}
