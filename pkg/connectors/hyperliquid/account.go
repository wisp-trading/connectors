package hyperliquid

import (
	"fmt"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (h *hyperliquid) GetAccountBalance() (*connector.AccountBalance, error) {
	userState, err := h.marketData.GetUserState(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	balance := &connector.AccountBalance{
		TotalBalance:     parseDecimal(userState.MarginSummary.AccountValue),
		AvailableBalance: parseDecimal(userState.Withdrawable),
		UsedMargin:       parseDecimal(userState.MarginSummary.TotalMarginUsed),
		UnrealizedPnL:    parseDecimal(userState.MarginSummary.TotalNtlPos),
		Currency:         "USD",
		UpdatedAt:        h.timeProvider.Now(),
	}

	return balance, nil
}

// GetPositions retrieves all positions from UserState and remaps them to connector.Position
func (h *hyperliquid) GetPositions() ([]connector.Position, error) {
	userState, err := h.marketData.GetUserState(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get user state: %w", err)
	}

	if len(userState.AssetPositions) == 0 {
		return []connector.Position{}, nil
	}

	var positions []connector.Position

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

		positions = append(positions, connector.Position{
			Exchange:         types.Hyperliquid,
			Symbol:           portfolio.NewAsset(pos.Coin),
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

// GetTradingHistory retrieves trading history for the specified symbol
func (h *hyperliquid) GetTradingHistory(symbol string, limit int) ([]connector.Trade, error) {
	fills, err := h.marketData.GetUserFills(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get user fills: %w", err)
	}

	trades := make([]connector.Trade, 0, limit)
	for _, fill := range fills {
		// Only include fills for the requested symbol
		if fill.Coin != symbol {
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
			Symbol:    fill.Coin,
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
