package websocket

import (
	"time"

	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// RealTimeService defines the WebSocket interface for real-time market data
type RealTimeService interface {
	Connect(websocketUrl *string) error
	Disconnect() error
	IsConnected() bool
	GetErrorChannel() <-chan error

	// Orderbook subscriptions
	SubscribeToOrderBook(coin string, callback func(*OrderBookMessage)) (int, error)
	UnsubscribeFromOrderBook(coin string, subscriptionID int) error

	// Trade subscriptions
	SubscribeToTrades(coin string, callback func([]TradeMessage)) (int, error)
	UnsubscribeFromTrades(coin string, subscriptionID int) error

	// Position subscriptions
	SubscribeToPositions(user string, callback func(*PositionMessage)) (int, error)

	// Account balance subscriptions
	SubscribeToAccountBalance(user string, callback func(*AccountBalanceMessage)) (int, error)

	// Kline subscriptions
	SubscribeToKlines(coin, interval string, callback func(*KlineMessage)) (int, error)
	UnsubscribeFromKlines(coin, interval string, subscriptionID int) error

	// Funding rate subscriptions (activeAssetCtx)
	SubscribeToFundingRates(coin string, callback func(*FundingRateMessage)) (int, error)
	UnsubscribeFromFundingRates(coin string, subscriptionID int) error
}

// OrderBookMessage represents a parsed L2 order book update from WebSocket
type OrderBookMessage struct {
	Coin      string
	Timestamp time.Time
	Bids      []PriceLevel
	Asks      []PriceLevel
}

// PriceLevel represents a single price level in the order book
type PriceLevel struct {
	Price    numerical.Decimal
	Quantity numerical.Decimal
}

// TradeMessage represents a parsed trade update from WebSocket
type TradeMessage struct {
	Coin      string
	Price     numerical.Decimal
	Quantity  numerical.Decimal
	Side      string
	Timestamp time.Time
	Hash      string
	TradeID   int64
}

// PositionMessage represents a parsed position update from WebSocket
type PositionMessage struct {
	Coin           string
	Size           numerical.Decimal
	EntryPrice     numerical.Decimal
	MarkPrice      numerical.Decimal
	LiquidationPx  numerical.Decimal
	UnrealizedPnl  numerical.Decimal
	Leverage       int
	MarginUsed     numerical.Decimal
	PositionValue  numerical.Decimal
	ReturnOnEquity numerical.Decimal
	Timestamp      time.Time
}

// AccountBalanceMessage represents a parsed account balance update from WebSocket
type AccountBalanceMessage struct {
	TotalValue        numerical.Decimal
	AvailableBalance  numerical.Decimal
	Withdrawable      numerical.Decimal
	TotalMarginUsed   numerical.Decimal
	TotalNtlPos       numerical.Decimal
	TotalRawUsd       numerical.Decimal
	TotalAccountValue numerical.Decimal
	Timestamp         time.Time
}

// KlineMessage represents a parsed kline/candlestick update from WebSocket
type KlineMessage struct {
	Coin      string
	Interval  string
	OpenTime  time.Time
	CloseTime time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timestamp time.Time
}

// FundingRateMessage represents a parsed funding rate update from activeAssetCtx WebSocket
type FundingRateMessage struct {
	Coin            string
	FundingRate     numerical.Decimal
	MarkPrice       numerical.Decimal
	OpenInterest    numerical.Decimal
	PreviousDayPx   numerical.Decimal
	DayNtlVlm       numerical.Decimal
	Premium         numerical.Decimal
	OraclePrice     numerical.Decimal
	NextFundingTime time.Time
	Timestamp       time.Time
}
