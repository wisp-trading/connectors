package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/wisp-trading/connectors/pkg/websocket/base"
	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	"github.com/wisp-trading/sdk/pkg/types/logging"
)

// globalSubID is a global counter for generating unique subscription IDs across all webSocketService instances
var globalSubID int64

const (
	OrderBookEventType = "book"
)

// webSocketService manages the Polymarket CLOB WebSocket connection using pkg/websocket infrastructure
type webSocketService struct {
	connManager  connection.ConnectionManager
	reconnectMgr connection.ReconnectManager
	baseService  base.BaseService
	logger       logging.ApplicationLogger

	// Subscription tracking
	subscriptionsMu   sync.RWMutex
	subscriptions     map[int]*SubscriptionHandler
	subscriptionID    int64
	subscriptionIndex map[string][]*SubscriptionHandler // channel:asset -> handlers

	// Parsed callbacks
	orderBookCallbacks map[string]func(*OrderBookMessage)
	orderBookMu        sync.RWMutex

	// Error channel
	errorCh chan error

	// State
	ctx    context.Context
	cancel context.CancelFunc
}

// NewWebSocketService creates a new Polymarket WebSocket service using pkg/websocket infrastructure
func NewWebSocketService(
	connManager connection.ConnectionManager,
	reconnectMgr connection.ReconnectManager,
	baseService base.BaseService,
	logger logging.ApplicationLogger,
) PolymarketWebsocket {
	ws := &webSocketService{
		connManager:        connManager,
		reconnectMgr:       reconnectMgr,
		baseService:        baseService,
		logger:             logger,
		subscriptions:      make(map[int]*SubscriptionHandler),
		subscriptionIndex:  make(map[string][]*SubscriptionHandler),
		orderBookCallbacks: make(map[string]func(*OrderBookMessage)),
		errorCh:            make(chan error, 100),
	}

	// Set up connection manager callbacks
	connManager.SetCallbacks(
		ws.onConnect,
		ws.onDisconnect,
		ws.onMessage,
		ws.onError,
	)

	// Set up reconnection manager callbacks
	reconnectMgr.SetCallbacks(
		ws.onReconnectStart,
		ws.onReconnectFail,
		ws.onReconnectSuccess,
	)

	return ws
}

// Connect establishes the WebSocket connection with automatic reconnection
func (ws *webSocketService) Connect(wsURL string) error {
	ws.ctx = context.Background()
	ws.logger.Info(fmt.Sprintf("🔌 Connecting to Polymarket WebSocket: %s", wsURL))

	// Pass the URL to connection manager
	if err := ws.connManager.Connect(ws.ctx, nil, &wsURL); err != nil {
		ws.logger.Error("❌ Failed to connect to WebSocket: %v", err)
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	ws.logger.Info("✅ WebSocket connected successfully")
	_ = ws.reconnectMgr.StartReconnection(ws.ctx) // Ignore error as it may already be running
	return nil
}

// Disconnect closes the WebSocket connection
func (ws *webSocketService) Disconnect() error {
	ws.logger.Info("Closing WebSocket connection")
	return ws.connManager.Disconnect()
}

// IsConnected returns whether the WebSocket is currently connected
func (ws *webSocketService) IsConnected() bool {
	return ws.connManager.GetState() == connection.StateConnected
}

// GetErrorChannel returns the error channel
func (ws *webSocketService) GetErrorChannel() <-chan error {
	return ws.errorCh
}

// Connection lifecycle callbacks
func (ws *webSocketService) onConnect() error {
	ws.logger.Info("Polymarket WebSocket connected")
	return nil
}

func (ws *webSocketService) onDisconnect() error {
	ws.logger.Info("Polymarket WebSocket disconnected")
	return nil
}

func (ws *webSocketService) onMessage(message []byte) error {
	// Use BaseService for rate limiting & validation - pass the callback
	if err := ws.baseService.ProcessMessage(message, func(validatedMsg []byte) error {
		return ws.handleValidatedMessage(validatedMsg)
	}); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}
	return nil
}

