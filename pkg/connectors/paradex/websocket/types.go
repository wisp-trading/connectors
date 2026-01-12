package websockets

import (
	"time"
)

// WebSocketService defines the interface for WebSocket-based exchange connectivity
type WebSocketService interface {
	// Connection management
	Connect() error
	Disconnect() error
	IsConnected() bool
	StartWebSocket() error
	StopWebSocket() error
	IsWebSocketConnected() bool

	// Subscription methods
	SubscribeOrderBook(asset string) error
	SubscribeTrades(asset string) error
	SubscribeAccount() error

	UnsubscribeOrderbook(symbol string) error
	UnsubscribeTrades(symbol string) error
	UnsubscribeAccount() error

	// Data channels
	OrderbookUpdates() <-chan OrderbookUpdate
	TradeUpdates() <-chan TradeUpdate
	AccountUpdates() <-chan AccountUpdate
	KlineUpdates() <-chan KlineUpdate
	ErrorChannel() <-chan error

	// Metrics
	GetMetrics() map[string]interface{}
}

type ParadexSubscriptionMessage struct {
	JSONRPC string                    `json:"jsonrpc"`
	ID      int64                     `json:"id"`
	Method  string                    `json:"method"`
	Params  ParadexSubscriptionParams `json:"params"`
}

type ParadexSubscriptionParams struct {
	Channel string `json:"channel"`
	Symbol  string `json:"symbol,omitempty"`
}

// Base message structure
type BaseMessage struct {
	Type    string `json:"type"`
	Channel string `json:"channel,omitempty"`
	Symbol  string `json:"symbol,omitempty"`
}

// Subscription messages
type SubscriptionMessage struct {
	Type   string             `json:"type"`
	ID     int64              `json:"id"`
	Method string             `json:"method"`
	Params SubscriptionParams `json:"params"`
}

type SubscriptionParams struct {
	Channel string `json:"channel"`
	Symbol  string `json:"symbol,omitempty"`
}

type Subscription struct {
	Channel string
	Symbol  string
	Active  bool
}

// Orderbook messages
type OrderbookMessage struct {
	Type   string        `json:"type"`
	Symbol string        `json:"symbol"`
	Data   OrderbookData `json:"data"`
}

type OrderbookData struct {
	Bids   [][]string `json:"bids"`
	Asks   [][]string `json:"asks"`
	SeqNum int64      `json:"seq_num"`
}

type OrderbookUpdate struct {
	Symbol    string       `json:"symbol"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp time.Time    `json:"timestamp"`
	SeqNum    int64        `json:"seq_num"`
}

type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// Trade messages
type TradeMessage struct {
	Type   string      `json:"type"`
	Symbol string      `json:"symbol"`
	Data   []TradeData `json:"data"`
}

type TradeData struct {
	ID        string `json:"id"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
	Side      string `json:"side"`
	Timestamp int64  `json:"timestamp"`
}

type TradeUpdate struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
	TradeID   string    `json:"trade_id"`
}

// Account messages
type AccountMessage struct {
	Type       string      `json:"type"`
	UpdateType string      `json:"update_type"`
	Data       AccountData `json:"data"`
}

type AccountData struct {
	// Balance updates
	Asset     string `json:"asset,omitempty"`
	Balance   string `json:"balance,omitempty"`
	Available string `json:"available,omitempty"`

	// Position updates
	Symbol        string `json:"symbol,omitempty"`
	Size          string `json:"size,omitempty"`
	EntryPrice    string `json:"entry_price,omitempty"`
	UnrealizedPnL string `json:"unrealized_pnl,omitempty"`
	Side          string `json:"side,omitempty"`

	// Order updates
	OrderID string `json:"order_id,omitempty"`
	Status  string `json:"status,omitempty"`
}

type AccountUpdate struct {
	Type          string    `json:"type"`
	Symbol        string    `json:"symbol,omitempty"`
	Balance       float64   `json:"balance,omitempty"`
	Available     float64   `json:"available,omitempty"`
	Size          float64   `json:"size,omitempty"`
	EntryPrice    float64   `json:"entry_price,omitempty"`
	UnrealizedPnL float64   `json:"unrealized_pnl,omitempty"`
	Side          string    `json:"side,omitempty"`
	OrderID       string    `json:"order_id,omitempty"`
	Status        string    `json:"status,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// Message handler interface
type MessageHandler interface {
	Handle(message []byte) error
}
