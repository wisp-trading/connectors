package websocket

// SubscribeToMarketBook registers a callback for orderbook updates for a specific market
func (ws *webSocketService) SubscribeToMarketBook(marketID string, callback func(*OrderBookMessage)) {
	ws.orderBookMu.Lock()
	defer ws.orderBookMu.Unlock()

	ws.orderBookCallbacks[marketID] = callback
}

// UnsubscribeFromMarketBook removes the callback for orderbook updates for a specific market
func (ws *webSocketService) UnsubscribeFromMarketBook(marketID string) {
	ws.orderBookMu.Lock()
	defer ws.orderBookMu.Unlock()

	delete(ws.orderBookCallbacks, marketID)
}
