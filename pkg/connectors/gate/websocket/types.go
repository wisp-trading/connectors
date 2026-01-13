package websocket

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
)

// RealTimeService defines the WebSocket interface for Gate.io Spot real-time market data
type RealTimeService interface {
	Connect() error
	Disconnect() error
	IsConnected() bool
	GetErrorChannel() <-chan error

	// Orderbook subscriptions
	SubscribeToOrderBook(symbol string, callback func(*OrderBookMessage)) (int, error)
	UnsubscribeFromOrderBook(symbol string, subscriptionID int) error

	// Trade subscriptions
	SubscribeToTrades(symbol string, callback func([]TradeMessage)) (int, error)
	UnsubscribeFromTrades(symbol string, subscriptionID int) error

	// Kline subscriptions
	SubscribeToKlines(symbol, interval string, callback func(*KlineMessage)) (int, error)
	UnsubscribeFromKlines(symbol, interval string, subscriptionID int) error

	// Account subscriptions (requires authentication)
	SubscribeToAccountBalance(callback func(*AccountBalanceMessage)) (int, error)
	SubscribeToOrders(callback func(*OrderMessage)) (int, error)
}

// OrderBookMessage represents order book updates
type OrderBookMessage struct {
	Symbol    string
	Bids      [][]string // [price, quantity]
	Asks      [][]string // [price, quantity]
	Timestamp int64
}

// TradeMessage represents trade updates
type TradeMessage struct {
	ID           int64
	Symbol       string
	Price        string
	Amount       string
	Side         connector.OrderSide
	Timestamp    int64
	CreateTimeMs int64
}

// KlineMessage represents kline/candlestick updates
type KlineMessage struct {
	Symbol      string
	Interval    string
	OpenTime    int64
	CloseTime   int64
	Open        string
	High        string
	Low         string
	Close       string
	Volume      string
	ClosePrice  string
	QuoteVolume string
}

// AccountBalanceMessage represents account balance updates
type AccountBalanceMessage struct {
	Timestamp int64
	Balances  map[string]Balance
}

type Balance struct {
	Currency  string
	Available string
	Locked    string
	Total     string
}

// OrderMessage represents order updates
type OrderMessage struct {
	ID           string
	Symbol       string
	Side         string
	Type         string
	Status       string
	Price        string
	Amount       string
	FilledAmount string
	CreateTime   int64
	UpdateTime   int64
}

// SubscriptionHandler manages a subscription callback
type SubscriptionHandler struct {
	ID       int
	Channel  string
	Symbol   string
	Interval string
	Callback interface{}
}
