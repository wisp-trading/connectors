package rest

import (
	"fmt"
	"math"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

func (t *spotTradingService) PlaceBuyLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, true)
}

func (t *spotTradingService) PlaceSellLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, false)
}

func (t *spotTradingService) PlaceBuyMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.MarketOpen(coin, true, roundToSigFigs(size, 5), nil, slippage, nil, nil)
}

func (t *spotTradingService) PlaceSellMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.MarketOpen(coin, false, roundToSigFigs(size, 5), nil, slippage, nil, nil)
}

// CancelOrderByID cancels a resting order. Uses ExchangeClient.CancelOrder
// directly to work around go-hyperliquid v0.5.0 serialising the OID as a
// JSON string instead of an integer.
func (t *spotTradingService) CancelOrderByID(coin string, orderID int64) error {
	info, err := t.infoClient.GetInfo()
	if err != nil {
		return fmt.Errorf("info client not configured: %w", err)
	}
	assetIndex := info.NameToAsset(coin)
	return t.client.CancelOrder(assetIndex, orderID)
}

func (t *spotTradingService) PlaceBulkOrders(orders []hyperliquid.CreateOrderRequest) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return nil, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.BulkOrders(orders, nil)
}

// placeLimitOrder places a limit order via the Go SDK.
// The SDK resolves coin -> asset index internally, including the spot offset.
// Prices and sizes are rounded to 5 significant figures to satisfy
// Hyperliquid's validation rules.
func (t *spotTradingService) placeLimitOrder(coin string, size, price float64, isBuy bool) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}

	req := hyperliquid.CreateOrderRequest{
		Coin:       coin,
		IsBuy:      isBuy,
		Price:      roundToSigFigs(price, 5),
		Size:       roundToSigFigs(size, 5),
		ReduceOnly: false,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
		},
	}

	return ex.Order(req, nil)
}

// roundToSigFigs rounds a number to n significant figures.
// Hyperliquid enforces a maximum of 5 significant figures on all prices and sizes.
func roundToSigFigs(num float64, sigFigs int) float64 {
	if num == 0 || sigFigs <= 0 {
		return num
	}
	magnitude := math.Floor(math.Log10(math.Abs(num)))
	multiplier := math.Pow(10, float64(sigFigs-1)-magnitude)
	return math.Round(num*multiplier) / multiplier
}
