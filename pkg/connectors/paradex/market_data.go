package paradex

import (
	"fmt"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
)

func (p *paradex) FetchPrice(symbol string) (*connector.Price, error) {
	price, err := p.paradexService.GetPrice(p.ctx, symbol)

	if err != nil {
		p.appLogger.Error("Failed to fetch price", "symbol", symbol, "error", err)
		return nil, fmt.Errorf("failed to fetch price for %s: %w", symbol, err)
	}

	if price == nil {
		p.appLogger.Error("Price not found", "symbol", symbol)
		return nil, fmt.Errorf("price not found for %s", symbol)
	}

	priceValue, err := numerical.NewFromString(price.Bid)
	if err != nil {
		p.appLogger.Error("Invalid price format", "symbol", symbol, "price", price.Bid, "error", err)
		return nil, fmt.Errorf("invalid price format for %s: %w", symbol, err)
	}

	return &connector.Price{
		Symbol:    symbol,
		Price:     priceValue,
		BidPrice:  priceValue,
		AskPrice:  priceValue,
		Volume24h: numerical.Zero(), // Volume not provided by paradex
		Change24h: numerical.Zero(), // Change not provided by paradex
		Source:    p.GetConnectorInfo().Name,
		Timestamp: time.Now(), // Use current time as paradex does not provide timestamp
	}, nil

}

func (p *paradex) FetchOrderBook(symbol portfolio.Asset, instrument connector.Instrument, depth int) (*connector.OrderBook, error) {
	if instrument != connector.TypePerpetual {
		return nil, fmt.Errorf("order book only supported for perpetual contracts")
	}

	symbolStr := p.GetPerpSymbol(symbol)
	depthInt := int64(depth)

	orderBook, err := p.paradexService.GetOrderBook(p.ctx, symbolStr, &depthInt)
	if err != nil {
		return nil, err
	}

	// Convert bids - each bid is []string where [0] is price, [1] is size
	bids := make([]connector.PriceLevel, len(orderBook.Bids))
	for i, bid := range orderBook.Bids {
		if len(bid) >= 2 {
			price, _ := numerical.NewFromString(bid[0]) // First element is price
			size, _ := numerical.NewFromString(bid[1])  // Second element is size
			bids[i] = connector.PriceLevel{Price: price, Quantity: size}
		}
	}

	// Convert asks - each ask is []string where [0] is price, [1] is size
	asks := make([]connector.PriceLevel, len(orderBook.Asks))
	for i, ask := range orderBook.Asks {
		if len(ask) >= 2 {
			price, _ := numerical.NewFromString(ask[0]) // First element is price
			size, _ := numerical.NewFromString(ask[1])  // Second element is size
			asks[i] = connector.PriceLevel{Price: price, Quantity: size}
		}
	}

	// Use the actual timestamp from paradex if available
	timestamp := time.Now()
	if orderBook.LastUpdatedAt > 0 {
		timestamp = time.UnixMilli(orderBook.LastUpdatedAt)
	}

	return &connector.OrderBook{
		Asset:     symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: timestamp,
	}, nil
}

func (p *paradex) FetchRecentTrades(symbol string, limit int) ([]connector.Trade, error) {
	return nil, fmt.Errorf("klines not needed for MM strategy")
}

func (p *paradex) FetchKlines(symbol, interval string, limit int) ([]connector.Kline, error) {
	// Convert interval string (e.g., "5m", "1h") to resolution in minutes
	resolution, err := parseIntervalToMinutes(interval)
	if err != nil {
		return nil, fmt.Errorf("invalid interval %s: %w", interval, err)
	}

	// Calculate time range: get last N klines based on resolution
	endTime := time.Now()
	duration := time.Duration(limit*resolution) * time.Minute
	startTime := endTime.Add(-duration)

	// Convert to milliseconds
	startMs := startTime.UnixMilli()
	endMs := endTime.UnixMilli()

	// Fetch klines from paradex
	klineData, err := p.paradexService.GetKlines(p.ctx, symbol, resolution, startMs, endMs)
	if err != nil {
		p.appLogger.Error("Failed to fetch klines", "symbol", symbol, "interval", interval, "error", err)
		return nil, fmt.Errorf("failed to fetch klines for %s: %w", symbol, err)
	}

	// Convert to connector.Kline format
	klines := make([]connector.Kline, 0, len(klineData))
	for _, k := range klineData {
		klines = append(klines, connector.Kline{
			Symbol:   symbol,
			Interval: interval,
			OpenTime: time.UnixMilli(k.Timestamp),
			Open:     k.Open,
			High:     k.High,
			Low:      k.Low,
			Close:    k.Close,
			Volume:   k.Volume,
		})
	}

	return klines, nil
}

// parseIntervalToMinutes converts interval string to minutes
// Supports: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, etc.
func parseIntervalToMinutes(interval string) (int, error) {
	if len(interval) < 2 {
		return 0, fmt.Errorf("interval too short")
	}

	unit := interval[len(interval)-1]
	valueStr := interval[:len(interval)-1]

	var value int
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return 0, fmt.Errorf("invalid interval value: %w", err)
	}

	switch unit {
	case 'm', 'M':
		return value, nil
	case 'h', 'H':
		return value * 60, nil
	default:
		return 0, fmt.Errorf("unsupported interval unit: %c", unit)
	}
}

func (p *paradex) FetchRiskFundBalance(symbol string) (*connector.RiskFundBalance, error) {
	return nil, fmt.Errorf("risk fund balance not needed for MM strategy")
}

func (p *paradex) FetchContracts() ([]connector.ContractInfo, error) {
	return nil, fmt.Errorf("contracts not needed for MM strategy")
}
