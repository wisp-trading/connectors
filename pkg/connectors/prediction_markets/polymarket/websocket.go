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
	//TODO implement me
	panic("implement me")
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

	// Create channel for this market if it doesn't exist
	p.orderBookMu.Lock()
	if _, exists := p.orderBookChannels[market.MarketId]; !exists {
		p.orderBookChannels[market.MarketId] = make(chan connector.OrderBook, 100)
		p.appLogger.Info("Created order book channel for market %s", market.MarketId)
	}
	ch := p.orderBookChannels[market.MarketId]
	p.orderBookMu.Unlock()

	// Register callback with WebSocket service
	p.wsClient.SubscribeToMarketBook(market.MarketId, func(msg *websocket.OrderBookMessage) {
		orderBook := convertToOrderBook(msg)

		// Send to channel with non-blocking write
		select {
		case ch <- orderBook:
			// Successfully sent
		default:
			p.appLogger.Warn("Order book channel full for market %s, dropping message", market.MarketId)
		}
	})

	p.appLogger.Info("Subscribed to market book for market %s", market.MarketId)
	return nil
}

func (p *polymarket) UnsubscribeOrderbook(market prediction.Market) error {
	// Remove channel and close it
	p.orderBookMu.Lock()
	ch, exists := p.orderBookChannels[market.MarketId]
	if exists {
		close(ch)
		delete(p.orderBookChannels, market.MarketId)
	}
	p.orderBookMu.Unlock()

	if !exists {
		return fmt.Errorf("market %s not subscribed", market.MarketId)
	}

	// Unsubscribe from WebSocket service
	p.wsClient.UnsubscribeFromMarketBook(market.MarketId)
	p.appLogger.Info("Unsubscribed from market book for market %s", market.MarketId)

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
