package websocket

// SubscribeToMarketBook registers a callback for orderbook updates for a specific market
// and sends the subscription request to the Polymarket WebSocket server
func (ws *webSocketService) SubscribeToMarketBook(marketID string, callback func(*OrderBookMessage)) {
	// Register callback first
	ws.orderBookMu.Lock()
	ws.orderBookCallbacks[marketID] = callback
	ws.orderBookMu.Unlock()

	// Send subscription message to WebSocket server
	// Polymarket expects: {"type":"subscribe","channel":"market","assets":["ASSET_ID"]}
	if err := ws.subscribe("market", []string{marketID}); err != nil {
		ws.logger.Error("Failed to send market book subscription for %s: %v", marketID, err)
	} else {
		ws.logger.Debug("Sent market book subscription for %s", marketID)
	}
}

// UnsubscribeFromMarketBook removes the callback for orderbook updates for a specific market
// and sends the unsubscription request to the Polymarket WebSocket server
func (ws *webSocketService) UnsubscribeFromMarketBook(marketID string) {
	// Remove callback first
	ws.orderBookMu.Lock()
	delete(ws.orderBookCallbacks, marketID)
	ws.orderBookMu.Unlock()

	// Send unsubscription message to WebSocket server
	// Polymarket expects: {"type":"unsubscribe","channel":"market","assets":["ASSET_ID"]}
	if err := ws.unsubscribe("market", []string{marketID}); err != nil {
		ws.logger.Error("Failed to send market book unsubscription for %s: %v", marketID, err)
	} else {
		ws.logger.Debug("Sent market book unsubscription for %s", marketID)
	}
}
