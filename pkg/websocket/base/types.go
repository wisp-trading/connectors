package base

import (
	"time"

	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// BaseMessage represents the common structure of all WebSocket messages
type BaseMessage struct {
	Type      string    `json:"type"`
	Channel   string    `json:"channel,omitempty"`
	Symbol    string    `json:"symbol,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	ID        int64     `json:"id,omitempty"`
}

// SubscriptionMessage represents subscription/unsubscription requests
type SubscriptionMessage struct {
	Type   string                 `json:"type"`
	ID     int64                  `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// ErrorMessage represents error responses from the WebSocket
type ErrorMessage struct {
	Type    string `json:"type"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
	ID      int64  `json:"id,omitempty"`
}

// PingPongMessage represents ping/pong messages for connection health
type PingPongMessage struct {
	Type string `json:"type"`
	ID   int64  `json:"id,omitempty"`
}

// OrderbookUpdate represents a generic orderbook update
type OrderbookUpdate struct {
	Symbol    string       `json:"symbol"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp time.Time    `json:"timestamp"`
	SeqNum    int64        `json:"seq_num,omitempty"`
}

// PriceLevel represents a price level in an orderbook
type PriceLevel struct {
	Price    numerical.Decimal `json:"price"`
	Quantity numerical.Decimal `json:"quantity"`
}

// TradeUpdate represents a trade execution update
type TradeUpdate struct {
	Symbol    string            `json:"symbol"`
	Price     numerical.Decimal `json:"price"`
	Quantity  numerical.Decimal `json:"quantity"`
	Side      string            `json:"side"`
	Timestamp time.Time         `json:"timestamp"`
	TradeID   string            `json:"trade_id"`
}

// TickerUpdate represents a price ticker update
type TickerUpdate struct {
	Symbol    string            `json:"symbol"`
	Price     numerical.Decimal `json:"price"`
	BidPrice  numerical.Decimal `json:"bid_price,omitempty"`
	AskPrice  numerical.Decimal `json:"ask_price,omitempty"`
	Volume24h numerical.Decimal `json:"volume_24h,omitempty"`
	Change24h numerical.Decimal `json:"change_24h,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// AccountUpdate represents account-related updates
type AccountUpdate struct {
	Type          string            `json:"type"`
	Symbol        string            `json:"symbol,omitempty"`
	Balance       numerical.Decimal `json:"balance,omitempty"`
	Available     numerical.Decimal `json:"available,omitempty"`
	Size          numerical.Decimal `json:"size,omitempty"`
	EntryPrice    numerical.Decimal `json:"entry_price,omitempty"`
	UnrealizedPnL numerical.Decimal `json:"unrealized_pnl,omitempty"`
	Side          string            `json:"side,omitempty"`
	OrderID       string            `json:"order_id,omitempty"`
	Status        string            `json:"status,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
}

// OrderUpdate represents order status updates
type OrderUpdate struct {
	OrderID      string            `json:"order_id"`
	Symbol       string            `json:"symbol"`
	Side         string            `json:"side"`
	Type         string            `json:"type"`
	Status       string            `json:"status"`
	Quantity     numerical.Decimal `json:"quantity"`
	Price        numerical.Decimal `json:"price,omitempty"`
	FilledQty    numerical.Decimal `json:"filled_quantity"`
	RemainingQty numerical.Decimal `json:"remaining_quantity"`
	AvgPrice     numerical.Decimal `json:"average_price,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// PositionUpdate represents position updates
type PositionUpdate struct {
	Symbol        string            `json:"symbol"`
	Side          string            `json:"side"`
	Size          numerical.Decimal `json:"size"`
	EntryPrice    numerical.Decimal `json:"entry_price"`
	MarkPrice     numerical.Decimal `json:"mark_price,omitempty"`
	UnrealizedPnL numerical.Decimal `json:"unrealized_pnl"`
	RealizedPnL   numerical.Decimal `json:"realized_pnl,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
}

// FundingRateUpdate represents funding rate updates
type FundingRateUpdate struct {
	Symbol      string            `json:"symbol"`
	FundingRate numerical.Decimal `json:"funding_rate"`
	NextFunding time.Time         `json:"next_funding"`
	Timestamp   time.Time         `json:"timestamp"`
}

// UpdateType represents the type of update received
type UpdateType string

const (
	UpdateTypeOrderbook    UpdateType = "orderbook"
	UpdateTypeTrade        UpdateType = "trade"
	UpdateTypeTicker       UpdateType = "ticker"
	UpdateTypeAccount      UpdateType = "account"
	UpdateTypeOrder        UpdateType = "order"
	UpdateTypePosition     UpdateType = "position"
	UpdateTypeFundingRate  UpdateType = "funding_rate"
	UpdateTypeError        UpdateType = "error"
	UpdateTypePing         UpdateType = "ping"
	UpdateTypePong         UpdateType = "pong"
	UpdateTypeSubscription UpdateType = "subscription"
)

// ChannelType represents different WebSocket channels
type ChannelType string

const (
	ChannelOrderbook    ChannelType = "orderbook"
	ChannelTrades       ChannelType = "trades"
	ChannelTicker       ChannelType = "ticker"
	ChannelAccount      ChannelType = "account"
	ChannelOrders       ChannelType = "orders"
	ChannelPositions    ChannelType = "positions"
	ChannelFundingRates ChannelType = "funding_rates"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusPending      SubscriptionStatus = "pending"
	SubscriptionStatusSubscribed   SubscriptionStatus = "subscribed"
	SubscriptionStatusUnsubscribed SubscriptionStatus = "unsubscribed"
	SubscriptionStatusError        SubscriptionStatus = "error"
)

// Subscription represents an active WebSocket subscription
type Subscription struct {
	Channel   string             `json:"channel"`
	Symbol    string             `json:"symbol,omitempty"`
	Status    SubscriptionStatus `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// ConnectionStats represents WebSocket connection statistics
type ConnectionStats struct {
	ConnectedAt       time.Time `json:"connected_at"`
	LastMessageAt     time.Time `json:"last_message_at"`
	MessagesReceived  int64     `json:"messages_received"`
	MessagesProcessed int64     `json:"messages_processed"`
	MessagesDropped   int64     `json:"messages_dropped"`
	Subscriptions     int       `json:"subscriptions"`
	ReconnectCount    int       `json:"reconnect_count"`
	State             string    `json:"state"`
}
