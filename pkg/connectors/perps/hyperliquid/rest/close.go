package rest

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (t *tradingService) ClosePosition(coin string, size *float64, slippage float64) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.MarketClose(coin, size, nil, slippage, nil, nil)
}

func (t *tradingService) CloseEntirePosition(coin string, slippage float64) (hyperliquid.OrderStatus, error) {
	return t.ClosePosition(coin, nil, slippage)
}
