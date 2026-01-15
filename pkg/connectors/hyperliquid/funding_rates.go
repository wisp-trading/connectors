package hyperliquid

import (
	"fmt"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector/perp"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
)

func (h *hyperliquid) FetchCurrentFundingRates() (map[portfolio.Asset]perp.FundingRate, error) {
	contexts, err := h.marketData.GetAllAssetContexts()
	if err != nil {
		return nil, fmt.Errorf("failed to get asset contexts: %w", err)
	}

	fundingRates := make(map[portfolio.Asset]perp.FundingRate)

	for _, ctx := range contexts {
		asset := portfolio.NewAsset(ctx.Name)

		funding, err := numerical.NewFromString(ctx.Funding)
		if err != nil {
			h.appLogger.Warn("Invalid funding rate, skipping asset",
				"asset", ctx.Name,
				"funding", ctx.Funding,
				"error", err)
			continue
		}

		markPrice, err := numerical.NewFromString(ctx.MarkPrice)
		if err != nil {
			h.appLogger.Warn("Invalid mark price, skipping asset",
				"asset", ctx.Name,
				"markPrice", ctx.MarkPrice,
				"error", err)
			continue
		}

		oraclePrice, err := numerical.NewFromString(ctx.OraclePrice)
		if err != nil {
			h.appLogger.Warn("Invalid oracle price, skipping asset",
				"asset", ctx.Name,
				"oraclePrice", ctx.OraclePrice,
				"error", err)
			continue
		}

		fundingRates[asset] = perp.FundingRate{
			CurrentRate:     funding,
			Timestamp:       h.timeProvider.Now(),
			MarkPrice:       markPrice,
			IndexPrice:      oraclePrice,
			NextFundingTime: h.timeProvider.Now(),
		}
	}

	return fundingRates, nil
}

func (h *hyperliquid) FetchFundingRate(asset portfolio.Asset) (*perp.FundingRate, error) {
	ctx, err := h.marketData.GetAssetContext(asset.Symbol())
	if err != nil {
		return nil, fmt.Errorf("failed to get asset context: %w", err)
	}

	funding, err := numerical.NewFromString(ctx.Funding)
	if err != nil {
		return nil, fmt.Errorf("invalid funding rate for %s: %w", asset.Symbol(), err)
	}

	markPrice, err := numerical.NewFromString(ctx.MarkPrice)
	if err != nil {
		return nil, fmt.Errorf("invalid mark price for %s: %w", asset.Symbol(), err)
	}

	oraclePrice, err := numerical.NewFromString(ctx.OraclePrice)
	if err != nil {
		return nil, fmt.Errorf("invalid oracle price for %s: %w", asset.Symbol(), err)
	}

	return &perp.FundingRate{
		CurrentRate:     funding,
		Timestamp:       h.timeProvider.Now(),
		MarkPrice:       markPrice,
		IndexPrice:      oraclePrice,
		NextFundingTime: h.timeProvider.Now().Add(time.Hour),
	}, nil
}

func (h *hyperliquid) FetchHistoricalFundingRates(symbol portfolio.Asset, startTime, endTime int64) ([]perp.HistoricalFundingRate, error) {
	rawData, err := h.marketData.GetHistoricalFundingRates(symbol.Symbol(), startTime, endTime)
	if err != nil {
		return nil, err
	}

	var rates []perp.HistoricalFundingRate
	for _, entry := range rawData {
		fundingRate, err := numerical.NewFromString(entry.FundingRate)

		if err != nil {
			return nil, fmt.Errorf("invalid funding rate %s for symbol %s: %w", entry.FundingRate, symbol.Symbol(), err)
		}

		rates = append(rates, perp.HistoricalFundingRate{
			FundingRate: fundingRate,
			Timestamp:   time.Unix(entry.Time/1000, 0),
		})
	}

	return rates, nil
}

func (h *hyperliquid) SupportsFundingRates() bool {
	return true
}