func (ws *webSocketService) handleValidatedMessage(message []byte) error {
	// Parse Polymarket WebSocket message
	var polyMsg map[string]interface{}
	if err := json.Unmarshal(message, &polyMsg); err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Check for subscription acknowledgment
	if msgType, ok := polyMsg["type"].(string); ok {
		if msgType == "subscribed" || msgType == "unsubscribed" {
			ws.logger.Debug("Received %s acknowledgment", msgType)
			return nil
		}
	}

	// Route based on event type
	eventType, ok := polyMsg["event_type"].(string)
	if !ok {
		ws.logger.Debug("Message without event_type", "message", string(message))
		return nil
	}

	// Handle different message types
	switch eventType {
	case OrderBookEventType:
		return ws.handleMarketMessage(polyMsg)
	default:
		ws.logger.Debug("Unknown event type", "event_type", eventType)
	}

	return nil
}

func (ws *webSocketService) onError(err error) {
	ws.logger.Error("WebSocket error: %v", err)
	select {
	case ws.errorCh <- err:
	default:
	}
}

// Reconnection callbacks
func (ws *webSocketService) onReconnectStart(attempt int) {
	ws.logger.Info("🔄 Reconnection attempt %d", attempt)
}

func (ws *webSocketService) onReconnectFail(attempt int, err error) {
	ws.logger.Warn("❌ Reconnection attempt %d failed: %v", attempt, err)
}

// subscribe sends a subscription request to Polymarket
func (ws *webSocketService) subscribe(channel string, assets []string) error {
	subMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": channel,
		"assets":  assets,
	}

	data, err := json.Marshal(subMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	return ws.connManager.Send(data)
}

// unsubscribe sends an unsubscription request to Polymarket
func (ws *webSocketService) unsubscribe(channel string, assets []string) error {
	unsubMsg := map[string]interface{}{
		"type":    "unsubscribe",
		"channel": channel,
		"assets":  assets,
	}

	data, err := json.Marshal(unsubMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscription: %w", err)
	}

	return ws.connManager.Send(data)
}

func generateSubscriptionID() int {
	return int(atomic.AddInt64(&globalSubID, 1))
}

// nextSubscriptionID generates a unique subscription ID
func (ws *webSocketService) nextSubscriptionID() int {
	return generateSubscriptionID()
}

// addSubscription registers a subscription handler
func (ws *webSocketService) addSubscription(handler *SubscriptionHandler) {
	ws.subscriptionsMu.Lock()
	defer ws.subscriptionsMu.Unlock()

	ws.subscriptions[handler.ID] = handler

	// Add to index
	key := handler.Channel + ":" + handler.Asset
	ws.subscriptionIndex[key] = append(ws.subscriptionIndex[key], handler)
}

// removeSubscription removes a subscription handler
func (ws *webSocketService) removeSubscription(subscriptionID int) *SubscriptionHandler {
	ws.subscriptionsMu.Lock()
	defer ws.subscriptionsMu.Unlock()

	handler, exists := ws.subscriptions[subscriptionID]
	if !exists {
		return nil
	}

	delete(ws.subscriptions, subscriptionID)

	// Remove from index
	key := handler.Channel + ":" + handler.Asset

	handlers := ws.subscriptionIndex[key]
	for i, h := range handlers {
		if h.ID == subscriptionID {
			ws.subscriptionIndex[key] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return handler
}

func (ws *webSocketService) onReconnectSuccess(attempt int) {
	ws.logger.Info("✅ Reconnection successful after %d attempts, resubscribing to markets", attempt)

	// Resubscribe to all registered market callbacks
	ws.orderBookMu.RLock()
	marketIDs := make([]string, 0, len(ws.orderBookCallbacks))
	for marketID := range ws.orderBookCallbacks {
		marketIDs = append(marketIDs, marketID)
	}
	ws.orderBookMu.RUnlock()

	// Resubscribe to each market
	if len(marketIDs) > 0 {
		if err := ws.subscribe("market", marketIDs); err != nil {
			ws.logger.Error("Failed to resubscribe to markets after reconnection: %v", err)
		} else {
			ws.logger.Info("Resubscribed to %d markets after reconnection", len(marketIDs))
		}
	}
}
