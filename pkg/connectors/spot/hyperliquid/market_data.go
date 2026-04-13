package hyperliquid

import (
	"fmt"
	"strconv"
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
	hlInterval := convertInterval(interval)
	endTime := h.timeProvider.Now().Unix()
	startTime := endTime - int64(limit*intervalToSeconds(hlInterval))

	coin := h.normaliseAssetName(pair.Base())
	candles, err := h.marketData.GetCandles(coin, hlInterval, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles for %s: %w", coin, err)
	}

	klines := make([]connector.Kline, 0, len(candles))
	for _, candle := range candles {
		open, err := strconv.ParseFloat(candle.Open, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid open price: %w", err)
		}

		high, err := strconv.ParseFloat(candle.High, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid high price: %w", err)
		}

		low, err := strconv.ParseFloat(candle.Low, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid low price: %w", err)
		}

		closeVal, err := strconv.ParseFloat(candle.Close, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid close price: %w", err)
		}

		volume, err := strconv.ParseFloat(candle.Volume, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid volume: %w", err)
		}

		klines = append(klines, connector.Kline{
			Pair:      pair,
			Interval:  interval,
			OpenTime:  time.Unix(candle.TimeOpen/1000, 0),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closeVal,
			Volume:    volume,
			CloseTime: time.Unix(candle.TimeClose/1000, 0),
		})
	}

	return klines, nil
}

// FetchRecentTrades implements connector.MarketDataReader.
// Returns the user's recent fills for the requested spot pair.
func (h *hyperliquidSpot) FetchRecentTrades(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	fills, err := h.marketData.GetUserFills(h.effectiveAddress())
	if err != nil {
		return nil, fmt.Errorf("failed to get user fills: %w", err)
	}

	symbol := h.normaliseAssetName(pair.Base())
	trades := make([]connector.Trade, 0, limit)

	for _, fill := range fills {
		if fill.Coin != symbol {
			continue
		}

		price, err := numerical.NewFromString(fill.Price)
		if err != nil {
			h.appLogger.Warn("Invalid price in fill",
				"coin", fill.Coin,
				"price", fill.Price,
				"error", err)
			continue
		}

		quantity, err := numerical.NewFromString(fill.Size)
		if err != nil {
			h.appLogger.Warn("Invalid quantity in fill",
				"coin", fill.Coin,
				"size", fill.Size,
				"error", err)
			continue
		}

		trades = append(trades, connector.Trade{
			ID:        fmt.Sprintf("%d", fill.Oid),
			Pair:      pair,
			Exchange:  types.HyperliquidSpot,
			Price:     price,
			Quantity:  quantity,
			Side:      connector.FromString(fill.Side),
			Fee:       numerical.NewFromInt(0),
			Timestamp: time.Unix(fill.Time/1000, 0),
		})

		if len(trades) >= limit {
			break
		}
	}

	return trades, nil
}
