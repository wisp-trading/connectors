package polymarket

import (
	"context"
	"fmt"
	"strconv"
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

	return p.buildMarketFromGamma(marketData, "", 0)
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
		return prediction.Market{}, fmt.Errorf("failed to get recurring market: %w", err)
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

	// Fetch via EventsAll: date range and status are safe to push to the API at
	// event level (an event's dates encompass its markets). Volume/liquidity are
	// applied client-side per-market from the embedded markets array so filtering
	// is precise. The event slug is available on each gamma.Event, which is the
	// only way to get it without a separate join call.
	req := &gamma.EventsRequest{
		Active: boolPtr(true),
	}

	if filter != nil {
		if filter.Limit != nil {
			req.Limit = filter.Limit
		}
		if filter.Offset != nil {
			req.Offset = filter.Offset
		}
		if filter.MinVolume != "" {
			req.VolumeMin = stringPtr(filter.MinVolume)
		}
		if filter.MaxVolume != "" {
			req.VolumeMax = stringPtr(filter.MaxVolume)
		}
		if filter.MinLiquidity != "" {
			req.LiquidityMin = stringPtr(filter.MinLiquidity)
		}
		if filter.MaxLiquidity != "" {
			req.LiquidityMax = stringPtr(filter.MaxLiquidity)
		}
		if filter.MinEndDate != "" {
			req.EndDateMin = filter.MinEndDate
		}
		if filter.MaxEndDate != "" {
			req.EndDateMax = filter.MaxEndDate
		}
		if filter.Active != nil {
			req.Active = filter.Active
		}
		if filter.Closed != nil {
			req.Closed = filter.Closed
		}
	}

	events, err := p.gammaClient.EventsAll(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	markets := make([]prediction.Market, 0)
	for _, event := range events {
		for _, gammaMarket := range event.Markets {
			if filter != nil && !marketPassesFilter(&gammaMarket, filter) {
				continue
			}
			market, err := p.buildMarketFromGamma(&gammaMarket, event.Slug, event.NegRiskFeeBips)
			if err != nil {
				p.appLogger.Warn("Failed to parse market %s: %v", gammaMarket.Slug, err)
				continue
			}
			markets = append(markets, market)
		}
	}

	return markets, nil
}

// marketPassesFilter applies per-market volume and liquidity filters client-side.
// Markets embedded in EventsAll responses carry these values as decimal strings.
func marketPassesFilter(m *gamma.Market, f *prediction.MarketsFilter) bool {
	if f.MinLiquidity != "" {
		min, err := strconv.ParseFloat(f.MinLiquidity, 64)
		if err == nil {
			liq, err := strconv.ParseFloat(m.Liquidity, 64)
			if err != nil || liq < min {
				return false
			}
		}
	}
	if f.MaxLiquidity != "" {
		max, err := strconv.ParseFloat(f.MaxLiquidity, 64)
		if err == nil {
			liq, err := strconv.ParseFloat(m.Liquidity, 64)
			if err != nil || liq > max {
				return false
			}
		}
	}
	if f.MinVolume != "" {
		min, err := strconv.ParseFloat(f.MinVolume, 64)
		if err == nil {
			vol, err := strconv.ParseFloat(m.Volume, 64)
			if err != nil || vol < min {
				return false
			}
		}
	}
	if f.MaxVolume != "" {
		max, err := strconv.ParseFloat(f.MaxVolume, 64)
		if err == nil {
			vol, err := strconv.ParseFloat(m.Volume, 64)
			if err != nil || vol > max {
				return false
			}
		}
	}
	return true
}

// buildMarketFromGamma converts a gamma.Market to prediction.Market.
// eventSlug is the parent event's slug (used for URL construction); pass empty
// string when the event context is unavailable (e.g. single-market lookups).
// negRiskFeeBips is the NegRisk merge/split fee from the parent event (0 when unknown).
func (p *polymarket) buildMarketFromGamma(gammaMarket *gamma.Market, eventSlug string, negRiskFeeBips int) (prediction.Market, error) {
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

	// Handle resolution date — field may be empty for some markets
	var resolutionTime *time.Time
	if gammaMarket.EndDate != "" {
		t, err := time.Parse(time.RFC3339, gammaMarket.EndDate)
		if err != nil {
			p.appLogger.Warn("Failed to parse resolution time for market %s: %v", gammaMarket.Slug, err)
		} else {
			resolutionTime = &t
		}
	}

	var startTime *time.Time
	if gammaMarket.StartDate != "" {
		t, err := time.Parse(time.RFC3339, gammaMarket.StartDate)
		if err != nil {
			p.appLogger.Warn("Failed to parse start time for market %s: %v", gammaMarket.Slug, err)
		} else {
			startTime = &t
		}
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

	// Fall back to the market's own slug only when no event slug was provided
	// (e.g. single-market lookups via GetMarket/GetRecurringMarket).
	if eventSlug == "" {
		eventSlug = gammaMarket.Slug
	}

	market := prediction.Market{
		MarketID:       prediction.MarketIDFromString(gammaMarket.ConditionID),
		Slug:           gammaMarket.Slug,
		Exchange:       p.GetConnectorInfo().Name,
		OutcomeType:    outcomeType,
		Outcomes:       outcomes,
		Active:         gammaMarket.Active,
		Closed:         gammaMarket.Closed,
		ResolutionTime: resolutionTime,
		StartTime:      startTime,
		Tags:           tags,
		EventSlug:      eventSlug,
		NegRisk:        gammaMarket.NegRisk,
		NegRiskFeeBips: negRiskFeeBips,
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
