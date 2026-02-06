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

func (b *bybit) FetchFundingRate(symbol string) (*perp.FundingRate, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
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
								Timestamp:       b.timeProvider.Now(),
								NextFundingTime: b.timeProvider.Now(),
							}, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("funding rate not found")
}

func (b *bybit) FetchCurrentFundingRates() (map[portfolio.Asset]perp.FundingRate, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	assets, err := b.FetchAvailablePerpetualAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perpetual assets: %w", err)
	}

	fundingRates := make(map[portfolio.Asset]perp.FundingRate)

	for _, asset := range assets {
		symbol := asset.Symbol() + "USDT"
		rate, err := b.FetchFundingRate(symbol)
		if err != nil {
			continue
		}
		if rate != nil {
			fundingRates[asset] = *rate
		}
	}

	return fundingRates, nil
}

func (b *bybit) FetchHistoricalFundingRates(symbol string, startTime, endTime int64) ([]perb.HistoricalFundingRate, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
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

	var rates []perb.HistoricalFundingRate

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if fundingData, ok := item.(map[string]interface{}); ok {
						var rate perb.HistoricalFundingRate

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
