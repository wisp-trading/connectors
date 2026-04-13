package websocket

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

// SubscribeToTrades subscribes to trade updates for a coin
func (ws *WebSocketService) SubscribeToTrades(coin string, callback func([]TradeMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	subID := generateSubscriptionID()

	ws.tradesMu.Lock()
	ws.tradesCallbacks[subID] = callback
	ws.tradesMu.Unlock()

	rawSubID, err := ws.subscribeToChannel("trades", coin, "", func(msg hyperliquid.WSMessage) {
		parsed, err := ws.parseTrades(msg)
		if err != nil {
			ws.logger.Warn("Failed to parse trades: %v", err)
			return
		}

		ws.tradesMu.RLock()
		cb, exists := ws.tradesCallbacks[subID]
		ws.tradesMu.RUnlock()

		if exists && cb != nil {
			cb(parsed)
		}
	})

	if err != nil {
		ws.tradesMu.Lock()
		delete(ws.tradesCallbacks, subID)
		ws.tradesMu.Unlock()
		return 0, err
	}

	ws.subscriptionsMu.Lock()
	ws.subscriptions[rawSubID].ID = subID
	ws.subscriptionsMu.Unlock()

	ws.logger.Info("✅ Subscribed to trades for %s (ID: %d)", coin, subID)
	return subID, nil
}

// UnsubscribeFromTrades unsubscribes from trade updates
func (ws *WebSocketService) UnsubscribeFromTrades(coin string, subscriptionID int) error {
	ws.tradesMu.Lock()
	delete(ws.tradesCallbacks, subscriptionID)
	ws.tradesMu.Unlock()

	ws.subscriptionsMu.Lock()
	for rawID, sub := range ws.subscriptions {
		if sub.ID == subscriptionID && sub.Channel == "trades" && sub.Coin == coin {
			delete(ws.subscriptions, rawID)
			ws.subscriptionsMu.Unlock()
			ws.logger.Info("Unsubscribed from trades for %s (ID: %d)", coin, subscriptionID)
			return nil
		}
	}
	ws.subscriptionsMu.Unlock()

	return fmt.Errorf("subscription not found")
}
