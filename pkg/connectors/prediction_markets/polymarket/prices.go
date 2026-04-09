package polymarket

import (
	"context"
	"fmt"

	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

func (p *polymarket) SubscribePriceChanges(market prediction.Market) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("websocket not connected")
	}

	if err := market.Validate(); err != nil {
		return fmt.Errorf("invalid market: %w", err)
	}

	p.priceChangeMu.Lock()
	if _, exists := p.priceChangeChannels[market.Slug]; exists {
		p.priceChangeMu.Unlock()
		p.appLogger.Info("Already subscribed to price changes for market %s", market.Slug)
		return nil
	}

	p.priceChangeChannels[market.Slug] = make(chan prediction.PriceChange, 100)
	priceChangeChannel := p.priceChangeChannels[market.Slug]
	p.appLogger.Info("Created price change channel for market %s", market.Slug)
	p.priceChangeMu.Unlock()

	ctx := context.Background()
	stream, err := p.clobWebsocket.SubscribePrices(ctx, market)
	if err != nil {
		p.priceChangeMu.Lock()
		delete(p.priceChangeChannels, market.Slug)
		p.priceChangeMu.Unlock()
		return fmt.Errorf("failed to subscribe to prices: %w", err)
	}

	go func() {
		defer func() {
			p.priceChangeMu.Lock()
			delete(p.priceChangeChannels, market.Slug)
			close(priceChangeChannel)
			p.priceChangeMu.Unlock()
			p.appLogger.Info("Unsubscribed from price changes for market %s", market.Slug)
		}()

		for msg := range stream {
			priceChange, err := p.parsePriceChange(msg, market)
			if err != nil {
				p.appLogger.Error("Error converting price change for market %s: %v", market.Slug, err)
				continue
			}

			select {
			case priceChangeChannel <- priceChange:
				// Successfully sent
			default:
				p.appLogger.Warn("Price change channel full for market %s, dropping message", market.Slug)
			}
		}
	}()

	p.appLogger.Info("Subscribed to price changes for market %s", market.Slug)
	return nil
}

func (p *polymarket) FetchPrice(pair portfolio.Pair) (*connector.Price, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) FetchKlines(pair portfolio.Pair, interval string, limit int) ([]connector.Kline, error) {
	//TODO implement me
	panic("implement me")
}
