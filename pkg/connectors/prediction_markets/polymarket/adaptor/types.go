package adaptor

import "time"

// OrderRequest represents a request to place an order on Polymarket
type OrderRequest struct {
	Salt          int64  `json:"salt"`
	Maker         string `json:"maker" validate:"required,eth_addr"`
	Signer        string `json:"signer" validate:"required,eth_addr"`
	Taker         string `json:"taker" validate:"required,eth_addr"`
	TokenID       string `json:"tokenId" validate:"required"`
	MakerAmount   string `json:"makerAmount" validate:"required"`
	TakerAmount   string `json:"takerAmount" validate:"required"`
	Side          string `json:"side" validate:"required,oneof=BUY SELL"`
	FeeRateBps    string `json:"feeRateBps"`
	Nonce         string `json:"nonce"`
	SignatureType int    `json:"signatureType"`
	Expiration    int64  `json:"expiration"`
	Signature     string `json:"signature"`
}

// OrderResponse represents the response from placing an order
type OrderResponse struct {
	OrderID       string    `json:"orderID"`
	MarketID      string    `json:"marketID,omitempty"`
	AssetID       string    `json:"assetID"`
	Owner         string    `json:"owner"`
	Type          string    `json:"type"`
	Side          string    `json:"side"`
	Price         string    `json:"price"`
	OriginalSize  string    `json:"originalSize"`
	Size          string    `json:"size"`
	SizeFilled    string    `json:"sizeFilled"`
	SizeRemaining string    `json:"sizeRemaining"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	LastUpdated   time.Time `json:"last_updated,omitempty"`
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

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}
