package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wisp-trading/connectors/pkg/websocket/base"
	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	"github.com/wisp-trading/sdk/pkg/types/logging"
)

// WebSocketService manages the Gate.io WebSocket connection using pkg/websocket infrastructure
type WebSocketService struct {
	connManager  connection.ConnectionManager
	reconnectMgr connection.ReconnectManager
	baseService  base.BaseService
	logger       logging.ApplicationLogger

	// Subscription tracking
	subscriptionsMu   sync.RWMutex
	subscriptions     map[int]*SubscriptionHandler
	subscriptionID    int64
	subscriptionIndex map[string][]*SubscriptionHandler // channel:symbol -> handlers

	// Parsed callbacks
	orderBookCallbacks map[int]func(*OrderBookMessage)
	orderBookMu        sync.RWMutex
	tradesCallbacks    map[int]func([]TradeMessage)
	tradesMu           sync.RWMutex
	klinesCallbacks    map[int]func(*KlineMessage)
	klinesMu           sync.RWMutex
	balanceCallbacks   map[int]func(*AccountBalanceMessage)
	balanceMu          sync.RWMutex
	orderCallbacks     map[int]func(*OrderMessage)
	orderMu            sync.RWMutex

	// Error channel
	errorCh chan error

	// State
	ctx    context.Context
	cancel context.CancelFunc
}

