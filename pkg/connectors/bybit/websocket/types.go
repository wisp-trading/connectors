package websocket

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// RealTimeService defines the WebSocket interface for Bybit Perpetual real-time data
// Following Gate.io pattern: callback-based subscriptions, NOT channels
type RealTimeService interface {
	Connect(wsURL string) error
	Disconnect() error
	IsConnected() bool
	GetErrorChannel() <-chan error

	// Orderbook subscriptions
	SubscribeToOrderBook(symbol string, callback func(*OrderBookMessage)) (int, error)
	UnsubscribeFromOrderBook(symbol string, subscriptionID int) error

	// Trade subscriptions
	SubscribeToTrades(symbol string, callback func([]TradeMessage)) (int, error)
	UnsubscribeFromTrades(symbol string, subscriptionID int) error

	// Position subscriptions (private)
	SubscribeToPositions(callback func(*PositionMessage)) (int, error)
	UnsubscribeFromPositions(subscriptionID int) error

	// Account balance subscriptions (private)
	SubscribeToAccountBalance(callback func(*AccountBalanceMessage)) (int, error)
	UnsubscribeFromAccountBalance(subscriptionID int) error

	// Kline subscriptions
	SubscribeToKlines(symbol, interval string, callback func(*KlineMessage)) (int, error)
	UnsubscribeFromKlines(symbol, interval string, subscriptionID int) error
}

// OrderBookMessage represents order book updates from Bybit
type OrderBookMessage struct {
	Symbol    string
	Bids      [][]string // [price, quantity]
	Asks      [][]string // [price, quantity]
	Timestamp int64
	UpdateID  int64
}

// TradeMessage represents trade updates from Bybit
type TradeMessage struct {
	ID        string
	Symbol    string
	Price     string
	Quantity  string
	Side      connector.OrderSide
	Timestamp int64
}

// PositionMessage represents position updates from Bybit
type PositionMessage struct {
	Symbol           string
	Side             string
	Size             string
	EntryPrice       string
	MarkPrice        string
	LiquidationPrice string
	UnrealizedPnL    string
	RealizedPnL      string
	Leverage         string
	Timestamp        int64
}

// AccountBalanceMessage represents account balance updates from Bybit
type AccountBalanceMessage struct {
	TotalEquity           string
	TotalAvailableBalance string
	TotalMarginBalance    string
	TotalPerpUPL          string
	Timestamp             int64
}

// KlineMessage represents kline/candlestick updates from Bybit
type KlineMessage struct {
	Symbol    string
	Interval  string
	StartTime int64
	Open      string
	High      string
	Low       string
	Close     string
	Volume    string
	Timestamp int64
}

// SubscriptionHandler manages a subscription callback
type SubscriptionHandler struct {
	ID       int
	Channel  string
	Symbol   string
	Interval string
	Callback interface{}
}

// BybitWSMessage represents the raw Bybit WebSocket message structure
type BybitWSMessage struct {
	Topic  string      `json:"topic"`
	Type   string      `json:"type"`
	Data   interface{} `json:"data"`
	Ts     int64       `json:"ts"`
	Op     string      `json:"op"`
	Args   []string    `json:"args"`
	RetMsg string      `json:"ret_msg"`
	ConnID string      `json:"conn_id"`
}

// OrderBookData represents the data field in orderbook messages
type OrderBookData struct {
	Symbol   string     `json:"s"`
	Bids     [][]string `json:"b"`
	Asks     [][]string `json:"a"`
	UpdateID int64      `json:"u"`
	Seq      int64      `json:"seq"`
}

// TradeData represents the data field in trade messages
type TradeData struct {
	Timestamp int64  `json:"T"`
	Symbol    string `json:"s"`
	Side      string `json:"S"`
	Size      string `json:"v"`
	Price     string `json:"p"`
	TradeID   string `json:"i"`
}

// PositionData represents the data field in position messages
type PositionData struct {
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	Size           string `json:"size"`
	PositionValue  string `json:"positionValue"`
	EntryPrice     string `json:"entryPrice"`
	MarkPrice      string `json:"markPrice"`
	LiqPrice       string `json:"liqPrice"`
	UnrealisedPnl  string `json:"unrealisedPnl"`
	CumRealisedPnl string `json:"cumRealisedPnl"`
	Leverage       string `json:"leverage"`
	PositionStatus string `json:"positionStatus"`
	UpdatedTime    string `json:"updatedTime"`
}

// WalletData represents the data field in wallet/balance messages
type WalletData struct {
	AccountType           string     `json:"accountType"`
	TotalEquity           string     `json:"totalEquity"`
	TotalWalletBalance    string     `json:"totalWalletBalance"`
	TotalMarginBalance    string     `json:"totalMarginBalance"`
	TotalAvailableBalance string     `json:"totalAvailableBalance"`
	TotalPerpUPL          string     `json:"totalPerpUPL"`
	Coin                  []CoinData `json:"coin"`
}

// CoinData represents individual coin balance in wallet
type CoinData struct {
	Coin                string `json:"coin"`
	Equity              string `json:"equity"`
	WalletBalance       string `json:"walletBalance"`
	AvailableToWithdraw string `json:"availableToWithdraw"`
	Bonus               string `json:"bonus"`
	LockedBalance       string `json:"locked"`
}

// KlineData represents the data field in kline messages
type KlineData struct {
	Start     int64  `json:"start"`
	End       int64  `json:"end"`
	Interval  string `json:"interval"`
	Open      string `json:"open"`
	Close     string `json:"close"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Volume    string `json:"volume"`
	Turnover  string `json:"turnover"`
	Confirm   bool   `json:"confirm"`
	Timestamp int64  `json:"timestamp"`
}
