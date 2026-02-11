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

	if err := market.Validate(); err != nil {
		return fmt.Errorf("invalid market: %w", err)
	}

	// Subscribe to each outcome as a separate pair
	for _, outcome := range market.Outcomes {
		pairSymbol := outcome.Pair.Symbol()

		p.orderBookMu.Lock()
		if _, exists := p.orderBookChannels[pairSymbol]; !exists {
			p.orderBookChannels[pairSymbol] = make(chan connector.OrderBook, 100)
			p.appLogger.Info("Created order book channel for %s", pairSymbol)
		}
		ch := p.orderBookChannels[pairSymbol]
		p.orderBookMu.Unlock()

		// Register callback for this outcome
		p.wsClient.SubscribeToMarketBook(outcome.OutcomeId, func(msg *websocket.OrderBookMessage) {
			orderBook := convertToOrderBook(msg)

			select {
			case ch <- orderBook:
				// Successfully sent
			default:
				p.appLogger.Warn("Order book channel full for %s, dropping message", pairSymbol)
			}
		})

		p.appLogger.Info("Subscribed to order book for %s (outcome ID: %s)", pairSymbol, outcome.OutcomeId)
	}

	return nil
}

func (p *polymarket) UnsubscribeOrderbook(market prediction.Market) error {
	for _, outcome := range market.Outcomes {
		pairSymbol := outcome.Pair.Symbol()

		p.orderBookMu.Lock()
		ch, exists := p.orderBookChannels[pairSymbol]
		if exists {
			close(ch)
			delete(p.orderBookChannels, pairSymbol)
		}
		p.orderBookMu.Unlock()

		if !exists {
			p.appLogger.Warn("Pair %s not subscribed", pairSymbol)
			continue
		}

		p.wsClient.UnsubscribeFromMarketBook(outcome.OutcomeId)
		p.appLogger.Info("Unsubscribed from %s", pairSymbol)
	}

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