// NewWebSocketService creates a new Gate.io WebSocket service using pkg/websocket infrastructure
func NewWebSocketService(
	connManager connection.ConnectionManager,
	reconnectMgr connection.ReconnectManager,
	baseService base.BaseService,
	logger logging.ApplicationLogger,
) *WebSocketService {
	ws := &WebSocketService{
		connManager:        connManager,
		reconnectMgr:       reconnectMgr,
		baseService:        baseService,
		logger:             logger,
		subscriptions:      make(map[int]*SubscriptionHandler),
		subscriptionIndex:  make(map[string][]*SubscriptionHandler),
		orderBookCallbacks: make(map[int]func(*OrderBookMessage)),
		tradesCallbacks:    make(map[int]func([]TradeMessage)),
		klinesCallbacks:    make(map[int]func(*KlineMessage)),
		balanceCallbacks:   make(map[int]func(*AccountBalanceMessage)),
		orderCallbacks:     make(map[int]func(*OrderMessage)),
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
func (ws *WebSocketService) Connect(wsURL string) error {
	ws.ctx = context.Background()
	ws.logger.Info(fmt.Sprintf("🔌 Connecting to Gate.io WebSocket: %s", wsURL))

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
func (ws *WebSocketService) Disconnect() error {
	ws.logger.Info("Closing WebSocket connection")
	return ws.connManager.Disconnect()
}

// IsConnected returns whether the WebSocket is currently connected
func (ws *WebSocketService) IsConnected() bool {
	return ws.connManager.GetState() == connection.StateConnected
}

// GetErrorChannel returns the error channel
func (ws *WebSocketService) GetErrorChannel() <-chan error {
	return ws.errorCh
}

// Connection lifecycle callbacks
func (ws *WebSocketService) onConnect() error {
	ws.logger.Info("✅ WebSocket connected")
	return nil
}

func (ws *WebSocketService) onDisconnect() error {
	ws.logger.Info("WebSocket disconnected")
	return nil
}

func (ws *WebSocketService) onMessage(message []byte) error {
	// Use BaseService for rate limiting & validation - pass the callback
	if err := ws.baseService.ProcessMessage(message, func(validatedMsg []byte) error {
		return ws.handleValidatedMessage(validatedMsg)
	}); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}
	return nil
}

func (ws *WebSocketService) handleValidatedMessage(message []byte) error {

	// Parse Gate.io WebSocket message
	var gateMsg map[string]interface{}
	if err := json.Unmarshal(message, &gateMsg); err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Check for event type (subscribe/unsubscribe responses)
	if event, ok := gateMsg["event"].(string); ok {
		if event == "subscribe" || event == "unsubscribe" {
			ws.logger.Debug("Received %s acknowledgment", event)
			return nil
		}
	}

	// Route based on channel
	channel, ok := gateMsg["channel"].(string)
	if !ok {
		ws.logger.Debug("Message without channel", "message", string(message))
		return nil
	}

	// Handle different message types
	switch channel {
	case "spot.order_book":
		return ws.handleOrderBookMessage(gateMsg)
	case "spot.trades":
		return ws.handleTradesMessage(gateMsg)
	case "spot.candlesticks":
		return ws.handleKlineMessage(gateMsg)
	case "spot.balances":
		return ws.handleBalanceMessage(gateMsg)
	case "spot.orders":
		return ws.handleOrderMessage(gateMsg)
	default:
		ws.logger.Debug("Unknown channel", "channel", channel)
	}

	return nil
}

func (ws *WebSocketService) onError(err error) {
	ws.logger.Error("WebSocket error: %v", err)
	select {
	case ws.errorCh <- err:
	default:
	}
}

// Reconnection callbacks
func (ws *WebSocketService) onReconnectStart(attempt int) {
	ws.logger.Info("🔄 Reconnection attempt %d", attempt)
}

func (ws *WebSocketService) onReconnectFail(attempt int, err error) {
	ws.logger.Warn("❌ Reconnection attempt %d failed: %v", attempt, err)
}

func (ws *WebSocketService) onReconnectSuccess(attempt int) {
	ws.logger.Info("✅ Reconnected after %d attempts", attempt)
	// TODO: Resubscribe to all channels after reconnection
}

// subscribe sends a subscription request to Gate.io
func (ws *WebSocketService) subscribe(channel string, payload []string) error {
	subMsg := map[string]interface{}{
		"time":    time.Now().Unix(),
		"channel": channel,
		"event":   "subscribe",
		"payload": payload,
	}

	data, err := json.Marshal(subMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	return ws.connManager.Send(data)
}

// unsubscribe sends an unsubscription request to Gate.io
func (ws *WebSocketService) unsubscribe(channel string, payload []string) error {
	unsubMsg := map[string]interface{}{
		"time":    time.Now().Unix(),
		"channel": channel,
		"event":   "unsubscribe",
		"payload": payload,
	}

	data, err := json.Marshal(unsubMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscription: %w", err)
	}

	return ws.connManager.Send(data)
}

// generateSubscriptionID generates a unique subscription ID
var globalSubID int64

func generateSubscriptionID() int {
	return int(atomic.AddInt64(&globalSubID, 1))
}

// nextSubscriptionID generates a unique subscription ID
func (ws *WebSocketService) nextSubscriptionID() int {
	return generateSubscriptionID()
}

// addSubscription registers a subscription handler
func (ws *WebSocketService) addSubscription(handler *SubscriptionHandler) {
	ws.subscriptionsMu.Lock()
	defer ws.subscriptionsMu.Unlock()

	ws.subscriptions[handler.ID] = handler

	// Add to index
	key := handler.Channel + ":" + handler.Symbol
	if handler.Interval != "" {
		key += ":" + handler.Interval
	}
	ws.subscriptionIndex[key] = append(ws.subscriptionIndex[key], handler)
}

// removeSubscription removes a subscription handler
func (ws *WebSocketService) removeSubscription(subscriptionID int) *SubscriptionHandler {
	ws.subscriptionsMu.Lock()
	defer ws.subscriptionsMu.Unlock()

	handler, exists := ws.subscriptions[subscriptionID]
	if !exists {
		return nil
	}

	delete(ws.subscriptions, subscriptionID)

	// Remove from index
	key := handler.Channel + ":" + handler.Symbol
	if handler.Interval != "" {
		key += ":" + handler.Interval
	}

	handlers := ws.subscriptionIndex[key]
	for i, h := range handlers {
		if h.ID == subscriptionID {
			ws.subscriptionIndex[key] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	return handler
}
