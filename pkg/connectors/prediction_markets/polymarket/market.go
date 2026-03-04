package polymarket

import (
	"context"
	"fmt"
	"time"

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

	market := prediction.Market{
		MarketID:       prediction.MarketIDFromString(marketData.ConditionID), // The Polymarket condition ID
		Slug:           marketData.Slug,
		Exchange:       p.GetConnectorInfo().Name,
		OutcomeType:    outcomeType,
		Outcomes:       outcomes,
		Active:         marketData.Active,
		Closed:         marketData.Closed,
		ResolutionTime: &resolutionTime,
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
	err := p.clobWebsocket.UnsubscribeMarket(market)
	if err != nil {
		return err
	}
	p.appLogger.Info("Unsubscribed from market %s", market.Slug)

	return nil
}
