package websocket

import (
	"fmt"
)

// SubscribeToOrderBook subscribes to order book updates for a symbol
func (ws *WebSocketService) SubscribeToOrderBook(symbol string, callback func(*OrderBookMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	subID := generateSubscriptionID()

	// Store the parsed callback
	ws.orderBookMu.Lock()
	ws.orderBookCallbacks[subID] = callback
	ws.orderBookMu.Unlock()

	handler := &SubscriptionHandler{
		ID:      subID,
		Channel: "spot.order_book",
		Symbol:  symbol,
	}

	ws.addSubscription(handler)

	// Subscribe to Gate.io WebSocket
	if err := ws.subscribe("spot.order_book", []string{symbol, "100ms", "20"}); err != nil {
		ws.orderBookMu.Lock()
		delete(ws.orderBookCallbacks, subID)
		ws.orderBookMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to order book for %s (ID: %d)", symbol, subID)
	return subID, nil
}

// UnsubscribeFromOrderBook unsubscribes from order book updates
func (ws *WebSocketService) UnsubscribeFromOrderBook(symbol string, subscriptionID int) error {
	ws.orderBookMu.Lock()
	delete(ws.orderBookCallbacks, subscriptionID)
	ws.orderBookMu.Unlock()

	// Find and remove the subscription
	ws.subscriptionsMu.Lock()
	for _, sub := range ws.subscriptions {
		if sub.ID == subscriptionID && sub.Channel == "spot.order_book" && sub.Symbol == symbol {
			delete(ws.subscriptions, subscriptionID)
			ws.subscriptionsMu.Unlock()

			// Unsubscribe from Gate.io WebSocket
			if err := ws.unsubscribe("spot.order_book", []string{symbol, "100ms", "20"}); err != nil {
				return fmt.Errorf("failed to unsubscribe: %w", err)
			}
			return nil
		}
	}
	ws.subscriptionsMu.Unlock()

	return fmt.Errorf("subscription not found")
}

// SubscribeToTrades subscribes to trade updates for a symbol
func (ws *WebSocketService) SubscribeToTrades(symbol string, callback func([]TradeMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	subID := generateSubscriptionID()

	ws.tradesMu.Lock()
	ws.tradesCallbacks[subID] = callback
	ws.tradesMu.Unlock()

	handler := &SubscriptionHandler{
		ID:      subID,
		Channel: "spot.trades",
		Symbol:  symbol,
	}

	ws.addSubscription(handler)

	if err := ws.subscribe("spot.trades", []string{symbol}); err != nil {
		ws.tradesMu.Lock()
		delete(ws.tradesCallbacks, subID)
		ws.tradesMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to trades for %s (ID: %d)", symbol, subID)
	return subID, nil
}

// UnsubscribeFromTrades unsubscribes from trade updates
func (ws *WebSocketService) UnsubscribeFromTrades(symbol string, subscriptionID int) error {
	ws.tradesMu.Lock()
	delete(ws.tradesCallbacks, subscriptionID)
	ws.tradesMu.Unlock()

	ws.subscriptionsMu.Lock()
	for _, sub := range ws.subscriptions {
		if sub.ID == subscriptionID && sub.Channel == "spot.trades" && sub.Symbol == symbol {
			delete(ws.subscriptions, subscriptionID)
			ws.subscriptionsMu.Unlock()

			if err := ws.unsubscribe("spot.trades", []string{symbol}); err != nil {
				return fmt.Errorf("failed to unsubscribe: %w", err)
			}
			return nil
		}
	}
	ws.subscriptionsMu.Unlock()

	return fmt.Errorf("subscription not found")
}

// SubscribeToKlines subscribes to kline/candlestick updates
func (ws *WebSocketService) SubscribeToKlines(symbol, interval string, callback func(*KlineMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	subID := generateSubscriptionID()

	ws.klinesMu.Lock()
	ws.klinesCallbacks[subID] = callback
	ws.klinesMu.Unlock()

	handler := &SubscriptionHandler{
		ID:       subID,
		Channel:  "spot.candlesticks",
		Symbol:   symbol,
		Interval: interval,
	}

	ws.addSubscription(handler)

	if err := ws.subscribe("spot.candlesticks", []string{interval, symbol}); err != nil {
		ws.klinesMu.Lock()
		delete(ws.klinesCallbacks, subID)
		ws.klinesMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to klines for %s %s (ID: %d)", symbol, interval, subID)
	return subID, nil
}

// UnsubscribeFromKlines unsubscribes from kline updates
func (ws *WebSocketService) UnsubscribeFromKlines(symbol, interval string, subscriptionID int) error {
	ws.klinesMu.Lock()
	delete(ws.klinesCallbacks, subscriptionID)
	ws.klinesMu.Unlock()

	ws.subscriptionsMu.Lock()
	for _, sub := range ws.subscriptions {
		if sub.ID == subscriptionID && sub.Channel == "spot.candlesticks" && sub.Symbol == symbol {
			delete(ws.subscriptions, subscriptionID)
			ws.subscriptionsMu.Unlock()

			if err := ws.unsubscribe("spot.candlesticks", []string{interval, symbol}); err != nil {
				return fmt.Errorf("failed to unsubscribe: %w", err)
			}
			return nil
		}
	}
	ws.subscriptionsMu.Unlock()

	return fmt.Errorf("subscription not found")
}

// SubscribeToAccountBalance subscribes to account balance updates (requires authentication)
func (ws *WebSocketService) SubscribeToAccountBalance(callback func(*AccountBalanceMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	subID := generateSubscriptionID()

	ws.balanceMu.Lock()
	ws.balanceCallbacks[subID] = callback
	ws.balanceMu.Unlock()

	handler := &SubscriptionHandler{
		ID:      subID,
		Channel: "spot.balances",
	}

	ws.addSubscription(handler)

	if err := ws.subscribe("spot.balances", []string{}); err != nil {
		ws.balanceMu.Lock()
		delete(ws.balanceCallbacks, subID)
		ws.balanceMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to account balance (ID: %d)", subID)
	return subID, nil
}

// SubscribeToOrders subscribes to order updates (requires authentication)
func (ws *WebSocketService) SubscribeToOrders(callback func(*OrderMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	subID := generateSubscriptionID()

	ws.orderMu.Lock()
	ws.orderCallbacks[subID] = callback
	ws.orderMu.Unlock()

	handler := &SubscriptionHandler{
		ID:      subID,
		Channel: "spot.orders",
	}

	ws.addSubscription(handler)

	if err := ws.subscribe("spot.orders", []string{"!all"}); err != nil {
		ws.orderMu.Lock()
		delete(ws.orderCallbacks, subID)
		ws.orderMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to orders (ID: %d)", subID)
	return subID, nil
}
