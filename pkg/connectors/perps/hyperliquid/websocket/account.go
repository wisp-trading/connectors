package websocket

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

// SubscribeToPositions subscribes to position updates
func (ws *WebSocketService) SubscribeToPositions(user string, callback func(*PositionMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	// For user-specific subscriptions on Hyperliquid, subscribe to webData2 channel
	subID := generateSubscriptionID()

	rawSubID, err := ws.subscribeToChannel("webData2", user, "", func(msg hyperliquid.WSMessage) {
		parsed, err := ws.parser.ParsePosition(msg)
		if err != nil {
			ws.logger.Warn("Failed to parse position: %v", err)
			return
		}
		if parsed != nil {
			callback(parsed)
		}
	})

	if err != nil {
		return 0, err
	}

	ws.subscriptionsMu.Lock()
	ws.subscriptions[rawSubID].ID = subID
	ws.subscriptionsMu.Unlock()

	ws.logger.Info("✅ Subscribed to positions for %s (ID: %d)", user, subID)
	return subID, nil
}

// SubscribeToAccountBalance subscribes to account balance updates
func (ws *WebSocketService) SubscribeToAccountBalance(user string, callback func(*AccountBalanceMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	// For user-specific subscriptions on Hyperliquid, subscribe to webData2 channel
	subID := generateSubscriptionID()

	rawSubID, err := ws.subscribeToChannel("webData2", user, "", func(msg hyperliquid.WSMessage) {
		parsed, err := ws.parser.ParseAccountBalance(msg)
		if err != nil {
			ws.logger.Warn("Failed to parse account balance: %v", err)
			return
		}
		if parsed != nil {
			callback(parsed)
		}
	})

	if err != nil {
		return 0, err
	}

	ws.subscriptionsMu.Lock()
	ws.subscriptions[rawSubID].ID = subID
	ws.subscriptionsMu.Unlock()

	ws.logger.Info("✅ Subscribed to account balance for %s (ID: %d)", user, subID)
	return subID, nil
}
