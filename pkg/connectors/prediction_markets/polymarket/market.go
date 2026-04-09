package polymarket

import (
	"context"
	"fmt"
	"time"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/gamma"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (p *polymarket) GetMarket(slug string) (prediction.Market, error) {
	ctx := context.Background()
	marketData, err := p.gammaClient.GetMarketBySlug(ctx, slug)
	if err != nil {
		return prediction.Market{}, fmt.Errorf("failed to get market: %w", err)
	}

	// Build outcomes from conditions
	outcomeIds := parseClobTokenIds(*marketData)
	if outcomeIds == nil {
		return prediction.Market{}, fmt.Errorf("failed to parse outcome IDs for market %s", slug)
	}

	outcomeLabels := parseOutcomes(*marketData)
	if outcomeLabels == nil {
		return prediction.Market{}, fmt.Errorf("failed to parse outcome labels for market %s", slug)
	}

	// Determine outcome type (binary for YES/NO, categorical for multi-outcome)
	outcomeType := prediction.OutcomeTypeBinary
	if len(outcomeIds) > 2 {
		outcomeType = prediction.OutcomeTypeCategorical
	}

	outcomes := make([]prediction.Outcome, len(outcomeIds))

	for i := range outcomeIds {
		pair := prediction.NewPredictionPair(
			marketData.Slug,
			outcomeLabels[i], // "YES", "NO", "UP", "DOWN", etc.
			getQuoteAsset(),
		)

		outcomes[i] = prediction.Outcome{
			Pair:      pair,
			OutcomeID: prediction.OutcomeIDFromString(outcomeIds[i]), // The CLOB token ID for orderbook
		}
	}

	// Handle resolution date (if closed)
	resolutionTime, err := time.Parse(time.RFC3339, marketData.EndDate)
	if err != nil {
		p.appLogger.Error("Failed to parse resolution time for market %s: %v", slug, err)
		return prediction.Market{}, fmt.Errorf("failed to parse resolution time: %w", err)
	}

	startTime, err := time.Parse(time.RFC3339, marketData.StartDate)
	if err != nil {
		p.appLogger.Error("Failed to parse start time for market %s: %v", slug, err)
		return prediction.Market{}, fmt.Errorf("failed to parse start time: %w", err)
	}

	market := prediction.Market{
		MarketID:       prediction.MarketIDFromString(marketData.ConditionID), // The Polymarket condition ID
		Slug:           marketData.Slug,
		Exchange:       p.GetConnectorInfo().Name,
		OutcomeType:    outcomeType,
		Outcomes:       outcomes,
		Active:         marketData.Active,
		Closed:         marketData.Closed,
		ResolutionTime: &resolutionTime,
		StartTime:      &startTime,
	}

	return market, nil
}

func (p *polymarket) GetRecurringMarket(baseSlug string, recurrence prediction.RecurrenceInterval) (prediction.Market, error) {
	duration, ok := recurrence.Duration()
	if !ok {
		return prediction.Market{}, fmt.Errorf("invalid recurrence interval")
	}

	now := time.Now().Unix()
	intervalSeconds := int64(duration.Seconds())

	// Round down to current interval boundary
	currentTimestamp := (now / intervalSeconds) * intervalSeconds

	// Build full slug
	slug := fmt.Sprintf("%s-%d", baseSlug, currentTimestamp)

	// Fetch market
	market, err := p.GetMarket(slug)
	if err != nil {
		return prediction.Market{}, fmt.Errorf("failed to get market: %w", err)
	}

	market.RecurringMarket = &prediction.RecurringMarket{
		RecurrenceInterval: recurrence,
	}

	return market, nil
}

func (p *polymarket) UnsubscribeMarket(market prediction.Market) error {
	// Websocket unsubscribe not yet implemented in SDK
	// TODO: Implement when polymarket SDK adds UnsubscribeMarketAssets support
	p.appLogger.Info("Unsubscribe for market %s not yet implemented", market.Slug)
	return nil
}

