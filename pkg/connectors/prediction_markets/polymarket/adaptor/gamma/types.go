package gamma

import (
	"context"
	"encoding/json"
	"time"
)

// GammaClient wraps Polymarket Gamma Market Discovery API endpoints
type GammaClient interface {
	GetMarket(ctx context.Context, slug string) (*MarketResponse, error)
}

// MarketResponse is returned directly as an array from /markets endpoint
type MarketResponse struct {
	ConditionID string `json:"conditionId"`
	Slug        string `json:"slug"`

	// Raw JSON strings
	OutcomesRaw      string `json:"outcomes"`
	OutcomePricesRaw string `json:"outcomePrices"`
	ClobTokenIdsRaw  string `json:"clobTokenIds"`

	// Parsed results
	Outcomes      []string
	OutcomePrices []string
	ClobTokenIds  []string

	Active          bool      `json:"active"`
	Closed          bool      `json:"closed"`
	AcceptingOrders bool      `json:"acceptingOrders"`
	BestBid         float64   `json:"bestBid"`
	BestAsk         float64   `json:"bestAsk"`
	Spread          float64   `json:"spread"`
	LastTradePrice  float64   `json:"lastTradePrice"`
	EndDate         time.Time `json:"endDate"`
	StartDate       time.Time `json:"startDate"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (m *MarketResponse) Parse() error {
	json.Unmarshal([]byte(m.OutcomesRaw), &m.Outcomes)
	json.Unmarshal([]byte(m.OutcomePricesRaw), &m.OutcomePrices)
	json.Unmarshal([]byte(m.ClobTokenIdsRaw), &m.ClobTokenIds)
	return nil
}
