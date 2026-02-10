package websocket

import (
	"encoding/json"
	"fmt"
)

// OrderBookMessage represents a Polymarket orderbook update
type OrderBookMessage struct {
	EventType string       `json:"event_type"`
	AssetID   string       `json:"asset_id"`
	Market    string       `json:"market"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp string       `json:"timestamp"`
	Hash      string       `json:"hash"`
}

// PriceLevel represents a single price level in the orderbook
type PriceLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

func (ws *webSocketService) handleMarketMessage(msg map[string]interface{}) error {
	// Marshal back to JSON for structured parsing
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal orderbook message: %w", err)
	}

	var orderBook OrderBookMessage
	if err := json.Unmarshal(data, &orderBook); err != nil {
		return fmt.Errorf("failed to parse orderbook message: %w", err)
	}

	// Validate required fields
	if orderBook.AssetID == "" || orderBook.Market == "" {
		return fmt.Errorf("missing required orderbook fields")
	}

	// Call registered callbacks using asset_id as key
	ws.orderBookMu.RLock()
	defer ws.orderBookMu.RUnlock()

	if callback, exists := ws.orderBookCallbacks[orderBook.AssetID]; exists {
		callback(&orderBook)
	}

	return nil
}
