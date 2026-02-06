package paradex

import (
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *paradex) GetBalance(asset portfolio.Asset) (*connector.AssetBalance, error) {
	account, err := p.paradexService.GetAccount(p.ctx)
	if err != nil {
		return nil, err
	}

	// Parse all needed fields
	accountValue, _ := numerical.NewFromString(account.AccountValue)
	freeCollateral, _ := numerical.NewFromString(account.FreeCollateral)
	initialMargin, _ := numerical.NewFromString(account.InitialMarginRequirement)

	currency := account.SettlementAsset
	if currency == "" {
		currency = "USDC"
	}

	// Locked = used margin (initial margin requirement)
	locked := initialMargin
	// Free = free collateral
	free := freeCollateral
	// Total = account value (includes unrealized PnL)
	total := accountValue

	updatedAt := p.timeProvider.Now()
	if account.UpdatedAt > 0 {
		updatedAt = time.UnixMilli(account.UpdatedAt)
	}

	return &connector.AssetBalance{
		Asset:     portfolio.NewAsset(currency),
		Free:      free,   // Free collateral
		Locked:    locked, // Initial margin requirement
		Total:     total,  // Account value
		UpdatedAt: updatedAt,
	}, nil
}

func (p *paradex) GetBalances() ([]connector.AssetBalance, error) {
	balance, err := p.GetBalance(portfolio.NewAsset("USDC"))
	if err != nil {
		return nil, err
	}

	return []connector.AssetBalance{*balance}, nil
}

func (p *paradex) GetTradingHistory(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	symbol := p.GetPerpSymbol(pair)
	limit64 := int64(limit)
	tradesResp, err := p.paradexService.GetTradeHistory(p.ctx, &symbol, &limit64)

	if err != nil {
		return nil, err
	}

	var result []connector.Trade
	for _, trade := range tradesResp.Results {
		size, _ := numerical.NewFromString(trade.Size)
		price, _ := numerical.NewFromString(trade.Price)
		fee, _ := numerical.NewFromString(trade.Fee)
		timestamp := time.UnixMilli(trade.CreatedAt)

		result = append(result, connector.Trade{
			ID:        trade.ID,
			Pair:      pair,
			Exchange:  p.GetConnectorInfo().Name,
			Side:      p.convertOrderSide(trade.Side.ResponsesOrderSide),
			Quantity:  size,
			Price:     price,
			Fee:       fee,
			Timestamp: timestamp,
		})
	}

	return result, nil
}

func (p *paradex) GetPositions() ([]perp.Position, error) {
	positionsResp, err := p.paradexService.GetUserPositions(p.ctx) // returns *models.ResponsesGetPositionsResp
	if err != nil {
		return nil, err
	}

	var result []perp.Position
	for _, pos := range positionsResp.Results {
		pair, err := p.PerpSymbolToPair(pos.Market)

		if err != nil {
			// If we can't parse the symbol, skip this position
			continue
		}

		size, _ := numerical.NewFromString(pos.Size)
		entryPrice, _ := numerical.NewFromString(pos.AverageEntryPrice)
		unrealizedPnL, _ := numerical.NewFromString(pos.UnrealizedPnl)

		// MarkPrice is not in the paradex API, so set to zero
		var markPrice numerical.Decimal

		realizedPnL, _ := numerical.NewFromString(pos.RealizedPositionalPnl)
		updatedAt := time.UnixMilli(pos.LastUpdatedAt)

		result = append(result, perp.Position{
			Pair:          pair,
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

func (p *paradex) GetMarginBalances() ([]perp.AssetBalance, error) {
	//TODO implement me
	panic("implement me")
}
