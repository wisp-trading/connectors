package polymarket

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (p *polymarket) GetTradesChannel() <-chan connector.Trade {
	return p.tradeChannel
}

func (p *polymarket) GetOrdersChannel() <-chan connector.Order {
	return p.orderChannel
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

func (p *polymarket) GetOrderBookUpdates() <-chan prediction.OrderBook {
	return p.orderBookChannel
}

func (p *polymarket) ErrorChannel() <-chan error {
	return p.errChannel
}
