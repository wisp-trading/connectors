package polymarket

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

func (p *polymarket) StartWebSocket() error {
	if !p.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if p.wsClient == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	// Pass the WebSocket URL from config
	return p.wsClient.Connect(p.config.WebSocketURL)
}

func (p *polymarket) StopWebSocket() error {
	if p.wsClient == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	return p.wsClient.Disconnect()
}

func (p *polymarket) IsWebSocketConnected() bool {
	if p.wsClient == nil {
		return false
	}

	return p.wsClient.IsConnected()
}

func (p *polymarket) ErrorChannel() <-chan error {
	//TODO implement me
	panic("implement me")
}
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

	p.priceChangeMu.Lock()
	if _, exists := p.priceChangeChannels[market.Slug]; !exists {
		p.priceChangeChannels[market.Slug] = make(chan []prediction.PriceChange, 100)
		p.appLogger.Info("Created price change channel for market %s", market.Slug)
	}
	priceChangeChannel := p.priceChangeChannels[market.Slug]
	p.priceChangeMu.Unlock()

	// Single subscription with callback that handles ALL outcomes
	p.wsClient.SubscribeToMarket(
		market,
		func(msg *websocket.OrderBookMessage) {
			orderBook := convertToOrderBook(msg)

			select {
			case orderBookChannel <- orderBook:
				// Successfully sent
			default:
				p.appLogger.Warn("Order book channel full for market %s, dropping message", market.Slug)
			}
		},
		func(msg *websocket.PriceChanges) {
			priceChange := convertToPriceChange(msg)

			select {
			case priceChangeChannel <- priceChange:
				// Successfully sent
			default:
				p.appLogger.Warn("Price change channel full for market %s, dropping message", market.Slug)
			}
		},
	)

	p.appLogger.Info("Subscribed to order book for market %s with %d outcomes", market.Slug, len(market.Outcomes))
	return nil
}

func (p *polymarket) UnsubscribeOrderbook(market prediction.Market) error {
	p.orderBookMu.Lock()
	ch, exists := p.orderBookChannels[market.Slug]
	if exists {
		close(ch)
		delete(p.orderBookChannels, market.Slug)
	}
	p.orderBookMu.Unlock()

	if !exists {
		p.appLogger.Warn("Market %s not subscribed", market.Slug)
		return nil
	}

	p.wsClient.UnsubscribeFromMarket(market)
	p.appLogger.Info("Unsubscribed from market %s", market.Slug)

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
