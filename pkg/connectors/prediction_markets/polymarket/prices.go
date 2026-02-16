package polymarket

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
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
	if _, exists := p.priceChangeChannels[market.Slug]; !exists {
		p.priceChangeChannels[market.Slug] = make(chan prediction.PriceChange, 100)
		p.appLogger.Info("Created price change channel for market %s", market.Slug)
	}
	priceChangeChannel := p.priceChangeChannels[market.Slug]
	p.priceChangeMu.Unlock()

	stream, err := p.clobWebsocket.SubscribePriceChanges(market)
	if err != nil {
		return fmt.Errorf("failed to subscribe to price changes: %w", err)
	}

	go func() {
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

func (p *polymarket) FetchPrice(pair portfolio.Pair) (*connector.Price, error) {
	//TODO implement me
	panic("implement me")
}

func (p *polymarket) FetchKlines(pair portfolio.Pair, interval string, limit int) ([]connector.Kline, error) {
	//TODO implement me
	panic("implement me")
}
