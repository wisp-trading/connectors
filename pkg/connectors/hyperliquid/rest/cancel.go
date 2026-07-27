package rest

import (
	"context"
	"fmt"
	"time"

	"github.com/sonirico/go-hyperliquid"
)

func (t *tradingService) CancelOrderByID(coin string, orderID int64) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return nil, fmt.Errorf("exchange not configured: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return ex.Cancel(ctx, coin, orderID)
}

func (t *tradingService) CancelOrderByCustomRef(coin, customRef string) (*hyperliquid.APIResponse[hyperliquid.CancelOrderResponse], error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return nil, fmt.Errorf("exchange not configured: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return ex.CancelByCloid(ctx, coin, customRef)
}
