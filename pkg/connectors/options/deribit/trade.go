package deribit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	optionsConnector "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// placeLimitOrder places a limit order using private/buy endpoint
// Per Deribit spec: https://docs.deribit.com/api-reference/trading/private-buy
func (d *deribitOptions) placeLimitOrder(ctx context.Context, pair portfolio.Pair, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	qty, _ := quantity.Float64()
	prc, _ := price.Float64()

	// Format instrument name per Deribit spec (BTC-31DEC25-50000-C)
	// Note: In production, would need full OptionContract details
	instrumentName := formatInstrumentName(optionsConnector.OptionContract{
		Pair: pair,
	})

	// Determine method (private/buy or private/sell)
	method := "private/buy"
	if side == connector.OrderSideSell {
		method = "private/sell"
	}

	// Call Deribit API per spec
	result, err := d.client.Call(ctx, method, map[string]interface{}{
		"instrument_name": instrumentName,
		"amount":          qty,
		"price":           prc,
		"type":            "limit",
	})
	if err != nil {
		return nil, fmt.Errorf("%s failed: %w", method, err)
	}

	var orderData struct {
		OrderID        string  `json:"order_id"`
		InstrumentName string  `json:"instrument_name"`
		OrderState     string  `json:"order_state"`
		OrderType      string  `json:"order_type"`
		Direction      string  `json:"direction"`
		Price          float64 `json:"price"`
		Amount         float64 `json:"amount"`
		FilledAmount   float64 `json:"filled_amount"`
		AveragePrice   float64 `json:"average_price"`
		CreationTime   int64   `json:"creation_timestamp"`
	}

	if err := json.Unmarshal(result, &orderData); err != nil {
		return nil, fmt.Errorf("failed to parse order response: %w", err)
	}

	return &connector.OrderResponse{
		OrderID:   orderData.OrderID,
		Symbol:    orderData.InstrumentName,
		Status:    mapOrderState(orderData.OrderState),
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  numerical.NewFromFloat(orderData.Amount),
		Price:     numerical.NewFromFloat(orderData.Price),
		FilledQty: numerical.NewFromFloat(orderData.FilledAmount),
		AvgPrice:  numerical.NewFromFloat(orderData.AveragePrice),
		Timestamp: time.UnixMilli(orderData.CreationTime),
	}, nil
}

// placeMarketOrder places a market order using private/buy or private/sell
// Per Deribit spec with type="market"
func (d *deribitOptions) placeMarketOrder(ctx context.Context, pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	qty, _ := quantity.Float64()

	instrumentName := formatInstrumentName(optionsConnector.OptionContract{
		Pair: pair,
	})

	method := "private/buy"
	if side == connector.OrderSideSell {
		method = "private/sell"
	}

	result, err := d.client.Call(ctx, method, map[string]interface{}{
		"instrument_name": instrumentName,
		"amount":          qty,
		"type":            "market",
	})
	if err != nil {
		return nil, fmt.Errorf("%s failed: %w", method, err)
	}

	var orderData struct {
		OrderID        string  `json:"order_id"`
		InstrumentName string  `json:"instrument_name"`
		OrderState     string  `json:"order_state"`
		OrderType      string  `json:"order_type"`
		Amount         float64 `json:"amount"`
		FilledAmount   float64 `json:"filled_amount"`
		AveragePrice   float64 `json:"average_price"`
		CreationTime   int64   `json:"creation_timestamp"`
	}

	if err := json.Unmarshal(result, &orderData); err != nil {
		return nil, fmt.Errorf("failed to parse order response: %w", err)
	}

	return &connector.OrderResponse{
		OrderID:   orderData.OrderID,
		Symbol:    orderData.InstrumentName,
		Status:    mapOrderState(orderData.OrderState),
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  numerical.NewFromFloat(orderData.Amount),
		FilledQty: numerical.NewFromFloat(orderData.FilledAmount),
		AvgPrice:  numerical.NewFromFloat(orderData.AveragePrice),
		Timestamp: time.UnixMilli(orderData.CreationTime),
	}, nil
}

// cancelOrder cancels an open order using private/cancel endpoint
// Per Deribit spec: https://docs.deribit.com/api-reference/trading/private-cancel
func (d *deribitOptions) cancelOrder(ctx context.Context, orderID string) (*connector.CancelResponse, error) {
	result, err := d.client.Call(ctx, "private/cancel", map[string]interface{}{
		"order_id": orderID,
	})
	if err != nil {
		return nil, fmt.Errorf("private/cancel failed: %w", err)
	}

	var orderData struct {
		OrderID        string `json:"order_id"`
		InstrumentName string `json:"instrument_name"`
		OrderState     string `json:"order_state"`
		LastUpdateTime int64  `json:"last_update_timestamp"`
	}

	if err := json.Unmarshal(result, &orderData); err != nil {
		return nil, fmt.Errorf("failed to parse cancel response: %w", err)
	}

	// Parse instrument name to extract pair
	pair, err := parseInstrumentName(orderData.InstrumentName, portfolio.NewAsset("USDT"))
	if err != nil {
		d.appLogger.Warn("failed to parse instrument name", map[string]interface{}{
			"instrument": orderData.InstrumentName,
			"error":      err,
		})
		// Don't fail - use empty pair
		pair = optionsConnector.OptionContract{
			Pair: portfolio.NewPair(portfolio.NewAsset(""), portfolio.NewAsset("USDT")),
		}
	}

	return &connector.CancelResponse{
		OrderID:   orderData.OrderID,
		Pair:      pair.Pair,
		Status:    mapOrderState(orderData.OrderState),
		Timestamp: time.UnixMilli(orderData.LastUpdateTime),
	}, nil
}

// mapOrderState converts Deribit order state to SDK OrderStatus
func mapOrderState(state string) connector.OrderStatus {
	switch strings.ToLower(state) {
	case "open":
		return connector.OrderStatusOpen
	case "filled":
		return connector.OrderStatusFilled
	case "cancelled":
		return connector.OrderStatusCanceled
	case "partially_filled":
		return connector.OrderStatusPartiallyFilled
	case "rejected":
		return connector.OrderStatusRejected
	default:
		return connector.OrderStatusNew
	}
}
