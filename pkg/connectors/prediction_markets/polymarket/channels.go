package polymarket

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

func (p *polymarket) GetTradesChannel() <-chan connector.Trade {
	return p.tradeChannel
}

func (p *polymarket) GetOrdersChannel() <-chan connector.Order {
	return p.orderChannel
}

func (p *polymarket) GetTradeUpdatesChannel() <-chan connector.Trade {
	return p.tradesChannel
}

func (p *polymarket) GetPriceChangeChannels() map[string]<-chan prediction.PriceChange {
	p.priceChangeMu.RLock()
	defer p.priceChangeMu.RUnlock()

	// Create a new map with read-only channels
	result := make(map[string]<-chan prediction.PriceChange, len(p.priceChangeChannels))
	for marketID, ch := range p.priceChangeChannels {
		result[marketID] = ch
	}

	return result
}

func (p *polymarket) GetOrderbookChannels() map[string]<-chan connector.OrderBook {
	p.orderBookMu.RLock()
	defer p.orderBookMu.RUnlock()

	// Create a new map with read-only channels
	result := make(map[string]<-chan connector.OrderBook, len(p.orderBookChannels))
	for marketID, ch := range p.orderBookChannels {
		result[marketID] = ch
	}

	return result
}

func (p *polymarket) ErrorChannel() <-chan error {
	//TODO implement me
	panic("implement me")
}
