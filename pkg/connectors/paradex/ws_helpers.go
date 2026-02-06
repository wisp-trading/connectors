package paradex

import (
	padexmodel "github.com/trishtzy/go-paradex/models"
	"github.com/wisp-trading/connectors/pkg/connectors/paradex/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *paradex) convertOrderBookUpdates(output chan<- connector.OrderBook) {
	defer close(output)

	for wsUpdate := range p.wsService.OrderbookUpdates() {
		pair, err := p.PerpSymbolToPair(wsUpdate.Symbol)
		if err != nil {
			p.appLogger.Error("Failed to convert symbol to pair: %v", err)
			continue
		}

		connectorOrderBook := connector.OrderBook{
			Pair:      pair,
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
		pair, err := p.PerpSymbolToPair(wsUpdate.Symbol)
		if err != nil {
			p.appLogger.Error("Failed to convert symbol to pair: %v", err)
			continue
		}
		
		side := padexmodel.ResponsesOrderSide(wsUpdate.Side)

		connectorTrade := connector.Trade{
			ID:        wsUpdate.TradeID,
			Pair:      pair,
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