func (p *polymarket) Markets(filter *prediction.MarketsFilter) ([]prediction.Market, error) {
	ctx := context.Background()

	req := &gamma.MarketsRequest{
		Active: boolPtr(true),
	}

	// Apply filters if provided
	if filter != nil {
		if filter.MinVolume != "" {
			req.VolumeMin = stringPtr(filter.MinVolume)
		}
		if filter.MinLiquidity != "" {
			req.LiquidityMin = stringPtr(filter.MinLiquidity)
		}
		if filter.Active != nil {
			req.Active = filter.Active
		}
	}

	gammaMarkets, err := p.gammaClient.MarketsAll(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch markets: %w", err)
	}

	markets := make([]prediction.Market, 0, len(gammaMarkets))
	for _, gammaMarket := range gammaMarkets {
		market, err := p.buildMarketFromGamma(&gammaMarket)
		if err != nil {
			p.appLogger.Warn("Failed to parse market %s: %v", gammaMarket.Slug, err)
			continue
		}
		markets = append(markets, market)
	}

	return markets, nil
}

// buildMarketFromGamma converts a gamma.Market to prediction.Market
// Reuses the same logic as GetMarket to ensure consistency
func (p *polymarket) buildMarketFromGamma(gammaMarket *gamma.Market) (prediction.Market, error) {
	// Build outcomes from conditions
	outcomeIds := parseClobTokenIds(*gammaMarket)
	if outcomeIds == nil {
		return prediction.Market{}, fmt.Errorf("failed to parse outcome IDs for market %s", gammaMarket.Slug)
	}

	outcomeLabels := parseOutcomes(*gammaMarket)
	if outcomeLabels == nil {
		return prediction.Market{}, fmt.Errorf("failed to parse outcome labels for market %s", gammaMarket.Slug)
	}

	// Determine outcome type (binary for YES/NO, categorical for multi-outcome)
	outcomeType := prediction.OutcomeTypeBinary
	if len(outcomeIds) > 2 {
		outcomeType = prediction.OutcomeTypeCategorical
	}

	outcomes := make([]prediction.Outcome, len(outcomeIds))
	for i := range outcomeIds {
		pair := prediction.NewPredictionPair(
			gammaMarket.Slug,
			outcomeLabels[i],
			getQuoteAsset(),
		)

		outcomes[i] = prediction.Outcome{
			Pair:      pair,
			OutcomeID: prediction.OutcomeIDFromString(outcomeIds[i]),
		}
	}

	// Handle resolution date
	resolutionTime, err := time.Parse(time.RFC3339, gammaMarket.EndDate)
	if err != nil {
		p.appLogger.Error("Failed to parse resolution time for market %s: %v", gammaMarket.Slug, err)
		return prediction.Market{}, fmt.Errorf("failed to parse resolution time: %w", err)
	}

	startTime, err := time.Parse(time.RFC3339, gammaMarket.StartDate)
	if err != nil {
		p.appLogger.Error("Failed to parse start time for market %s: %v", gammaMarket.Slug, err)
		return prediction.Market{}, fmt.Errorf("failed to parse start time: %w", err)
	}

	market := prediction.Market{
		MarketID:       prediction.MarketIDFromString(gammaMarket.ConditionID),
		Slug:           gammaMarket.Slug,
		Exchange:       p.GetConnectorInfo().Name,
		OutcomeType:    outcomeType,
		Outcomes:       outcomes,
		Active:         gammaMarket.Active,
		Closed:         gammaMarket.Closed,
		ResolutionTime: &resolutionTime,
		StartTime:      &startTime,
	}

	return market, nil
}

// boolPtr is a helper to convert bool to *bool
func boolPtr(b bool) *bool {
	return &b
}

// stringPtr is a helper to convert string to *string
func stringPtr(s string) *string {
	return &s
}
