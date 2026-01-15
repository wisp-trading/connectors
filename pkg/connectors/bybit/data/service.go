package data

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector/perp"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	bybit "github.com/bybit-exchange/bybit.go.api"
)

type Config struct {
	APIKey          string
	APISecret       string
	BaseURL         string
	IsTestnet       bool
	DefaultSlippage float64
}

type MarketDataService interface {
	Initialize(config *Config) error
	FetchKlines(symbol, interval string, limit int) ([]connector.Kline, error)
	FetchPrice(symbol string) (*connector.Price, error)
	FetchOrderBook(symbol string, depth int) (*connector.OrderBook, error)
	FetchRecentTrades(symbol string, limit int) ([]connector.Trade, error)
	FetchFundingRate(symbol string) (*perp.FundingRate, error)
	FetchCurrentFundingRates() (map[portfolio.Asset]perp.FundingRate, error)
	FetchHistoricalFundingRates(symbol string, startTime, endTime int64) ([]perp.HistoricalFundingRate, error)
	FetchAvailablePerpetualAssets() ([]portfolio.Asset, error)
	FetchAvailableSpotAssets() ([]portfolio.Asset, error)
}

type marketDataService struct {
	client       *bybit.Client
	config       *Config
	timeProvider temporal.TimeProvider
	mu           sync.RWMutex
}

func NewMarketDataService(timeProvider temporal.TimeProvider) MarketDataService {
	return &marketDataService{
		timeProvider: timeProvider,
	}
}

func (m *marketDataService) Initialize(config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		return fmt.Errorf("market data service already initialized")
	}

	m.config = config
	m.client = bybit.NewBybitHttpClient(config.APIKey, config.APISecret, bybit.WithBaseURL(config.BaseURL))
	return nil
}

func (m *marketDataService) FetchKlines(symbol, interval string, limit int) ([]connector.Kline, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("market data service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
		"interval": interval,
		"limit":    limit,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetMarketKline(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch klines: %w", err)
	}

	var klines []connector.Kline
	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if klineData, ok := item.([]interface{}); ok && len(klineData) >= 7 {
						kline := m.parseKline(klineData)
						klines = append(klines, kline)
					}
				}
			}
		}
	}

	return klines, nil
}

func (m *marketDataService) parseKline(data []interface{}) connector.Kline {
	kline := connector.Kline{}

	if len(data) >= 7 {
		if openTimeStr, ok := data[0].(string); ok {
			if timestamp, err := strconv.ParseInt(openTimeStr, 10, 64); err == nil {
				// Bybit returns timestamp in milliseconds
				kline.OpenTime = time.Unix(timestamp/1000, (timestamp%1000)*1000000)
			}
		}
		if openStr, ok := data[1].(string); ok {
			if val, err := strconv.ParseFloat(openStr, 64); err == nil {
				kline.Open = val
			}
		}
		if highStr, ok := data[2].(string); ok {
			if val, err := strconv.ParseFloat(highStr, 64); err == nil {
				kline.High = val
			}
		}
		if lowStr, ok := data[3].(string); ok {
			if val, err := strconv.ParseFloat(lowStr, 64); err == nil {
				kline.Low = val
			}
		}
		if closeStr, ok := data[4].(string); ok {
			if val, err := strconv.ParseFloat(closeStr, 64); err == nil {
				kline.Close = val
			}
		}
		if volumeStr, ok := data[5].(string); ok {
			if val, err := strconv.ParseFloat(volumeStr, 64); err == nil {
				kline.Volume = val
			}
		}
	}

	return kline
}

func (m *marketDataService) FetchPrice(symbol string) (*connector.Price, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("market data service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetMarketTickers(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price: %w", err)
	}

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok && len(listData) > 0 {
				if tickerData, ok := listData[0].(map[string]interface{}); ok {
					if lastPrice, ok := tickerData["lastPrice"].(string); ok {
						if price, err := numerical.NewFromString(lastPrice); err == nil {
							return &connector.Price{
								Symbol:    symbol,
								Price:     price,
								Source:    "Bybit",
								Timestamp: m.timeProvider.Now(),
							}, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("price not found")
}

func (m *marketDataService) FetchOrderBook(symbol string, depth int) (*connector.OrderBook, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("market data service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
		"limit":    depth,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetOrderBookInfo(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch orderbook: %w", err)
	}

	orderBook := &connector.OrderBook{
		Asset:     portfolio.NewAsset(symbol),
		Timestamp: m.timeProvider.Now(),
		Bids:      make([]connector.PriceLevel, 0),
		Asks:      make([]connector.PriceLevel, 0),
	}

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if bids, ok := resultData["b"].([]interface{}); ok {
				for _, item := range bids {
					if bidData, ok := item.([]interface{}); ok && len(bidData) >= 2 {
						if priceStr, ok := bidData[0].(string); ok {
							if qtyStr, ok := bidData[1].(string); ok {
								price, _ := numerical.NewFromString(priceStr)
								qty, _ := numerical.NewFromString(qtyStr)
								orderBook.Bids = append(orderBook.Bids, connector.PriceLevel{
									Price:    price,
									Quantity: qty,
								})
							}
						}
					}
				}
			}
			if asks, ok := resultData["a"].([]interface{}); ok {
				for _, item := range asks {
					if askData, ok := item.([]interface{}); ok && len(askData) >= 2 {
						if priceStr, ok := askData[0].(string); ok {
							if qtyStr, ok := askData[1].(string); ok {
								price, _ := numerical.NewFromString(priceStr)
								qty, _ := numerical.NewFromString(qtyStr)
								orderBook.Asks = append(orderBook.Asks, connector.PriceLevel{
									Price:    price,
									Quantity: qty,
								})
							}
						}
					}
				}
			}
		}
	}

	return orderBook, nil
}

func (m *marketDataService) FetchRecentTrades(symbol string, limit int) ([]connector.Trade, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("market data service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
		"limit":    limit,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetPublicRecentTrades(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent trades: %w", err)
	}

	var trades []connector.Trade
	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if tradeData, ok := item.(map[string]interface{}); ok {
						trade := m.parseTrade(tradeData, symbol)
						trades = append(trades, trade)
					}
				}
			}
		}
	}

	return trades, nil
}

