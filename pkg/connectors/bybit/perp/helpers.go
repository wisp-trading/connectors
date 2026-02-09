package perp

import (
	"fmt"
	"strconv"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (b *bybit) GetPerpSymbol(pair portfolio.Pair) string {
	return pair.Base().Symbol() + pair.Quote().Symbol()
}

func (b *bybit) symbolToPair(symbol string) (portfolio.Pair, error) {
	// Bybit symbols are in the format "BTCUSDT", "ETHUSDT", etc.

	quoteAsset := "USDT"
	if len(symbol) <= len(quoteAsset) {
		return portfolio.Pair{}, fmt.Errorf("invalid symbol format: %s", symbol)
	}

	baseAsset := symbol[:len(symbol)-len(quoteAsset)]
	return portfolio.NewPair(portfolio.NewAsset(baseAsset), portfolio.NewAsset(quoteAsset)), nil
}

func (b *bybit) parseKline(data []interface{}) connector.Kline {
	kline := connector.Kline{}

	if len(data) >= 7 {
		if openTimeStr, ok := data[0].(string); ok {
			if timestamp, err := strconv.ParseInt(openTimeStr, 10, 64); err == nil {
				kline.OpenTime = time.Unix(timestamp/1000, (timestamp%1000)*1000000)
			}
		}
		if openStr, ok := data[1].(string); ok {
			if val, err := strconv.ParseFloat(openStr, 64); err == nil {
				kline.Open = val
			}
		}
		if highStr, ok := data[2].(string); ok {
			if val, err := strconv.ParseFloat(highStr, 64); err == nil {
				kline.High = val
			}
		}
		if lowStr, ok := data[3].(string); ok {
			if val, err := strconv.ParseFloat(lowStr, 64); err == nil {
				kline.Low = val
			}
		}
		if closeStr, ok := data[4].(string); ok {
			if val, err := strconv.ParseFloat(closeStr, 64); err == nil {
				kline.Close = val
			}
		}
		if volumeStr, ok := data[5].(string); ok {
			if val, err := strconv.ParseFloat(volumeStr, 64); err == nil {
				kline.Volume = val
			}
		}
	}

	return kline
}

func (b *bybit) parseFundingRate(data map[string]interface{}) (portfolio.Pair, perp.FundingRate, error) {
	var rate perp.FundingRate

	symbol, ok := data["symbol"].(string)
	if !ok {
		return portfolio.Pair{}, rate, fmt.Errorf("missing or invalid symbol")
	}

	pair, err := b.symbolToPair(symbol)
	if err != nil {
		return portfolio.Pair{}, rate, fmt.Errorf("failed to parse symbol %s: %w", symbol, err)
	}

	fundingRateStr, ok := data["fundingRate"].(string)
	if !ok {
		return pair, rate, fmt.Errorf("missing or invalid fundingRate")
	}

	fundingRateVal, err := numerical.NewFromString(fundingRateStr)
	if err != nil {
		return pair, rate, fmt.Errorf("failed to parse funding rate: %w", err)
	}

	rate.CurrentRate = fundingRateVal
	rate.Timestamp = b.timeProvider.Now()
	rate.NextFundingTime = b.timeProvider.Now()

	return pair, rate, nil
}

func (b *bybit) parseTrade(data map[string]interface{}, pair portfolio.Pair) connector.Trade {
	trade := connector.Trade{
		Pair:      pair,
		Timestamp: b.timeProvider.Now(),
	}

	if side, ok := data["side"].(string); ok {
		trade.Side = connector.OrderSide(side)
	}
	if price, ok := data["price"].(string); ok {
		if val, err := numerical.NewFromString(price); err == nil {
			trade.Price = val
		}
	}
	if quantity, ok := data["size"].(string); ok {
		if val, err := numerical.NewFromString(quantity); err == nil {
			trade.Quantity = val
		}
	}

	return trade
}

func (b *bybit) parseOrder(data map[string]interface{}) connector.Order {
	order := connector.Order{
		UpdatedAt: b.timeProvider.Now(),
	}

	if orderId, ok := data["orderId"].(string); ok {
		order.ID = orderId
	}
	if orderLinkId, ok := data["orderLinkId"].(string); ok {
		order.ClientOrderID = orderLinkId
	}
	if symbol, ok := data["symbol"].(string); ok {
		order.Symbol = symbol
		var err error
		order.Pair, err = b.symbolToPair(symbol)
		if err != nil {
			fmt.Printf("failed to parse symbol %s: %v\n", symbol, err)
		}
	}
	if side, ok := data["side"].(string); ok {
		order.Side = connector.FromString(side)
	}
	if orderType, ok := data["orderType"].(string); ok {
		order.Type = connector.OrderType(orderType)
	}
	if orderStatus, ok := data["orderStatus"].(string); ok {
		order.Status = connector.OrderStatus(orderStatus)
	}
	if qty, ok := data["qty"].(string); ok {
		if val, err := numerical.NewFromString(qty); err == nil {
			order.Quantity = val
		}
	}
	if price, ok := data["price"].(string); ok {
		if val, err := numerical.NewFromString(price); err == nil {
			order.Price = val
		}
	}
	if cumExecQty, ok := data["cumExecQty"].(string); ok {
		if val, err := numerical.NewFromString(cumExecQty); err == nil {
			order.FilledQty = val
		}
	}
	if leavesQty, ok := data["leavesQty"].(string); ok {
		if val, err := numerical.NewFromString(leavesQty); err == nil {
			order.RemainingQty = val
		}
	}
	if avgPrice, ok := data["avgPrice"].(string); ok {
		if val, err := numerical.NewFromString(avgPrice); err == nil {
			order.AvgPrice = val
		}
	}
	if createdTime, ok := data["createdTime"].(string); ok {
		if timestamp, err := numerical.NewFromString(createdTime); err == nil {
			order.CreatedAt = time.UnixMilli(timestamp.IntPart())
		}
	}
	if updatedTime, ok := data["updatedTime"].(string); ok {
		if timestamp, err := numerical.NewFromString(updatedTime); err == nil {
			order.UpdatedAt = time.UnixMilli(timestamp.IntPart())
		}
	}

	return order
}

func (b *bybit) parsePosition(data map[string]interface{}) perp.Position {
	pos := perp.Position{
		UpdatedAt: b.timeProvider.Now(),
	}

	if symbol, ok := data["symbol"].(string); ok {
		var err error
		pos.Pair, err = b.symbolToPair(symbol)

		if err != nil {
			// If symbol parsing fails, we can log the error and skip this position
			fmt.Printf("failed to parse symbol %s: %v\n", symbol, err)
		}
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
