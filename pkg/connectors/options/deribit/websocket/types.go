package websocket

import "encoding/json"

// DeribitWSMessage is the generic envelope for all Deribit WebSocket messages.
// Deribit uses JSON-RPC 2.0 over WebSocket — the same wire format as the REST API.
//
// Incoming message patterns:
//
//	Subscription update:  { "method": "subscription", "params": { "channel": "...", "data": {...} } }
//	Heartbeat:            { "method": "heartbeat",     "params": { "type": "test_request" } }
//	RPC response:         { "id": 1, "result": {...} }
//	RPC error:            { "id": 1, "error": { "code": ..., "message": "..." } }
type DeribitWSMessage struct {
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Params  *MessageParams  `json:"params,omitempty"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// MessageParams is the params field for subscription and heartbeat messages.
type MessageParams struct {
	Channel string          `json:"channel,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Type    string          `json:"type,omitempty"` // heartbeat: "test_request"
}

// RPCError is the JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TickerData is the payload for ticker.{instrument}.{interval} channels.
// This is the primary data source for options mark price, IV, and Greeks.
type TickerData struct {
	InstrumentName  string   `json:"instrument_name"`
	Timestamp       int64    `json:"timestamp"`
	UnderlyingPrice float64  `json:"underlying_price"`
	MarkPrice       float64  `json:"mark_price"`
	MarkIV          float64  `json:"mark_iv"`
	BestBidPrice    *float64 `json:"best_bid_price"`
	BestAskPrice    *float64 `json:"best_ask_price"`
	OpenInterest    float64  `json:"open_interest"`
	Stats           struct {
		Volume float64 `json:"volume_usd"`
	} `json:"stats"`
	Greeks struct {
		Delta float64 `json:"delta"`
		Gamma float64 `json:"gamma"`
		Theta float64 `json:"theta"`
		Vega  float64 `json:"vega"`
		Rho   float64 `json:"rho"`
	} `json:"greeks"`
}

// TradeData is the payload for trades.{instrument}.{interval} channels.
type TradeData struct {
	TradeID        string  `json:"trade_id"`
	InstrumentName string  `json:"instrument_name"`
	Price          float64 `json:"price"`
	Amount         float64 `json:"amount"`
	Direction      string  `json:"direction"` // "buy" or "sell"
	Timestamp      int64   `json:"timestamp"`
	IV             float64 `json:"iv"`
	MarkPrice      float64 `json:"mark_price"`
}

// OrderBookAction represents what Deribit wants us to do with an order book level.
type OrderBookAction string

const (
	OrderBookActionNew    OrderBookAction = "new"
	OrderBookActionChange OrderBookAction = "change"
	OrderBookActionDelete OrderBookAction = "delete"
)

// OrderBookEntry is a single price level in a Deribit order book update.
// Deribit encodes each entry as a JSON array: ["action", price, amount].
type OrderBookEntry struct {
	Action OrderBookAction
	Price  float64
	Amount float64
}

// UnmarshalJSON parses the Deribit array format ["action", price, amount].
func (e *OrderBookEntry) UnmarshalJSON(data []byte) error {
	var raw [3]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[0], &e.Action); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[1], &e.Price); err != nil {
		return err
	}
	return json.Unmarshal(raw[2], &e.Amount)
}

// OrderBookData is the payload for book.{instrument}.{group}.{depth}.{interval} channels.
// Deribit sends a "snapshot" first (full state), then "change" updates (diffs).
type OrderBookData struct {
	InstrumentName string           `json:"instrument_name"`
	Timestamp      int64            `json:"timestamp"`
	Type           string           `json:"type"` // "snapshot" or "change"
	Bids           []OrderBookEntry `json:"bids"`
	Asks           []OrderBookEntry `json:"asks"`
}

// SubscriptionHandler holds an instrument-scoped callback for ticker updates.
type SubscriptionHandler struct {
	Instrument string
	Callback   func(*TickerData)
}
