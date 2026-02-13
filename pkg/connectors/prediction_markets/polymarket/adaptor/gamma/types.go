package gamma

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GammaClient wraps Polymarket Gamma Market Discovery API endpoints
type GammaClient interface {
	GetMarket(ctx context.Context, slug string) (*MarketResponse, error)
}

// MarketResponse represents a Polymarket prediction market
type MarketResponse struct {
	ConditionID string `json:"conditionId"` // The market/condition ID
	Slug        string `json:"slug"`

	// Market outcomes (e.g., ["YES", "NO"] or ["UP", "DOWN"])
	OutcomesRaw     string `json:"outcomes"`
	ClobTokenIdsRaw string `json:"clobTokenIds"` // Asset IDs for orderbook subscription

	// Market metadata
	Active          bool      `json:"active"`
	Closed          bool      `json:"closed"`
	AcceptingOrders bool      `json:"acceptingOrders"`
	StartTime       time.Time `json:"startTime"`
	ResolutionTime  time.Time `json:"resolutionTime"`
	UpdatedAt       time.Time `json:"updatedAt"`

	// Parsed conditions
	Conditions []Condition
}

// Condition represents one outcome in a market (e.g., YES or NO side)
type Condition struct {
	Name    string // "YES", "NO", "UP", "DOWN", etc.
	AssetID string // The CLOB token ID for orderbook subscription
	Index   int    // Position in outcomes array (0 or 1 typically)
}

func (m *MarketResponse) Parse() error {
	// Parse raw JSON arrays
	var outcomes []string
	var assetIds []string

	if err := json.Unmarshal([]byte(m.OutcomesRaw), &outcomes); err != nil {
		return fmt.Errorf("failed to parse outcomes: %w", err)
	}

	if err := json.Unmarshal([]byte(m.ClobTokenIdsRaw), &assetIds); err != nil {
		return fmt.Errorf("failed to parse asset IDs: %w", err)
	}

	// Validate they match
	if len(outcomes) != len(assetIds) {
		return fmt.Errorf("outcomes (%d) and assetIds (%d) length mismatch",
			len(outcomes), len(assetIds))
	}

	// Build conditions
	m.Conditions = make([]Condition, len(outcomes))
	for i := range outcomes {
		m.Conditions[i] = Condition{
			Name:    outcomes[i],
			AssetID: assetIds[i],
			Index:   i,
		}
	}

	return nil
}

// GetConditionByName finds a condition by outcome name (e.g., "YES", "UP")
func (m *MarketResponse) GetConditionByName(name string) *Condition {
	for i := range m.Conditions {
		if m.Conditions[i].Name == name {
			return &m.Conditions[i]
		}
	}
	return nil
}

// GetConditionByAssetID finds a condition by its asset/token ID
func (m *MarketResponse) GetConditionByAssetID(assetID string) *Condition {
	for i := range m.Conditions {
		if m.Conditions[i].AssetID == assetID {
			return &m.Conditions[i]
		}
	}
	return nil
}

// GetAllAssetIDs returns all asset IDs for websocket subscription
func (m *MarketResponse) GetAllAssetIDs() []string {
	ids := make([]string, len(m.Conditions))
	for i, condition := range m.Conditions {
		ids[i] = condition.AssetID
	}
	return ids
}

// IsActive returns true if market is open for trading
func (m *MarketResponse) IsActive() bool {
	return m.Active && m.AcceptingOrders && !m.Closed
}

// IsClosed returns true if market has closed (outcome decided)
func (m *MarketResponse) IsClosed() bool {
	return m.Closed || time.Now().After(m.ResolutionTime)
}

// TimeUntilClose returns duration until market closes
func (m *MarketResponse) TimeUntilClose() time.Duration {
	return time.Until(m.ResolutionTime)
}
