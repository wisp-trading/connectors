package polymarket

import (
	"context"
	"fmt"

	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (p *polymarket) SubscribeOrderBook(market prediction.Market) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("websocket not connected")
	}

	if err := market.Validate(); err != nil {
		return fmt.Errorf("invalid market: %w", err)
	}

	if _, exists := p.subscribedMarkets[market.MarketID]; exists {
		return fmt.Errorf("already subscribed to market %s", market.Slug)
	}

	// Get channel from SDK wrapper
	msgChannel, err := p.clobWebsocket.SubscribeOrderbook(market)
	if err != nil {
		return fmt.Errorf("failed to subscribe to orderbook: %w", err)
	}

	// Convert messages in a goroutine
	go func() {
		for msg := range msgChannel {
			orderBook := p.parseOrderbookEvent(msg, market)

			select {
			case p.orderBookChannel <- orderBook:
				// Successfully sent
			default:
				p.appLogger.Warn("Order book channel full for market %s, dropping message", market.Slug)
			}
		}
	}()

	p.appLogger.Info("Subscribed to order book for market %s with %d outcomes", market.Slug, len(market.Outcomes))
	return nil
}

func (p *polymarket) FetchOrderBooks(
	market prediction.Market,
	outcome prediction.Outcome,
) (*prediction.OrderBook, error) {
	ctx := context.Background()
	orderbook, err := p.orderManager.GetOrderBook(ctx, outcome)
	if err != nil {
		return nil, fmt.Errorf("failed to get market: %w", err)
	}

	orderBook := p.parseOrderbook(orderbook, market, outcome)
	return &orderBook, nil
}
