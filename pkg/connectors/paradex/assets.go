package paradex

import (
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

func (p *paradex) FetchAvailableSpotAssets() ([]portfolio.Asset, error) {
	return []portfolio.Asset{}, nil
}

func (p *paradex) FetchAvailablePerpetualAssets() ([]portfolio.Asset, error) {
	markets, err := p.paradexService.GetMarkets(p.ctx)
	if err != nil {
		return nil, err
	}

	var assets []portfolio.Asset
	for _, market := range markets {
		if market.AssetKind != "PERP" {
			continue
		}
		asset := portfolio.NewAsset(market.BaseCurrency)
		assets = append(assets, asset)
	}

	return assets, nil
}
