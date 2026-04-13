package hyperliquid

import (
	"fmt"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (h *hyperliquid) GetBalances() ([]connector.AssetBalance, error) {
	// Delegate to GetMarginBalances and extract base AssetBalance
	marginBalances, err := h.GetMarginBalances()
	if err != nil {
		return nil, err
	}

	balances := make([]connector.AssetBalance, len(marginBalances))
	for i, mb := range marginBalances {
		balances[i] = mb.AssetBalance
	}
	return balances, nil
}

func (h *hyperliquid) GetMarginBalances() ([]perp.AssetBalance, error) {
	userState, err := h.marketData.GetUserState(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	balances := make([]perp.AssetBalance, 0, len(userState.AssetPositions)+1)

	// Add USDC balance (the only withdrawable asset)
	withdrawable := parseDecimal(userState.Withdrawable)
	totalAccountValue := parseDecimal(userState.MarginSummary.AccountValue)
	totalMarginUsed := parseDecimal(userState.MarginSummary.TotalMarginUsed)

	balances = append(balances, perp.AssetBalance{
		AssetBalance: connector.AssetBalance{
			Asset:     portfolio.NewAsset("USDC"),
			Free:      withdrawable,
			Locked:    totalMarginUsed,
			Total:     totalAccountValue,
			UpdatedAt: h.timeProvider.Now(),
		},
		UsedMargin:    totalMarginUsed,
		UnrealizedPnL: parseDecimal(userState.MarginSummary.TotalNtlPos),
	})

	// Add balance for each asset with an open position
	for _, assetPos := range userState.AssetPositions {
		pos := assetPos.Position

		positionValue := parseDecimal(pos.PositionValue)
		marginUsed := parseDecimal(pos.MarginUsed)
		unrealizedPnl := parseDecimal(pos.UnrealizedPnl)

		balances = append(balances, perp.AssetBalance{
			AssetBalance: connector.AssetBalance{
				Asset:     portfolio.NewAsset(pos.Coin),
				Free:      numerical.Zero(),
				Locked:    numerical.Zero(),
				Total:     positionValue,
				UpdatedAt: h.timeProvider.Now(),
			},
			UsedMargin:    marginUsed,
			UnrealizedPnL: unrealizedPnl,
		})
	}

	return balances, nil
}

func (h *hyperliquid) GetBalance(asset portfolio.Asset) (*connector.AssetBalance, error) {
	marginBalances, err := h.GetMarginBalances()
	if err != nil {
		return nil, err
	}

	for _, mb := range marginBalances {
		if mb.Asset.Symbol() == asset.Symbol() {
			return &mb.AssetBalance, nil
		}
	}

	// Asset not found - return zero balance
	return &connector.AssetBalance{
		Asset:     asset,
		Free:      numerical.Zero(),
		Locked:    numerical.Zero(),
		Total:     numerical.Zero(),
		UpdatedAt: h.timeProvider.Now(),
	}, nil
}

// GetPositions retrieves all positions from UserState and remaps them to connector.Position
func (h *hyperliquid) GetPositions() ([]perp.Position, error) {
	userState, err := h.marketData.GetUserState(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	if len(userState.AssetPositions) == 0 {
		return []perp.Position{}, nil
	}

	var positions []perp.Position

	for _, assetPos := range userState.AssetPositions {
		pos := assetPos.Position

		// Simple decimal conversion - defaults to zero on error
		positionSize := parseDecimal(pos.Szi)
		unrealizedPnL := parseDecimal(pos.UnrealizedPnl)
		leverage := numerical.NewFromInt(int64(pos.Leverage.Value))

		var entryPrice numerical.Decimal
		if pos.EntryPx != nil {
			entryPrice = parseDecimal(*pos.EntryPx)
		}

		var liquidationPrice numerical.Decimal
		if pos.LiquidationPx != nil {
			liquidationPrice = parseDecimal(*pos.LiquidationPx)
		}

		markPrice := parseDecimal(pos.PositionValue)

		// Determine side based on position size
		var side connector.OrderSide
		if positionSize.IsPositive() {
			side = connector.OrderSideBuy
		} else if positionSize.IsNegative() {
			side = connector.OrderSideSell
		} else {
			side = connector.OrderSideBuy
		}

		positions = append(positions, perp.Position{
			Exchange:         types.Hyperliquid,
			Pair:             h.coinToPair(pos.Coin),
			Side:             side,
			Size:             positionSize.Abs(),
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			UnrealizedPnL:    unrealizedPnL,
			RealizedPnL:      numerical.Zero(),
			Leverage:         leverage,
			MarginType:       pos.Leverage.Type,
			LiquidationPrice: liquidationPrice,
			UpdatedAt:        h.timeProvider.Now(),
		})
	}

	return positions, nil
}

// GetTradingHistory retrieves trading history for the specified pair
func (h *hyperliquid) GetTradingHistory(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	fills, err := h.marketData.GetUserFills(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get user fills: %w", err)
	}

	trades := make([]connector.Trade, 0, limit)
	for _, fill := range fills {
		// Only include fills for the requested symbol
		if fill.Coin != pair.Base().Symbol() {
			continue
		}

		if len(trades) >= limit {
			break
		}

		price := parseDecimal(fill.Price)
		quantity := parseDecimal(fill.Size)

		// Determine side from fill.Side ("A" = ask/sell, "B" = bid/buy)
		var side connector.OrderSide
		if fill.Side == "B" {
			side = connector.OrderSideBuy
		} else {
			side = connector.OrderSideSell
		}

		trades = append(trades, connector.Trade{
			ID:        fmt.Sprintf("%d", fill.Oid),
			OrderID:   fmt.Sprintf("%d", fill.Oid),
			Pair:      h.coinToPair(fill.Coin),
			Side:      side,
			Price:     price,
			Quantity:  quantity,
			Fee:       numerical.Zero(),             // Hyperliquid doesn't provide fee in Fill
			Timestamp: time.Unix(fill.Time/1000, 0), // Convert milliseconds to seconds
			IsMaker:   false,                        // Can't determine from Fill struct
		})
	}

	return trades, nil
}
