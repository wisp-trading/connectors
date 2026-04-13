package hyperliquid

import (
	"fmt"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// GetBalances implements connector.AccountReader.
// Returns all spot token balances from Hyperliquid's SpotUserState endpoint.
// The SDK deserialises the spot clearinghouse state into a UserState struct
// where each AssetPosition represents a spot token holding.
func (h *hyperliquidSpot) GetBalances() ([]connector.AssetBalance, error) {
	state, err := h.marketData.FetchSpotUserState(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spot balances: %w", err)
	}

	balances := make([]connector.AssetBalance, 0, len(state.AssetPositions))
	for _, ap := range state.AssetPositions {
		pos := ap.Position
		total := parseDecimal(pos.Szi)
		if total.IsZero() {
			continue
		}

		balances = append(balances, connector.AssetBalance{
			Asset:     portfolio.NewAsset(pos.Coin),
			Free:      total,
			Locked:    numerical.Zero(),
			Total:     total,
			UpdatedAt: time.Now(),
		})
	}

	return balances, nil
}

// GetBalance implements connector.AccountReader.
// Returns the balance for a specific spot asset.
func (h *hyperliquidSpot) GetBalance(asset portfolio.Asset) (*connector.AssetBalance, error) {
	balances, err := h.GetBalances()
	if err != nil {
		return nil, err
	}

	coin := h.normaliseAssetName(asset)
	for _, b := range balances {
		if h.normaliseAssetName(b.Asset) == coin {
			return &b, nil
		}
	}

	return nil, fmt.Errorf("no balance found for spot asset %s", coin)
}

// GetTradingHistory implements connector.AccountReader.
func (h *hyperliquidSpot) GetTradingHistory(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	return nil, fmt.Errorf("GetTradingHistory not yet implemented for Hyperliquid spot")
}
