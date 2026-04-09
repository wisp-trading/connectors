package pyth

import (
	"encoding/json"
	"time"
)

// PriceUpdate represents a parsed Pyth price feed update
type PriceUpdate struct {
	ID        string
	Price     float64
	Exponent  int32
	Timestamp time.Time
}

// PythMessage is the JSON envelope from Pyth Hermes WebSocket
type PythMessage struct {
	Type   string      `json:"type"`
	ID     string      `json:"id,omitempty"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

// PriceUpdateMsg is the subscription message payload
type PriceUpdateMsg struct {
	Parsed []PriceUpdate `json:"parsed"`
}

// PythSubscribeReq is a subscription request
type PythSubscribeReq struct {
	Type string   `json:"type"`
	IDs  []string `json:"ids"`
}

// PythUnsubscribeReq is an unsubscribe request
type PythUnsubscribeReq struct {
	Type string   `json:"type"`
	IDs  []string `json:"ids"`
}

// Ensure PriceUpdate can be unmarshaled
var _ json.Unmarshaler = (*PriceUpdate)(nil)

// UnmarshalJSON handles custom deserialization from Pyth format
func (p *PriceUpdate) UnmarshalJSON(data []byte) error {
	type Alias PriceUpdate
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	return json.Unmarshal(data, &aux)
}
