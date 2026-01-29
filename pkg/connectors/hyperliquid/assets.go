package hyperliquid

import (
	"fmt"
	"strings"

	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

// FetchAvailableSpotAssets fetches all available spot assets from Hyperliquid
func (h *hyperliquid) FetchAvailableSpotAssets() ([]portfolio.Asset, error) {
	spotMeta, err := h.marketData.GetSpotMeta()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot meta: %w", err)
	}

	var assets []portfolio.Asset

	// Convert spot universe to models.Asset
	for _, spotAsset := range spotMeta.Tokens {
		asset := portfolio.NewAsset(baseSymbol(spotAsset.Name))
		assets = append(assets, asset)
	}

	return assets, nil
}

// FetchAvailablePerpetualAssets fetches all available perpetual assets from Hyperliquid
func (h *hyperliquid) FetchAvailablePerpetualAssets() ([]portfolio.Asset, error) {
	meta, err := h.marketData.GetMeta()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch perpetual meta: %w", err)
	}

	var assets []portfolio.Asset

	// Convert perpetual universe to models.Asset
	for _, perpAsset := range meta.Universe {
		asset := portfolio.NewAsset(baseSymbol(perpAsset.Name))
		assets = append(assets, asset)
	}

	return assets, nil
}

func baseSymbol(pair string) string {
	parts := strings.Split(pair, "/")
	return parts[0]
}
