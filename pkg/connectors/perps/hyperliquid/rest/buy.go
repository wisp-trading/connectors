package rest

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (t *tradingService) PlaceBuyLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, true, nil)
}

func (t *tradingService) PlaceBuyMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.MarketOpen(coin, true, size, nil, slippage, nil, nil)
}

func (t *tradingService) PlaceBuyStopLoss(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error) {
	return t.placeTriggerOrder(coin, size, triggerPrice, true, true)
}

func (t *tradingService) PlaceBuyTakeProfit(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error) {
	return t.placeTriggerOrder(coin, size, triggerPrice, true, true)
}

func (t *tradingService) PlaceBuyLimitOrderWithCustomRef(coin string, size, price float64, customRef string) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, true, &customRef)
}