func (m *marketDataService) parseTrade(data map[string]interface{}, symbol string) connector.Trade {
	trade := connector.Trade{
		Symbol:    symbol,
		Timestamp: m.timeProvider.Now(),
	}

	if side, ok := data["side"].(string); ok {
		trade.Side = connector.OrderSide(side)
	}
	if price, ok := data["price"].(string); ok {
		if val, err := numerical.NewFromString(price); err == nil {
			trade.Price = val
		}
	}
	if quantity, ok := data["size"].(string); ok {
		if val, err := numerical.NewFromString(quantity); err == nil {
			trade.Quantity = val
		}
	}

	return trade
}

func (m *marketDataService) FetchFundingRate(symbol string) (*perp.FundingRate, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("market data service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetFundingRateHistory(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch funding rate: %w", err)
	}

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok && len(listData) > 0 {
				if fundingData, ok := listData[0].(map[string]interface{}); ok {
					if fundingRate, ok := fundingData["fundingRate"].(string); ok {
						if rate, err := numerical.NewFromString(fundingRate); err == nil {
							return &perp.FundingRate{
								CurrentRate:     rate,
								Timestamp:       m.timeProvider.Now(),
								NextFundingTime: m.timeProvider.Now(),
							}, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("funding rate not found")
}

func (m *marketDataService) FetchAvailablePerpetualAssets() ([]portfolio.Asset, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("market data service not initialized")
	}

	params := map[string]interface{}{
		"category": "linear",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetInstrumentInfo(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instruments: %w", err)
	}

	var assets []portfolio.Asset

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if instrumentData, ok := item.(map[string]interface{}); ok {
						if symbol, ok := instrumentData["symbol"].(string); ok {
							// Extract base symbol from perpetual symbol (e.g., "BTCUSDT" -> "BTC")
							baseSymbol := symbol
							if len(symbol) > 4 && symbol[len(symbol)-4:] == "USDT" {
								baseSymbol = symbol[:len(symbol)-4]
							}
							asset := portfolio.NewAsset(baseSymbol)
							assets = append(assets, asset)
						}
					}
				}
			}
		}
	}

	return assets, nil
}

func (m *marketDataService) FetchAvailableSpotAssets() ([]portfolio.Asset, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("market data service not initialized")
	}

	params := map[string]interface{}{
		"category": "spot",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetInstrumentInfo(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot instruments: %w", err)
	}

	var assets []portfolio.Asset

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if instrumentData, ok := item.(map[string]interface{}); ok {
						if symbol, ok := instrumentData["symbol"].(string); ok {
							// Extract base symbol from spot pair (e.g., "BTCUSDT" -> "BTC")
							baseSymbol := symbol
							if len(symbol) > 4 && symbol[len(symbol)-4:] == "USDT" {
								baseSymbol = symbol[:len(symbol)-4]
							}
							asset := portfolio.NewAsset(baseSymbol)
							assets = append(assets, asset)
						}
					}
				}
			}
		}
	}

	return assets, nil
}

func (m *marketDataService) FetchCurrentFundingRates() (map[portfolio.Asset]perp.FundingRate, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("market data service not initialized")
	}

	// Fetch all perpetual assets first
	assets, err := m.FetchAvailablePerpetualAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perpetual assets: %w", err)
	}

	fundingRates := make(map[portfolio.Asset]perp.FundingRate)

	// Fetch funding rate for each asset
	for _, asset := range assets {
		symbol := asset.Symbol() + "USDT"
		rate, err := m.FetchFundingRate(symbol)
		if err != nil {
			// Log warning but continue with other assets
			continue
		}
		if rate != nil {
			fundingRates[asset] = *rate
		}
	}

	return fundingRates, nil
}

func (m *marketDataService) FetchHistoricalFundingRates(symbol string, startTime, endTime int64) ([]perp.HistoricalFundingRate, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("market data service not initialized")
	}

	params := map[string]interface{}{
		"category":  "linear",
		"symbol":    symbol,
		"startTime": startTime,
		"endTime":   endTime,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetFundingRateHistory(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical funding rates: %w", err)
	}

	var rates []perp.HistoricalFundingRate

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if fundingData, ok := item.(map[string]interface{}); ok {
						var rate perp.HistoricalFundingRate

						if fundingRateStr, ok := fundingData["fundingRate"].(string); ok {
							if fundingRate, err := numerical.NewFromString(fundingRateStr); err == nil {
								rate.FundingRate = fundingRate
							}
						}

						if fundingTimeStr, ok := fundingData["fundingRateTimestamp"].(string); ok {
							if timestamp, err := strconv.ParseInt(fundingTimeStr, 10, 64); err == nil {
								// Bybit returns timestamp in milliseconds
								rate.Timestamp = time.Unix(timestamp/1000, (timestamp%1000)*1000000)
							}
						}

						rates = append(rates, rate)
					}
				}
			}
		}
	}

	return rates, nil
}
