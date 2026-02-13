package websocket

import (
	"encoding/json"
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
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

func (ws *webSocketService) handleOrderbookMessage(msg map[string]interface{}) error {
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

	if callback, exists := ws.orderBookCallbacks[orderBook.Market]; exists {
		callback(&orderBook)
	}

	return nil
}

// SubscribeToMarket registers a callback for orderbook updates for a specific market
// and sends the subscription request to the Polymarket WebSocket server
func (ws *webSocketService) SubscribeToMarket(
	market prediction.Market,
	orderbookCallback func(*OrderBookMessage),
	priceChangeCallback func(*PriceChanges),
) {
	// Register orderbookCallback first
	ws.orderBookMu.Lock()
	ws.orderBookCallbacks[market.MarketId] = orderbookCallback
	ws.orderBookMu.Unlock()

	ws.priceChangeMu.Lock()
	ws.priceChangeCallbacks[market.MarketId] = priceChangeCallback
	ws.priceChangeMu.Unlock()

	assetIds := make([]string, len(market.Outcomes))
	for i, outcome := range market.Outcomes {
		assetIds[i] = outcome.OutcomeId
	}

	// Send subscription message to WebSocket server
	// Polymarket expects: {"type":"subscribe","channel":"market","assets":["ASSET_ID"]}
	if err := ws.subscribe("market", assetIds); err != nil {
		ws.logger.Error("Failed to send market book subscription for %s: %v", market.MarketId, err)
	} else {
		ws.logger.Debug("Sent market book subscription for %s", market.MarketId)
	}
}

// UnsubscribeFromMarket removes the callback for orderbook updates for a specific market
// and sends the unsubscription request to the Polymarket WebSocket server
// todo need to check if this actually cancels the subscription on Polymarket's end
func (ws *webSocketService) UnsubscribeFromMarket(market prediction.Market) {
	// Remove callback first
	ws.orderBookMu.Lock()
	delete(ws.orderBookCallbacks, market.MarketId)
	ws.orderBookMu.Unlock()

	ws.priceChangeMu.Lock()
	delete(ws.priceChangeCallbacks, market.MarketId)
	ws.priceChangeMu.Unlock()

	assetIds := make([]string, len(market.Outcomes))
	for i, outcome := range market.Outcomes {
		assetIds[i] = outcome.OutcomeId
	}

	// Send unsubscription message to WebSocket server
	// Polymarket expects: {"type":"unsubscribe","channel":"market","assets":["ASSET_ID"]}
	if err := ws.unsubscribe("market", assetIds); err != nil {
		ws.logger.Error("Failed to send market book unsubscription for %s: %v", market.Slug, err)
	} else {
		ws.logger.Debug("Sent market book unsubscription for %s", market.Slug)
	}
}
