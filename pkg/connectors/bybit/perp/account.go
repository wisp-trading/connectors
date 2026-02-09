package perp

import (
	"context"
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (b *bybit) GetBalances() ([]connector.AssetBalance, error) {
	client, err := b.client.GetClient()
	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"accountType": "UNIFIED",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetAccountWallet(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet balance: %w", err)
	}

	var balances []connector.AssetBalance

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok && len(listData) > 0 {
				if accountData, ok := listData[0].(map[string]interface{}); ok {
					if coinData, ok := accountData["coin"].([]interface{}); ok {
						for _, c := range coinData {
							if coinInfo, ok := c.(map[string]interface{}); ok {
								balance := b.parseAssetBalance(coinInfo)
								balances = append(balances, balance.AssetBalance)
							}
						}
					}
				}
			}
		}
	}

	return balances, nil
}

func (b *bybit) GetBalance(asset portfolio.Asset) (*connector.AssetBalance, error) {
	client, err := b.client.GetClient()
	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"accountType": "UNIFIED",
		"coin":        asset.Symbol(), // Query specific coin directly
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetAccountWallet(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get balance for %s: %w", asset.Symbol(), err)
	}

	// Parse the response for the specific coin
	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok && len(listData) > 0 {
				if accountData, ok := listData[0].(map[string]interface{}); ok {
					if coinData, ok := accountData["coin"].([]interface{}); ok && len(coinData) > 0 {
						if coinInfo, ok := coinData[0].(map[string]interface{}); ok {
							balance := b.parseAssetBalance(coinInfo)
							return &balance.AssetBalance, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("balance not found for asset: %s", asset.Symbol())
}

func (b *bybit) GetPositions() ([]perp.Position, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category":   "linear",
		"settleCoin": "USDT",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetPositionList(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var positions []perp.Position

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if posData, ok := item.(map[string]interface{}); ok {
						pos := b.parsePosition(posData)
						if !pos.Size.IsZero() {
							positions = append(positions, pos)
						}
					}
				}
			}
		}
	}

	return positions, nil
}

func (b *bybit) GetTradingHistory(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   b.GetPerpSymbol(pair),
		"limit":    limit,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetTransactionLog(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trading history: %w", err)
	}

	var trades []connector.Trade

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if tradeData, ok := item.(map[string]interface{}); ok {
						trade := connector.Trade{
							Pair:      pair,
							Timestamp: b.timeProvider.Now(),
						}

						if side, ok := tradeData["side"].(string); ok {
							trade.Side = connector.OrderSide(side)
						}
						if price, ok := tradeData["execPrice"].(string); ok {
							if val, err := numerical.NewFromString(price); err == nil {
								trade.Price = val
							}
						}
						if qty, ok := tradeData["execQty"].(string); ok {
							if val, err := numerical.NewFromString(qty); err == nil {
								trade.Quantity = val
							}
						}

						trades = append(trades, trade)
					}
				}
			}
		}
	}

	return trades, nil
}

func (b *bybit) FetchRecentTrades(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   b.GetPerpSymbol(pair),
		"limit":    limit,
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetPublicRecentTrades(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent trades: %w", err)
	}

	var trades []connector.Trade
	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok {
				for _, item := range listData {
					if tradeData, ok := item.(map[string]interface{}); ok {
						trade := b.parseTrade(tradeData, pair)
						trades = append(trades, trade)
					}
				}
			}
		}
	}

	return trades, nil
}

func (b *bybit) GetMarginBalances() ([]perp.AssetBalance, error) {
	client, err := b.client.GetClient()
	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"accountType": "UNIFIED",
	}

	result, err := client.NewUtaBybitServiceWithParams(params).GetAccountWallet(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet balance: %w", err)
	}

	var balances []perp.AssetBalance

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok && len(listData) > 0 {
				if accountData, ok := listData[0].(map[string]interface{}); ok {
					if coinData, ok := accountData["coin"].([]interface{}); ok {
						for _, c := range coinData {
							if coinInfo, ok := c.(map[string]interface{}); ok {
								balance := b.parseAssetBalance(coinInfo)
								balances = append(balances, *balance)
							}
						}
					}
				}
			}
		}
	}

	return balances, nil
}
