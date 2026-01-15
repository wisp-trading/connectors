package hyperliquid

import (
	"fmt"
	"strconv"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
)

// FetchKlines retrieves historical candlestick data with decimal precision
func (h *hyperliquid) FetchKlines(symbol, interval string, limit int) ([]connector.Kline, error) {
	hlInterval := convertInterval(interval)
	endTime := h.timeProvider.Now().Unix()
	startTime := endTime - int64(limit*intervalToSeconds(hlInterval))

	candles, err := h.marketData.GetCandles(symbol, hlInterval, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles: %w", err)
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
			Symbol:    symbol,
			Interval:  interval,
			OpenTime:  time.Unix(candle.Time/1000, 0),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closeVal,
			Volume:    volume,
			CloseTime: time.Unix(candle.Timestamp/1000, 0),
		})
	}

	return klines, nil
}

// FetchPrice retrieves current price with decimal precision
func (h *hyperliquid) FetchPrice(symbol string) (*connector.Price, error) {
	mids, err := h.marketData.GetAllMids()
	if err != nil {
		return nil, fmt.Errorf("failed to get current prices: %w", err)
	}

	priceStr, exists := mids[symbol]
	if !exists {
		return nil, fmt.Errorf("price not found for symbol: %s", symbol)
	}

	price, err := numerical.NewFromString(priceStr)
	if err != nil {
		return nil, fmt.Errorf("invalid price format for %s: %w", symbol, err)
	}

	return &connector.Price{
		Symbol:    symbol,
		Price:     price,
		Source:    "Hyperliquid",
		Timestamp: h.timeProvider.Now(),
	}, nil
}

// FetchOrderBook retrieves order book with decimal precision
func (h *hyperliquid) FetchOrderBook(symbol portfolio.Asset, depth int) (*connector.OrderBook, error) {
	l2Book, err := h.marketData.GetL2Book(symbol.Symbol())

	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}

	orderBook := &connector.OrderBook{
		Asset:     symbol,
		Timestamp: h.timeProvider.Now(),
		Bids:      make([]connector.PriceLevel, 0, depth),
		Asks:      make([]connector.PriceLevel, 0, depth),
	}

	if l2Book.Levels == nil || len(l2Book.Levels) < 2 {
		return orderBook, nil
	}

	// Process bids (buy orders)
	for i, level := range l2Book.Levels[0] {
		if i >= depth {
			break
		}

		price := numerical.NewFromFloat(level.Px)
		quantity := numerical.NewFromFloat(level.Sz)

		orderBook.Bids = append(orderBook.Bids, connector.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Process asks (sell orders)
	for i, level := range l2Book.Levels[1] {
		if i >= depth {
			break
		}

		price := numerical.NewFromFloat(level.Px)
		quantity := numerical.NewFromFloat(level.Sz)

		orderBook.Asks = append(orderBook.Asks, connector.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	return orderBook, nil
}

func (h *hyperliquid) GetPerpSymbol(symbol portfolio.Asset) string {
	return symbol.Symbol()

}

// FetchRecentTrades retrieves recent trades for the specified symbol
func (h *hyperliquid) FetchRecentTrades(symbol string, limit int) ([]connector.Trade, error) {
	// Get user's fills (their own trades)
	fills, err := h.marketData.GetUserFills(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get user fills: %w", err)
	}

	// Filter by symbol and convert to connector.Trade
	trades := make([]connector.Trade, 0, limit)
	for i, fill := range fills {
		if i >= limit {
			break
		}

		// Only include trades for the requested symbol
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
			Symbol:    fill.Coin,
			Exchange:  types.Hyperliquid,
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

// FetchRiskFundBalance retrieves risk fund balance for the specified symbol
func (h *hyperliquid) FetchRiskFundBalance(symbol string) (*connector.RiskFundBalance, error) {
	return nil, fmt.Errorf("FetchRiskFundBalance not implemented for Hyperliquid")
}

// FetchContracts retrieves available contract information
func (h *hyperliquid) FetchContracts() ([]connector.ContractInfo, error) {
	return nil, fmt.Errorf("FetchContracts not implemented for Hyperliquid")
}
