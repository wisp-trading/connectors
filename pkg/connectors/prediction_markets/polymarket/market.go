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

	return p.buildMarketFromGamma(marketData)
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
		Active:     boolPtr(true),
		IncludeTag: boolPtr(true),
	}

	// Apply filters if provided
	if filter != nil {
		// Pagination
		if filter.Limit != nil {
			req.Limit = filter.Limit
		}
		if filter.Offset != nil {
			req.Offset = filter.Offset
		}

		// Volume filters
		if filter.MinVolume != "" {
			req.VolumeMin = stringPtr(filter.MinVolume)
		}
		if filter.MaxVolume != "" {
			req.VolumeMax = stringPtr(filter.MaxVolume)
		}

		// Liquidity filters
		if filter.MinLiquidity != "" {
			req.LiquidityMin = stringPtr(filter.MinLiquidity)
		}
		if filter.MaxLiquidity != "" {
			req.LiquidityMax = stringPtr(filter.MaxLiquidity)
		}

		// Date range filters
		if filter.MinEndDate != "" {
			req.EndDateMin = filter.MinEndDate
		}
		if filter.MaxEndDate != "" {
			req.EndDateMax = filter.MaxEndDate
		}

		// Status filters
		if filter.Active != nil {
			req.Active = filter.Active
		}
		if filter.Closed != nil {
			req.Closed = filter.Closed
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

	// Convert gamma tags to prediction tags
	tags := make([]prediction.Tag, len(gammaMarket.Tags))
	for i, tag := range gammaMarket.Tags {
		tags[i] = prediction.Tag{
			ID:    tag.ID,
			Label: tag.Label,
			Slug:  tag.Slug,
		}
	}

	// Extract the parent event slug so we can build a valid Polymarket URL.
	// Polymarket URLs use the event slug: https://polymarket.com/event/{event_slug}
	eventSlug := ""
	if len(gammaMarket.Events) > 0 {
		eventSlug = gammaMarket.Events[0].Slug
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
		Tags:           tags,
		EventSlug:      eventSlug,
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
