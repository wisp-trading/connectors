package perp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (b *bybit) FetchCurrentFundingRates() (map[portfolio.Pair]perp.FundingRate, error) {
	client, err := b.client.GetClient()
	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetFundingRateHistory(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch funding rates: %w", err)
	}

	fundingRates := make(map[portfolio.Pair]perp.FundingRate)

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if fundingData, ok := item.(map[string]interface{}); ok {
						pair, rate, err := b.parseFundingRate(fundingData)
						if err != nil {
							fmt.Printf("failed to parse funding rate: %v\n", err)
							continue
						}
						fundingRates[pair] = rate
					}
				}
			}
		}
	}

	return fundingRates, nil
}

func (b *bybit) FetchFundingRate(pair portfolio.Pair) (*perp.FundingRate, error) {
	client, err := b.client.GetClient()
	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	symbol := b.GetPerpSymbol(pair)
	params := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
		"limit":    1,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetFundingRateHistory(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch funding rate for %s: %w", symbol, err)
	}

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				if len(listData) > 0 {
					if fundingData, ok := listData[0].(map[string]interface{}); ok {
						_, rate, err := b.parseFundingRate(fundingData)
						if err != nil {
							return nil, fmt.Errorf("failed to parse funding rate: %w", err)
						}
						return &rate, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("no funding rate data found for %s", symbol)
}

func (b *bybit) FetchHistoricalFundingRates(pair portfolio.Pair, startTime, endTime int64) ([]perp.HistoricalFundingRate, error) {
	client, err := b.client.GetClient()
	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	symbol := b.GetPerpSymbol(pair)
	params := map[string]interface{}{
		"category":  "linear",
		"symbol":    symbol,
		"startTime": startTime,
		"endTime":   endTime,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetFundingRateHistory(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical funding rates for %s: %w", symbol, err)
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
