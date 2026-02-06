package perp

import (
	"context"
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (b *bybit) GetBalances() ([]*perp.AssetBalance, error) {
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

	var balances []*perp.AssetBalance

	if result != nil && result.Result != nil {
		if resultData, ok := result.Result.(map[string]interface{}); ok {
			if listData, ok := resultData["list"].([]interface{}); ok && len(listData) > 0 {
				if accountData, ok := listData[0].(map[string]interface{}); ok {
					if coinData, ok := accountData["coin"].([]interface{}); ok {
						for _, c := range coinData {
							if coinInfo, ok := c.(map[string]interface{}); ok {
								balance := b.parseAssetBalance(coinInfo)
								balances = append(balances, balance)
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

func (b *bybit) parseAssetBalance(data map[string]interface{}) *perp.AssetBalance {
	balance := &perp.AssetBalance{
		AssetBalance: connector.AssetBalance{
			UpdatedAt: b.timeProvider.Now(),
		},
	}

	if coin, ok := data["coin"].(string); ok {
		balance.Asset = portfolio.NewAsset(coin)
	}
	if walletBalance, ok := data["walletBalance"].(string); ok {
		if val, err := numerical.NewFromString(walletBalance); err == nil {
			balance.Total = val
		}
	}
	if availableToWithdraw, ok := data["availableToWithdraw"].(string); ok {
		if val, err := numerical.NewFromString(availableToWithdraw); err == nil {
			balance.Free = val
		}
	}
	if locked, ok := data["locked"].(string); ok {
		if val, err := numerical.NewFromString(locked); err == nil {
			balance.Locked = val
		}
	}

	return balance
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

func (b *bybit) parsePosition(data map[string]interface{}) perp.Position {
	pos := perp.Position{
		UpdatedAt: b.timeProvider.Now(),
	}

	if symbol, ok := data["symbol"].(string); ok {
		// Extract base symbol from "BTCUSDT" format
		baseSymbol := symbol
		quoteSymbol := "USDT"
		if len(symbol) > 4 && symbol[len(symbol)-4:] == "USDT" {
			baseSymbol = symbol[:len(symbol)-4]
		}
		pos.Pair = portfolio.NewPair(portfolio.NewAsset(baseSymbol), portfolio.NewAsset(quoteSymbol))
	}
	if side, ok := data["side"].(string); ok {
		pos.Side = connector.OrderSide(side)
	}
	if size, ok := data["size"].(string); ok {
		if val, err := numerical.NewFromString(size); err == nil {
			pos.Size = val
		}
	}
	if avgPrice, ok := data["avgPrice"].(string); ok {
		if val, err := numerical.NewFromString(avgPrice); err == nil {
			pos.EntryPrice = val
		}
	}
	if markPrice, ok := data["markPrice"].(string); ok {
		if val, err := numerical.NewFromString(markPrice); err == nil {
			pos.MarkPrice = val
		}
	}
	if unrealizedPnl, ok := data["unrealisedPnl"].(string); ok {
		if val, err := numerical.NewFromString(unrealizedPnl); err == nil {
			pos.UnrealizedPnL = val
		}
	}
	if cumRealisedPnl, ok := data["cumRealisedPnl"].(string); ok {
		if val, err := numerical.NewFromString(cumRealisedPnl); err == nil {
			pos.RealizedPnL = val
		}
	}
	if leverage, ok := data["leverage"].(string); ok {
		if val, err := numerical.NewFromString(leverage); err == nil {
			pos.Leverage = val
		}
	}
	if liqPrice, ok := data["liqPrice"].(string); ok {
		if val, err := numerical.NewFromString(liqPrice); err == nil {
			pos.LiquidationPrice = val
		}
	}

	return pos
}

func (b *bybit) GetTradingHistory(symbol string, limit int) ([]connector.Trade, error) {
	client, err := b.client.GetClient()

	if err != nil {
		return nil, fmt.Errorf("client not configured: %w", err)
	}

	params := map[string]interface{}{
		"category": "linear",
		"symbol":   symbol,
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
							Symbol:    symbol,
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
	balances, err := b.GetBalances()
	if err != nil {
		return nil, err
	}
	result := make([]perp.AssetBalance, len(balances))
	for i, b := range balances {
		result[i] = *b
	}
	return result, nil
}
