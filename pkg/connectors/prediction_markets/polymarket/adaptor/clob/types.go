package clob

import (
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

// Order represents a Polymarket CLOB order for signing
type OrderRequest struct {
	Taker         string `json:"taker"`
	TokenID       string `json:"tokenId"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Side          string `json:"side"`
	FeeRateBps    string `json:"feeRateBps"`
	Nonce         string `json:"nonce"`
	SignatureType int    `json:"signatureType"`
	Expiration    int64  `json:"expiration"`
}

func (OrderRequest) FromLimitOrder(order prediction.LimitOrder) OrderRequest {
	maker, taker := calculateAmounts(order)

	return OrderRequest{
		TokenID:     order.Outcome.OutcomeId,
		MakerAmount: maker,
		TakerAmount: taker,
		Side:        string(order.Side),
		Expiration:  order.Expiration,
	}
}

// CancelOrderRequest represents a request to cancel an order
type CancelOrderRequest struct {
	OrderID string `json:"orderID"`
}

// CancelOrderResponse represents the response from cancelling an order
type CancelOrderResponse struct {
	OrderID string `json:"orderID"`
	Status  string `json:"status"` // "CANCELLED"
	Success bool   `json:"success"`
}

// CancelAllOrdersRequest represents a request to cancel all orders for a market
type CancelAllOrdersRequest struct {
	MarketID string `json:"marketID,omitempty" validate:"required_without=AssetID"`
	AssetID  string `json:"assetID,omitempty" validate:"required_without=MarketID"`
}

// CancelAllOrdersResponse represents the response from cancelling all orders
type CancelAllOrdersResponse struct {
	Cancelled []string `json:"cancelled"` // List of cancelled order IDs
	Success   bool     `json:"success"`
}
