package rest

import (
	"context"
	"fmt"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func (t *tradingService) PlaceSellLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, false, nil)
}

func (t *tradingService) PlaceSellMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return ex.MarketOpen(ctx, coin, false, size, nil, slippage, nil, nil)
}

func (t *tradingService) PlaceSellStopLoss(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error) {
	return t.placeTriggerOrder(coin, size, triggerPrice, false, true)
}

func (t *tradingService) PlaceSellTakeProfit(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error) {
	return t.placeTriggerOrder(coin, size, triggerPrice, false, true)
}

func (t *tradingService) PlaceSellLimitOrderWithCustomRef(coin string, size, price float64, customRef string) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, false, &customRef)
}
