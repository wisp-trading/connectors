package rest

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (t *tradingService) placeLimitOrder(coin string, size, price float64, isBuy bool, clientOrderID *string) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}

	// Round price to valid tick size
	roundedPrice, err := t.priceValidator.RoundPrice(coin, price)
	if err != nil {
		// If validation fails, use original price and let API reject it with clear error
		fmt.Printf("Warning: failed to validate price for %s: %v, using original price\n", coin, err)
		roundedPrice = price
	}

	// Round size to valid decimals
	roundedSize, err := t.priceValidator.RoundSize(coin, size)
	if err != nil {
		fmt.Printf("Warning: failed to validate size for %s: %v, using original size\n", coin, err)
		roundedSize = size
	}

	// Log if rounding occurred
	if roundedPrice != price {
		fmt.Printf("Price rounded for %s: %.6f -> %.6f (tick size: ", coin, price, roundedPrice)
		if tickSize, err := t.priceValidator.GetTickSize(coin); err == nil {
			fmt.Printf("%.6f)\n", tickSize)
		} else {
			fmt.Printf("unknown)\n")
		}
	}
	if roundedSize != size {
		fmt.Printf("Size rounded for %s: %.6f -> %.6f\n", coin, size, roundedSize)
	}

	req := hyperliquid.CreateOrderRequest{
		Coin:       coin,
		IsBuy:      isBuy,
		Price:      roundedPrice,
		Size:       roundedSize,
		ReduceOnly: false,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
		},
		ClientOrderID: clientOrderID,
	}

	return ex.Order(req, nil)
}

func (t *tradingService) placeTriggerOrder(coin string, size, triggerPrice float64, isBuy bool, isMarket bool) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}

	// Round trigger price to valid tick size
	roundedPrice, err := t.priceValidator.RoundPrice(coin, triggerPrice)
	if err != nil {
		roundedPrice = triggerPrice
	}

	// Round size to valid decimals
	roundedSize, err := t.priceValidator.RoundSize(coin, size)
	if err != nil {
		roundedSize = size
	}

	req := hyperliquid.CreateOrderRequest{
		Coin:       coin,
		IsBuy:      isBuy,
		Price:      roundedPrice,
		Size:       roundedSize,
		ReduceOnly: false,
		OrderType: hyperliquid.OrderType{
			Trigger: &hyperliquid.TriggerOrderType{
				TriggerPx: roundedPrice,
				IsMarket:  isMarket,
			},
		},
	}

	return ex.Order(req, nil)
}
