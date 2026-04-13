package hyperliquid

import (
	"fmt"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// FetchPrice implements connector.MarketDataReader.
// Uses the L2 order book to derive a mid-price for the spot asset.
func (h *hyperliquidSpot) FetchPrice(pair portfolio.Pair) (*connector.Price, error) {
	coin := h.normaliseAssetName(pair.Base())

	l2, err := h.marketData.FetchL2Book(coin)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 book for price: %w", err)
	}

	if len(l2.Levels) < 2 || len(l2.Levels[0]) == 0 || len(l2.Levels[1]) == 0 {
		return nil, fmt.Errorf("insufficient order book levels for %s", coin)
	}

	bestBid := l2.Levels[0][0].Px
	bestAsk := l2.Levels[1][0].Px
	midPx := (bestBid + bestAsk) / 2.0

	return &connector.Price{
		Pair:      pair,
		Price:     numerical.NewFromFloat(midPx),
		Source:    types.HyperliquidSpot,
		Timestamp: time.Now(),
	}, nil
}

// FetchOrderBook implements connector.MarketDataReader
func (h *hyperliquidSpot) FetchOrderBook(pair portfolio.Pair, depth int) (*connector.OrderBook, error) {
	coin := h.normaliseAssetName(pair.Base())

	l2, err := h.marketData.FetchL2Book(coin)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch L2 book: %w", err)
	}

	bids := make([]connector.PriceLevel, 0, depth)
	if len(l2.Levels) > 0 {
		for _, level := range l2.Levels[0] {
			bids = append(bids, connector.PriceLevel{
				Price:    numerical.NewFromFloat(level.Px),
				Quantity: numerical.NewFromFloat(level.Sz),
			})
			if len(bids) >= depth {
				break
			}
		}
	}

	asks := make([]connector.PriceLevel, 0, depth)
	if len(l2.Levels) > 1 {
		for _, level := range l2.Levels[1] {
			asks = append(asks, connector.PriceLevel{
				Price:    numerical.NewFromFloat(level.Px),
				Quantity: numerical.NewFromFloat(level.Sz),
			})
			if len(asks) >= depth {
				break
			}
		}
	}

	return &connector.OrderBook{
		Pair:      pair,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}, nil
}

// FetchKlines implements connector.MarketDataReader
func (h *hyperliquidSpot) FetchKlines(pair portfolio.Pair, interval string, limit int) ([]connector.Kline, error) {
	return nil, fmt.Errorf("FetchKlines not yet implemented for Hyperliquid spot")
}

// FetchRecentTrades implements connector.MarketDataReader
func (h *hyperliquidSpot) FetchRecentTrades(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	return nil, fmt.Errorf("FetchRecentTrades not yet implemented for Hyperliquid spot")
}
