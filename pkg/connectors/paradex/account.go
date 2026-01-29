package paradex

import (
	"context"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *paradex) GetAccountBalance() (*connector.AccountBalance, error) {
	account, err := p.paradexService.GetAccount(p.ctx)
	if err != nil {
		return nil, err
	}

	// Parse all needed fields
	accountValue, _ := numerical.NewFromString(account.AccountValue)
	totalCollateral, _ := numerical.NewFromString(account.TotalCollateral)
	freeCollateral, _ := numerical.NewFromString(account.FreeCollateral)
	initialMargin, _ := numerical.NewFromString(account.InitialMarginRequirement)
	currency := account.SettlementAsset
	if currency == "" {
		currency = "USD"
	}

	// Calculations
	usedMargin := initialMargin
	unrealizedPnL := accountValue.Sub(totalCollateral)

	updatedAt := time.Now()
	if account.UpdatedAt > 0 {
		updatedAt = time.UnixMilli(account.UpdatedAt)
	}

	return &connector.AccountBalance{
		TotalBalance:     accountValue,   // account_value (includes unrealized PnL)
		AvailableBalance: freeCollateral, // free_collateral
		UsedMargin:       usedMargin,     // initial_margin_requirement
		UnrealizedPnL:    unrealizedPnL,  // account_value - total_collateral
		Currency:         currency,
		UpdatedAt:        updatedAt,
	}, nil
}

func (p *paradex) GetPositions() ([]connector.Position, error) {
	positionsResp, err := p.paradexService.GetUserPositions(p.ctx) // returns *models.ResponsesGetPositionsResp
	if err != nil {
		return nil, err
	}

	var result []connector.Position
	for _, pos := range positionsResp.Results {
		size, _ := numerical.NewFromString(pos.Size)
		entryPrice, _ := numerical.NewFromString(pos.AverageEntryPrice)
		unrealizedPnL, _ := numerical.NewFromString(pos.UnrealizedPnl)

		// MarkPrice is not in the paradex API, so set to zero
		var markPrice numerical.Decimal

		realizedPnL, _ := numerical.NewFromString(pos.RealizedPositionalPnl)
		updatedAt := time.UnixMilli(pos.LastUpdatedAt)

		result = append(result, connector.Position{
			Symbol:        portfolio.NewAsset(pos.Market),
			Exchange:      p.GetConnectorInfo().Name,
			Side:          connector.OrderSide(pos.Side),
			Size:          size,
			EntryPrice:    entryPrice,
			MarkPrice:     markPrice,
			UnrealizedPnL: unrealizedPnL,
			RealizedPnL:   realizedPnL,
			UpdatedAt:     updatedAt,
		})
	}

	return result, nil
}

// GetSubAccounts returns all sub-accounts for the current account
func (p *paradex) GetSubAccounts(ctx context.Context) (interface{}, error) {
	return p.paradexService.GetSubAccounts(ctx)
}

// GetAccountInfo returns account information
func (p *paradex) GetAccountInfo(ctx context.Context) (interface{}, error) {
	return p.paradexService.GetAccountInfo(ctx)
}
