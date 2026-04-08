package deribit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// fetchAccountSummary response represents the result from private/get_account_summary endpoint
type accountSummaryData struct {
	Equity            float64       `json:"equity"`
	Balance           float64       `json:"balance"`
	AvailableFunds    float64       `json:"available_funds"`
	AvailableWithdraw float64       `json:"available_withdraw"`
	Positions         []interface{} `json:"positions"`
	UpdateTime        int64         `json:"timestamp"`
}

// fetchUserTrades response represents the result from private/get_user_trades_by_instrument endpoint
type userTradeData struct {
	TradeID        string  `json:"trade_id"`
	OrderID        string  `json:"order_id"`
	InstrumentName string  `json:"instrument_name"`
	Direction      string  `json:"direction"` // buy or sell
	Price          float64 `json:"price"`
	Amount         float64 `json:"amount"`
	Fee            float64 `json:"fee"`
	Timestamp      int64   `json:"timestamp"`
	MakerOrTaker   string  `json:"liquidity"`
}

// fetchAccountSummary retrieves the account summary including balances
// Per Deribit spec: https://docs.deribit.com/api-reference/account/private-get_account_summary
func (d *deribitOptions) fetchAccountSummary(ctx context.Context) (*accountSummaryData, error) {
	result, err := d.client.Call(ctx, "private/get_account_summary", map[string]interface{}{
		"extended": true,
	})
	if err != nil {
		return nil, fmt.Errorf("private/get_account_summary failed: %w", err)
	}

	var accountData accountSummaryData
	if err := json.Unmarshal(result, &accountData); err != nil {
		return nil, fmt.Errorf("failed to parse account summary: %w", err)
	}

	return &accountData, nil
}

// fetchUserTrades retrieves the user's trading history for an instrument
// Per Deribit spec: https://docs.deribit.com/api-reference/account/private-get_user_trades_by_instrument
func (d *deribitOptions) fetchUserTrades(ctx context.Context, instrumentName string, limit int) ([]userTradeData, error) {
	result, err := d.client.Call(ctx, "private/get_user_trades_by_instrument", map[string]interface{}{
		"instrument_name": instrumentName,
		"count":           limit,
		"include_old":     true,
	})
	if err != nil {
		return nil, fmt.Errorf("private/get_user_trades_by_instrument failed: %w", err)
	}

	var trades []userTradeData
	if err := json.Unmarshal(result, &trades); err != nil {
		return nil, fmt.Errorf("failed to parse user trades: %w", err)
	}

	return trades, nil
}

// buildAssetBalance creates an AssetBalance from account data
func (d *deribitOptions) buildAssetBalance(accountData *accountSummaryData, asset portfolio.Asset) *connector.AssetBalance {
	return &connector.AssetBalance{
		Asset:     asset,
		Free:      numerical.NewFromFloat(accountData.AvailableFunds),
		Locked:    numerical.NewFromFloat(accountData.Equity - accountData.AvailableFunds),
		Total:     numerical.NewFromFloat(accountData.Equity),
		UpdatedAt: time.UnixMilli(accountData.UpdateTime),
	}
}

// convertTradeToConnectorTrade converts Deribit trades to SDK Trade type
func convertTradeToConnectorTrade(tradeDataSlice []userTradeData, pair portfolio.Pair) []connector.Trade {
	trades := make([]connector.Trade, 0, len(tradeDataSlice))

	for _, t := range tradeDataSlice {
		side := connector.OrderSideBuy
		if t.Direction == "sell" {
			side = connector.OrderSideSell
		}

		isMaker := t.MakerOrTaker == "M"

		trade := connector.Trade{
			ID:        t.TradeID,
			OrderID:   t.OrderID,
			Pair:      pair,
			Price:     numerical.NewFromFloat(t.Price),
			Quantity:  numerical.NewFromFloat(t.Amount),
			Side:      side,
			IsMaker:   isMaker,
			Fee:       numerical.NewFromFloat(t.Fee),
			Timestamp: time.UnixMilli(t.Timestamp),
		}

		trades = append(trades, trade)
	}

	return trades
}
