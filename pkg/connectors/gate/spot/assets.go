package spot

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
)

// FetchAvailableSpotAssets fetches all available spot assets from Gate.io
func (g *gateSpot) FetchAvailableSpotAssets() ([]portfolio.Asset, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()

	// Get all currency pairs from Gate.io
	pairs, _, err := client.SpotApi.ListCurrencyPairs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch currency pairs: %w", err)
	}

	assetMap := make(map[string]bool)
	var assets []portfolio.Asset

	// Extract unique base currencies
	for _, pair := range pairs {
		// Only include USDT pairs for simplicity
		if pair.Quote == "USDT" && pair.TradeStatus == "tradable" {
			if !assetMap[pair.Base] {
				assetMap[pair.Base] = true
				assets = append(assets, portfolio.NewAsset(pair.Base))
			}
		}
	}

	return assets, nil
}

// FetchAvailablePerpetualAssets returns empty slice (not supported in spot)
func (g *gateSpot) FetchAvailablePerpetualAssets() ([]portfolio.Asset, error) {
	// Spot connector doesn't support perpetuals
	return []portfolio.Asset{}, nil
}
