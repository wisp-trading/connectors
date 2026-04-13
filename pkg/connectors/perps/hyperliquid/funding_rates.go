package hyperliquid

import (
	"fmt"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (h *hyperliquid) FetchCurrentFundingRates() (map[portfolio.Pair]perp.FundingRate, error) {
	contexts, err := h.marketData.GetAllAssetContexts()
	if err != nil {
		return nil, fmt.Errorf("failed to get asset contexts: %w", err)
	}

	fundingRates := make(map[portfolio.Pair]perp.FundingRate)

	for _, ctx := range contexts {
		pair := h.coinToPair(ctx.Name)

		funding, err := numerical.NewFromString(ctx.Funding)
		if err != nil {
			h.appLogger.Warn("Invalid funding rate, skipping pair",
				"pair", ctx.Name,
				"funding", ctx.Funding,
				"error", err)
			continue
		}

		markPrice, err := numerical.NewFromString(ctx.MarkPrice)
		if err != nil {
			h.appLogger.Warn("Invalid mark price, skipping pair",
				"pair", ctx.Name,
				"markPrice", ctx.MarkPrice,
				"error", err)
			continue
		}

		oraclePrice, err := numerical.NewFromString(ctx.OraclePrice)
		if err != nil {
			h.appLogger.Warn("Invalid oracle price, skipping pair",
				"pair", ctx.Name,
				"oraclePrice", ctx.OraclePrice,
				"error", err)
			continue
		}

		fundingRates[pair] = perp.FundingRate{
			Pair:            pair,
			CurrentRate:     funding,
			Timestamp:       h.timeProvider.Now(),
			MarkPrice:       markPrice,
			IndexPrice:      oraclePrice,
			NextFundingTime: h.timeProvider.Now(),
		}
	}

	return fundingRates, nil
}

func (h *hyperliquid) FetchFundingRate(pair portfolio.Pair) (*perp.FundingRate, error) {
	symbol := pair.Base().Symbol()
	ctx, err := h.marketData.GetAssetContext(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get asset context: %w", err)
	}

	funding, err := numerical.NewFromString(ctx.Funding)
	if err != nil {
		return nil, fmt.Errorf("invalid funding rate for %s: %w", symbol, err)
	}

	markPrice, err := numerical.NewFromString(ctx.MarkPrice)
	if err != nil {
		return nil, fmt.Errorf("invalid mark price for %s: %w", symbol, err)
	}

	oraclePrice, err := numerical.NewFromString(ctx.OraclePrice)
	if err != nil {
		return nil, fmt.Errorf("invalid oracle price for %s: %w", symbol, err)
	}

	return &perp.FundingRate{
		CurrentRate:     funding,
		Timestamp:       h.timeProvider.Now(),
		MarkPrice:       markPrice,
		IndexPrice:      oraclePrice,
		NextFundingTime: h.timeProvider.Now().Add(time.Hour),
	}, nil
}

func (h *hyperliquid) FetchHistoricalFundingRates(pair portfolio.Pair, startTime, endTime int64) ([]perp.HistoricalFundingRate, error) {
	rawData, err := h.marketData.GetHistoricalFundingRates(pair.Base().Symbol(), startTime, endTime)
	if err != nil {
		return nil, err
	}

	var rates []perp.HistoricalFundingRate
	for _, entry := range rawData {
		fundingRate, err := numerical.NewFromString(entry.FundingRate)

		if err != nil {
			return nil, fmt.Errorf("invalid funding rate %s for symbol %s: %w", entry.FundingRate, pair.Base().Symbol(), err)
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
