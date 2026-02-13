package polymarket

import (
	"context"
	"fmt"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

func (p *polymarket) GetMarket(slug string) (prediction.Market, error) {
	ctx := context.Background()
	marketData, err := p.gammaClient.GetMarket(ctx, slug)
	if err != nil {
		return prediction.Market{}, fmt.Errorf("failed to get market: %w", err)
	}

	// Parse the raw JSON fields
	if err := marketData.Parse(); err != nil {
		return prediction.Market{}, fmt.Errorf("failed to parse market data: %w", err)
	}

	// Determine outcome type (binary for YES/NO, categorical for multi-outcome)
	outcomeType := prediction.OutcomeTypeBinary
	if len(marketData.Conditions) > 2 {
		outcomeType = prediction.OutcomeTypeCategorical
	}

	// Build outcomes from conditions
	outcomes := make([]prediction.Outcome, len(marketData.Conditions))
	for i, condition := range marketData.Conditions {
		pair := prediction.NewPredictionPair(
			marketData.Slug,
			condition.Name, // "YES", "NO", "UP", "DOWN", etc.
			getQuoteAsset(),
		)

		outcomes[i] = prediction.Outcome{
			Pair:      pair,
			OutcomeId: condition.AssetID, // The CLOB token ID for orderbook
		}
	}

	// Handle resolution date (if closed)
	resolutionTime := &marketData.EndDate

	market := prediction.Market{
		MarketId:       marketData.ConditionID,
		Slug:           marketData.Slug,
		Exchange:       connector.ExchangeName("Polymarket"),
		OutcomeType:    outcomeType,
		Outcomes:       outcomes,
		Active:         marketData.Active,
		Closed:         marketData.Closed,
		ResolutionTime: resolutionTime,
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
