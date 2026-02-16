package polymarket

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

func (p *polymarket) SubscribeOrderBook(market prediction.Market) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("websocket not connected")
	}

	if err := market.Validate(); err != nil {
		return fmt.Errorf("invalid market: %w", err)
	}

	p.orderBookMu.Lock()
	if _, exists := p.orderBookChannels[market.Slug]; !exists {
		p.orderBookChannels[market.Slug] = make(chan connector.OrderBook, 100)
		p.appLogger.Info("Created order book channel for market %s", market.Slug)
	}
	orderBookChannel := p.orderBookChannels[market.Slug]
	p.orderBookMu.Unlock()

	// Get channel from SDK wrapper
	msgChannel, err := p.clobWebsocket.SubscribeOrderbook(market)
	if err != nil {
		return fmt.Errorf("failed to subscribe to orderbook: %w", err)
	}

	// Convert messages in a goroutine
	go func() {
		for msg := range msgChannel {
			orderBook := p.convertToOrderBook(msg, market)

			select {
			case orderBookChannel <- orderBook:
				// Successfully sent
			default:
				p.appLogger.Warn("Order book channel full for market %s, dropping message", market.Slug)
			}
		}
	}()

	p.appLogger.Info("Subscribed to order book for market %s with %d outcomes", market.Slug, len(market.Outcomes))
	return nil
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

func (p *polymarket) FetchOrderBook(pair portfolio.Pair, depth int) (*connector.OrderBook, error) {
	//TODO implement me
	panic("implement me")
}
