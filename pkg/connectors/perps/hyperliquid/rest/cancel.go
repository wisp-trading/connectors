package rest

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (t *tradingService) CancelOrderByID(coin string, orderID int64) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return nil, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.Cancel(coin, orderID)
}

func (t *tradingService) CancelOrderByCustomRef(coin, customRef string) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return nil, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.CancelByCloid(coin, customRef)
}
