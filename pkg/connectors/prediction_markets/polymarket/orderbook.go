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
		p.appLogger.Info("Already subscribed to order book for market %s", market.Slug)
		return nil
	}

	// Get channel from SDK wrapper
	ctx := context.Background()
	msgChannel, err := p.clobWebsocket.SubscribeOrderbook(ctx, market)
	if err != nil {
		return fmt.Errorf("failed to subscribe to orderbook: %w", err)
	}

	// Mark market as subscribed
	p.subscribedMarkets[market.MarketID] = market

	// Convert messages in a goroutine
	go func() {
		defer func() {
			delete(p.subscribedMarkets, market.MarketID)
			p.appLogger.Info("Unsubscribed from order book for market %s", market.Slug)
		}()

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

func (p *polymarket) FetchOrderBooksForMarket(market prediction.Market) (map[string]*prediction.OrderBook, error) {
	ctx := context.Background()
	books, err := p.orderManager.GetOrderBooks(ctx, market.Outcomes)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch orderbooks for market %s: %w", market.MarketID, err)
	}

	result := make(map[string]*prediction.OrderBook, len(books))
	for i, book := range books {
		if i < len(market.Outcomes) {
			outcome := market.Outcomes[i]
			parsed := p.parseOrderbook(book, market, outcome)
			result[outcome.OutcomeID.String()] = &parsed
		}
	}

	return result, nil
}
