package websocket

import (
	"encoding/json"
	"fmt"
)

type PriceChanges struct {
	Market      string        `json:"market"`
	PriceChange []PriceChange `json:"price_changes"`
	Timestamp   string        `json:"timestamp"`
	EventType   string        `json:"event_type"`
}

type PriceChange struct {
	AssetId string `json:"asset_id"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	Side    string `json:"side"`
	Hash    string `json:"hash"`
	BestBid string `json:"best_bid"`
	BestAsk string `json:"best_ask"`
}

func (ws *webSocketService) handlePriceChangeMessage(msg map[string]interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal price change message: %w", err)
	}

	var priceChange PriceChanges
	if err := json.Unmarshal(data, &priceChange); err != nil {
		return fmt.Errorf("failed to parse price change message: %w", err)
	}

	// Validate required fields
	if priceChange.Market == "" {
		return fmt.Errorf("missing required price change fields")
	}

	// Call registered callbacks using market as key
	ws.priceChangeMu.RLock()
	defer ws.priceChangeMu.RUnlock()

	if callback, exists := ws.priceChangeCallbacks[priceChange.Market]; exists {
		callback(&priceChange)
	}

	return nil
}
