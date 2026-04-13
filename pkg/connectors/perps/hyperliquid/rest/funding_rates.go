package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sonirico/go-hyperliquid"
)

// AssetContext represents the parsed asset context data
type AssetContext struct {
	Name         string
	Funding      string
	MarkPrice    string
	OraclePrice  string
	Premium      string
	OpenInterest string
}

// universeItem represents a single asset in the universe array
type universeItem struct {
	Name         string `json:"name"`
	SzDecimals   int    `json:"szDecimals"`
	MaxLeverage  int    `json:"maxLeverage"`
	OnlyIsolated bool   `json:"onlyIsolated,omitempty"`
}

// assetCtxItem represents a single asset context with funding data
type assetCtxItem struct {
	Funding      string   `json:"funding"`
	MarkPx       string   `json:"markPx"`
	OraclePx     string   `json:"oraclePx"`
	Premium      string   `json:"premium"`
	OpenInterest string   `json:"openInterest"`
	MidPx        string   `json:"midPx,omitempty"`
	DayNtlVlm    string   `json:"dayNtlVlm,omitempty"`
	PrevDayPx    string   `json:"prevDayPx,omitempty"`
	ImpactPxs    []string `json:"impactPxs,omitempty"`
}

// metaObject represents the first element of the API response
type metaObject struct {
	Universe []universeItem `json:"universe"`
}

func (m *marketDataService) fetchMetaAndAssetCtxs() ([]universeItem, []assetCtxItem, error) {
	reqBody := map[string]string{"type": "metaAndAssetCtxs"}
	jsonData, _ := json.Marshal(reqBody)

	// Make direct HTTP call
	resp, err := http.Post("https://api.hyperliquid.xyz/info", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	// API returns: [{"universe": [...], ...}, [{...}, {...}]]
	// Parse as raw JSON array first
	var rawResponse []json.RawMessage
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(rawResponse) < 2 {
		return nil, nil, fmt.Errorf("invalid response: expected 2 elements, got %d", len(rawResponse))
	}

	// First element is an object with "universe" key
	var metaObj metaObject
	if err := json.Unmarshal(rawResponse[0], &metaObj); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal meta object: %w", err)
	}

	// Second element is array of asset contexts
	var assetCtxs []assetCtxItem
	if err := json.Unmarshal(rawResponse[1], &assetCtxs); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal asset contexts: %w", err)
	}

	return metaObj.Universe, assetCtxs, nil
}

// GetAssetContext returns the asset context for a specific coin
func (m *marketDataService) GetAssetContext(coin string) (*AssetContext, error) {
	universe, assetCtxs, err := m.fetchMetaAndAssetCtxs()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta and asset contexts: %w", err)
	}

	for i := 0; i < len(universe) && i < len(assetCtxs); i++ {
		if universe[i].Name != coin {
			continue
		}

		ctx := assetCtxs[i]
		return &AssetContext{
			Name:         universe[i].Name,
			Funding:      ctx.Funding,
			MarkPrice:    ctx.MarkPx,
			OraclePrice:  ctx.OraclePx,
			Premium:      ctx.Premium,
			OpenInterest: ctx.OpenInterest,
		}, nil
	}

	return nil, fmt.Errorf("asset %s not found", coin)
}

// GetAllAssetContexts returns asset contexts for all assets
func (m *marketDataService) GetAllAssetContexts() ([]AssetContext, error) {
	universe, assetCtxs, err := m.fetchMetaAndAssetCtxs()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta and asset contexts: %w", err)
	}

	var contexts []AssetContext

	for i := 0; i < len(universe) && i < len(assetCtxs); i++ {
		ctx := assetCtxs[i]
		contexts = append(contexts, AssetContext{
			Name:         universe[i].Name,
			Funding:      ctx.Funding,
			MarkPrice:    ctx.MarkPx,
			OraclePrice:  ctx.OraclePx,
			Premium:      ctx.Premium,
			OpenInterest: ctx.OpenInterest,
		})
	}

	return contexts, nil
}

// GetHistoricalFundingRates returns historical funding rates for a specific coin
// This is a public endpoint and doesn't require authentication
func (m *marketDataService) GetHistoricalFundingRates(coin string, startTime, endTime int64) ([]hyperliquid.FundingHistory, error) {
	info, err := m.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("info client not configured: %w", err)
	}

	// Convert both timestamps from seconds to milliseconds
	startTimeMs := startTime * millisecondsPerSecond
	endTimeMs := endTime * millisecondsPerSecond

	return info.FundingHistory(coin, startTimeMs, &endTimeMs)
}
