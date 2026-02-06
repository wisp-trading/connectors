package perp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (b *bybit) FetchKlines(pair portfolio.Pair, interval string, limit int) ([]connector.Kline, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   b.GetPerpSymbol(pair),
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
						kline := b.parseKline(klineData)
						klines = append(klines, kline)
					}
				}
			}
		}
	}

	return klines, nil
}

func (b *bybit) parseKline(data []interface{}) connector.Kline {
	kline := connector.Kline{}

	if len(data) >= 7 {
		if openTimeStr, ok := data[0].(string); ok {
			if timestamp, err := strconv.ParseInt(openTimeStr, 10, 64); err == nil {
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

func (b *bybit) FetchPrice(pair portfolio.Pair) (*connector.Price, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   b.GetPerpSymbol(pair),
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
								Symbol:    pair.Symbol(),
								Price:     price,
								Source:    b.GetConnectorInfo().Name,
								Timestamp: b.timeProvider.Now(),
							}, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("price not found")
}

func (b *bybit) FetchOrderBook(pair portfolio.Pair, depth int) (*connector.OrderBook, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   b.GetPerpSymbol(pair),
		"limit":    depth,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetOrderBookInfo(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch orderbook: %w", err)
	}

	orderBook := &connector.OrderBook{
		Pair:      pair,
		Timestamp: b.timeProvider.Now(),
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

// FetchContracts implements perp.Connector interface
func (b *bybit) FetchContracts() ([]connector.ContractInfo, error) {
	// Bybit doesn't have a direct contract info endpoint
	// Return empty slice for now
	return []connector.ContractInfo{}, nil
}
