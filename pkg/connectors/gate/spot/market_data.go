package spot

import (
	"fmt"
	"strconv"
	"time"

	"github.com/antihax/optional"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
	"github.com/gate/gateapi-go/v7"
)

// FetchKlines retrieves historical candlestick data
func (g *gateSpot) FetchKlines(symbol, interval string, limit int) ([]connector.Kline, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()
	currencyPair := g.formatSymbol(symbol)

	// Convert interval format (1m -> 1m, 5m -> 5m, 1h -> 1h, etc.)
	gateInterval := convertInterval(interval)

	// Debug logging
	g.appLogger.Info("Fetching klines",
		"currencyPair", currencyPair,
		"interval", gateInterval,
		"limit", limit)

	// Get candlesticks
	candles, resp, err := client.SpotApi.ListCandlesticks(ctx, currencyPair, &gateapi.ListCandlesticksOpts{
		Limit:    optional.NewInt32(int32(limit)),
		Interval: optional.NewString(gateInterval),
	})
	if err != nil {
		g.appLogger.Error("Failed to fetch candles",
			"error", err,
			"statusCode", resp.StatusCode,
			"currencyPair", currencyPair)
		return nil, fmt.Errorf("failed to fetch candles: %w", err)
	}

	klines := make([]connector.Kline, 0, len(candles))
	for _, candle := range candles {
		timestamp, _ := strconv.ParseInt(candle[0], 10, 64)
		openTime := time.Unix(timestamp, 0)

		volume, _ := strconv.ParseFloat(candle[1], 64)
		closePrice, _ := strconv.ParseFloat(candle[2], 64)
		high, _ := strconv.ParseFloat(candle[3], 64)
		low, _ := strconv.ParseFloat(candle[4], 64)
		open, _ := strconv.ParseFloat(candle[5], 64)

		klines = append(klines, connector.Kline{
			Symbol:    symbol,
			Interval:  interval,
			OpenTime:  openTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
			CloseTime: openTime,
		})
	}

	return klines, nil
}

// FetchPrice retrieves current price
func (g *gateSpot) FetchPrice(symbol string) (*connector.Price, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()
	currencyPair := g.formatSymbol(symbol)

	// Get ticker for the currency pair
	tickers, _, err := client.SpotApi.ListTickers(ctx, &gateapi.ListTickersOpts{
		CurrencyPair: optional.NewString(currencyPair),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ticker: %w", err)
	}

	if len(tickers) == 0 {
		return nil, fmt.Errorf("no ticker found for %s", symbol)
	}

	price, err := numerical.NewFromString(tickers[0].Last)
	if err != nil {
		return nil, fmt.Errorf("invalid price format: %w", err)
	}

	return &connector.Price{
		Symbol:    symbol,
		Price:     price,
		Source:    types.GateSpot,
		Timestamp: g.timeProvider.Now(),
	}, nil
}

// FetchOrderBook retrieves order book
func (g *gateSpot) FetchOrderBook(symbol portfolio.Asset, instrument connector.Instrument, depth int) (*connector.OrderBook, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()
	currencyPair := g.formatSymbol(symbol.Symbol())

	orderBook, _, err := client.SpotApi.ListOrderBook(ctx, currencyPair, &gateapi.ListOrderBookOpts{
		Limit: optional.NewInt32(int32(depth)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order book: %w", err)
	}

	// Convert bids
	bids := make([]connector.PriceLevel, 0, len(orderBook.Bids))
	for _, bid := range orderBook.Bids {
		price, _ := numerical.NewFromString(bid[0])
		quantity, _ := numerical.NewFromString(bid[1])
		bids = append(bids, connector.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Convert asks
	asks := make([]connector.PriceLevel, 0, len(orderBook.Asks))
	for _, ask := range orderBook.Asks {
		price, _ := numerical.NewFromString(ask[0])
		quantity, _ := numerical.NewFromString(ask[1])
		asks = append(asks, connector.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	return &connector.OrderBook{
		Bids:      bids,
		Asks:      asks,
		Timestamp: g.timeProvider.Now(),
	}, nil
}

// FetchRecentTrades retrieves recent public trades for a symbol
func (g *gateSpot) FetchRecentTrades(symbol string, limit int) ([]connector.Trade, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()
	currencyPair := g.formatSymbol(symbol)

	// Fetch public trades
	gateTrades, _, err := client.SpotApi.ListTrades(ctx, currencyPair, &gateapi.ListTradesOpts{
		Limit: optional.NewInt32(int32(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trades: %w", err)
	}

	trades := make([]connector.Trade, 0, len(gateTrades))
	for _, trade := range gateTrades {
		price, err := numerical.NewFromString(trade.Price)
		if err != nil {
			g.appLogger.Warn("Invalid price in trade", "id", trade.Id, "price", trade.Price)
			continue
		}

		quantity, err := numerical.NewFromString(trade.Amount)
		if err != nil {
			g.appLogger.Warn("Invalid quantity in trade", "id", trade.Id, "amount", trade.Amount)
			continue
		}

		// Convert side string to connector.OrderSide
		side := connector.OrderSideBuy
		if trade.Side == "sell" {
			side = connector.OrderSideSell
		}

		// Parse timestamp - CreateTimeMs is in milliseconds as string
		createTimeMs, err := strconv.ParseInt(trade.CreateTimeMs, 10, 64)
		if err != nil {
			g.appLogger.Warn("Invalid timestamp in trade", "id", trade.Id, "createTimeMs", trade.CreateTimeMs)
			continue
		}
		timestamp := time.Unix(createTimeMs/1000, (createTimeMs%1000)*1000000)

		trades = append(trades, connector.Trade{
			ID:        trade.Id,
			Symbol:    symbol,
			Exchange:  "gate",
			Price:     price,
			Quantity:  quantity,
			Side:      side,
			Timestamp: timestamp,
		})
	}

	return trades, nil
}

// convertInterval converts Kronos interval format to Gate.io format
func convertInterval(interval string) string {
	// Gate.io uses: 10s, 1m, 5m, 15m, 30m, 1h, 4h, 8h, 1d, 7d, 30d
	// Kronos typically uses: 1m, 5m, 15m, 30m, 1h, 4h, 1d
	return interval
}

// GetPerpSymbol returns the perpetual symbol format (not used in spot)
func (g *gateSpot) GetPerpSymbol(asset portfolio.Asset) string {
	return asset.Symbol()
}

// GetSpotSymbol returns the spot symbol format
func (g *gateSpot) GetSpotSymbol(asset portfolio.Asset) string {
	return g.formatSymbol(asset.Symbol())
}
