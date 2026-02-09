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

// WebSocketService manages the Bybit WebSocket connection using pkg/websocket infrastructure
type WebSocketService struct {
	connManager  connection.ConnectionManager
	reconnectMgr connection.ReconnectManager
	baseService  base.BaseService
	logger       logging.ApplicationLogger

	// Subscription tracking
	subscriptionsMu   sync.RWMutex
	subscriptions     map[int]*SubscriptionHandler
	subscriptionID    int64
	subscriptionIndex map[string][]*SubscriptionHandler // topic -> handlers

	// Parsed callbacks
	orderBookCallbacks map[int]func(*OrderBookMessage)
	orderBookMu        sync.RWMutex
	tradesCallbacks    map[int]func([]TradeMessage)
	tradesMu           sync.RWMutex
	klinesCallbacks    map[int]func(*KlineMessage)
	klinesMu           sync.RWMutex
	positionCallbacks  map[int]func(*PositionMessage)
	positionMu         sync.RWMutex
	balanceCallbacks   map[int]func(*AccountBalanceMessage)
	balanceMu          sync.RWMutex

	// Error channel
	errorCh chan error

	// State
	ctx    context.Context
	cancel context.CancelFunc
}

// NewWebSocketService creates a new Bybit WebSocket service using pkg/websocket infrastructure
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
		positionCallbacks:  make(map[int]func(*PositionMessage)),
		balanceCallbacks:   make(map[int]func(*AccountBalanceMessage)),
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
	ws.ctx, ws.cancel = context.WithCancel(context.Background())
	ws.logger.Info(fmt.Sprintf("🔌 Connecting to Bybit WebSocket: %s", wsURL))

	// Pass the URL to connection manager
	if err := ws.connManager.Connect(ws.ctx, &wsURL); err != nil {
		ws.logger.Error("❌ Failed to connect to WebSocket: %v", err)
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	ws.logger.Info("✅ WebSocket connected successfully")
	_ = ws.reconnectMgr.StartReconnection(ws.ctx) // Ignore error as it may already be running
	return nil
}

// Disconnect closes the WebSocket connection
func (ws *WebSocketService) Disconnect() error {
	if ws.cancel != nil {
		ws.cancel()
	}
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

// ============================================================================
// Connection lifecycle callbacks
// ============================================================================

func (ws *WebSocketService) onConnect() error {
	ws.logger.Info("Bybit WebSocket connected")
	return nil
}

func (ws *WebSocketService) onDisconnect() error {
	ws.logger.Info("Bybit WebSocket disconnected")
	return nil
}

func (ws *WebSocketService) onMessage(message []byte) error {
	// Use BaseService for rate limiting & validation
	if err := ws.baseService.ProcessMessage(message, func(validatedMsg []byte) error {
		return ws.handleValidatedMessage(validatedMsg)
	}); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}
	return nil
}

func (ws *WebSocketService) handleValidatedMessage(message []byte) error {
	// Parse Bybit WebSocket message
	var bybitMsg BybitWSMessage
	if err := json.Unmarshal(message, &bybitMsg); err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Handle subscription responses
	if bybitMsg.Op == "subscribe" || bybitMsg.Op == "unsubscribe" {
		ws.logger.Debug("Received %s acknowledgment: %s", bybitMsg.Op, bybitMsg.RetMsg)
		return nil
	}

	// Handle ping/pong
	if bybitMsg.Op == "ping" {
		return ws.sendPong()
	}

	// Route based on topic
	if bybitMsg.Topic == "" {
		ws.logger.Debug("Message without topic", "message", string(message))
		return nil
	}

	// Handle different message types based on topic prefix
	return ws.routeMessageByTopic(bybitMsg.Topic, bybitMsg.Data, bybitMsg.Ts)
}

func (ws *WebSocketService) routeMessageByTopic(topic string, data interface{}, timestamp int64) error {
	// Extract topic type (orderbook, publicTrade, kline, position, wallet)
	// Bybit format: "orderbook.50.BTCUSDT", "publicTrade.BTCUSDT", "kline.1.BTCUSDT", "position", "wallet"

	// Check for orderbook
	if len(topic) > 9 && topic[:9] == "orderbook" {
		return ws.handleOrderBookMessage(topic, data, timestamp)
	}

	// Check for trades
	if len(topic) > 11 && topic[:11] == "publicTrade" {
		return ws.handleTradesMessage(topic, data, timestamp)
	}

	// Check for klines
	if len(topic) > 5 && topic[:5] == "kline" {
		return ws.handleKlineMessage(topic, data, timestamp)
	}

	// Check for positions
	if topic == "position" {
		return ws.handlePositionMessage(data, timestamp)
	}

	// Check for wallet/balance
	if topic == "wallet" {
		return ws.handleBalanceMessage(data, timestamp)
	}

	ws.logger.Debug("Unknown topic", "topic", topic)
	return nil
}

func (ws *WebSocketService) onError(err error) {
	ws.logger.Error("WebSocket error: %v", err)
	select {
	case ws.errorCh <- err:
	default:
	}
}

// ============================================================================
// Reconnection callbacks
// ============================================================================

func (ws *WebSocketService) onReconnectStart(attempt int) {
	ws.logger.Info("🔄 Reconnection attempt %d", attempt)
}

func (ws *WebSocketService) onReconnectFail(attempt int, err error) {
	ws.logger.Warn("❌ Reconnection attempt %d failed: %v", attempt, err)
}

func (ws *WebSocketService) onReconnectSuccess(attempt int) {
	ws.logger.Info("✅ Reconnected after %d attempts", attempt)
	// TODO: Resubscribe to all channels after reconnection
	ws.resubscribeAll()
}

func (ws *WebSocketService) resubscribeAll() {
	ws.subscriptionsMu.RLock()
	defer ws.subscriptionsMu.RUnlock()

	// Collect unique topics to resubscribe
	topics := make(map[string]bool)
	for _, handler := range ws.subscriptions {
		topicKey := ws.buildTopicKey(handler.Channel, handler.Symbol, handler.Interval)
		topics[topicKey] = true
	}

	// Resubscribe to all topics
	if len(topics) > 0 {
		topicList := make([]string, 0, len(topics))
		for topic := range topics {
			topicList = append(topicList, topic)
		}
		ws.logger.Info("Resubscribing to %d topics after reconnection", len(topicList))
		if err := ws.subscribe(topicList); err != nil {
			ws.logger.Error("Failed to resubscribe after reconnection: %v", err)
		}
	}
}

// ============================================================================
// Subscription management
// ============================================================================

func (ws *WebSocketService) subscribe(topics []string) error {
	subMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": topics,
	}

	data, err := json.Marshal(subMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	return ws.connManager.Send(data)
}

func (ws *WebSocketService) unsubscribe(topics []string) error {
	unsubMsg := map[string]interface{}{
		"op":   "unsubscribe",
		"args": topics,
	}

	data, err := json.Marshal(unsubMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscription: %w", err)
	}

	return ws.connManager.Send(data)
}

func (ws *WebSocketService) sendPong() error {
	pongMsg := map[string]interface{}{
		"op": "pong",
	}

	data, err := json.Marshal(pongMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal pong: %w", err)
	}

	return ws.connManager.Send(data)
}

func (ws *WebSocketService) nextSubscriptionID() int {
	return int(atomic.AddInt64(&ws.subscriptionID, 1))
}

func (ws *WebSocketService) buildTopicKey(channel, symbol, interval string) string {
	if channel == "kline" {
		return fmt.Sprintf("kline.%s.%s", interval, symbol)
	}
	if channel == "orderbook" {
		return fmt.Sprintf("orderbook.50.%s", symbol) // Using depth 50
	}
	if channel == "publicTrade" {
		return fmt.Sprintf("publicTrade.%s", symbol)
	}
	if channel == "position" {
		return "position"
	}
	if channel == "wallet" {
		return "wallet"
	}
	return channel
}

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

// ============================================================================
// Subscription API
// ============================================================================

func (ws *WebSocketService) SubscribeToOrderBook(symbol string, callback func(*OrderBookMessage)) (int, error) {
	id := ws.nextSubscriptionID()

	handler := &SubscriptionHandler{
		ID:       id,
		Channel:  "orderbook",
		Symbol:   symbol,
		Callback: callback,
	}

	ws.addSubscription(handler)

	ws.orderBookMu.Lock()
	ws.orderBookCallbacks[id] = callback
	ws.orderBookMu.Unlock()

	topic := fmt.Sprintf("orderbook.50.%s", symbol)
	if err := ws.subscribe([]string{topic}); err != nil {
		ws.removeSubscription(id)
		ws.orderBookMu.Lock()
		delete(ws.orderBookCallbacks, id)
		ws.orderBookMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to orderbook", "symbol", symbol, "id", id)
	return id, nil
}

func (ws *WebSocketService) UnsubscribeFromOrderBook(symbol string, subscriptionID int) error {
	handler := ws.removeSubscription(subscriptionID)
	if handler == nil {
		return fmt.Errorf("subscription not found")
	}

	ws.orderBookMu.Lock()
	delete(ws.orderBookCallbacks, subscriptionID)
	ws.orderBookMu.Unlock()

	topic := fmt.Sprintf("orderbook.50.%s", symbol)
	if err := ws.unsubscribe([]string{topic}); err != nil {
		ws.logger.Warn("Failed to unsubscribe from orderbook", "error", err)
	}

	ws.logger.Info("Unsubscribed from orderbook", "symbol", symbol)
	return nil
}

func (ws *WebSocketService) SubscribeToTrades(symbol string, callback func([]TradeMessage)) (int, error) {
	id := ws.nextSubscriptionID()

	handler := &SubscriptionHandler{
		ID:       id,
		Channel:  "publicTrade",
		Symbol:   symbol,
		Callback: callback,
	}

	ws.addSubscription(handler)

	ws.tradesMu.Lock()
	ws.tradesCallbacks[id] = callback
	ws.tradesMu.Unlock()

	topic := fmt.Sprintf("publicTrade.%s", symbol)
	if err := ws.subscribe([]string{topic}); err != nil {
		ws.removeSubscription(id)
		ws.tradesMu.Lock()
		delete(ws.tradesCallbacks, id)
		ws.tradesMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to trades", "symbol", symbol, "id", id)
	return id, nil
}

func (ws *WebSocketService) UnsubscribeFromTrades(symbol string, subscriptionID int) error {
	handler := ws.removeSubscription(subscriptionID)
	if handler == nil {
		return fmt.Errorf("subscription not found")
	}

	ws.tradesMu.Lock()
	delete(ws.tradesCallbacks, subscriptionID)
	ws.tradesMu.Unlock()

	topic := fmt.Sprintf("publicTrade.%s", symbol)
	if err := ws.unsubscribe([]string{topic}); err != nil {
		ws.logger.Warn("Failed to unsubscribe from trades", "error", err)
	}

	ws.logger.Info("Unsubscribed from trades", "symbol", symbol)
	return nil
}

func (ws *WebSocketService) SubscribeToKlines(symbol, interval string, callback func(*KlineMessage)) (int, error) {
	id := ws.nextSubscriptionID()

	handler := &SubscriptionHandler{
		ID:       id,
		Channel:  "kline",
		Symbol:   symbol,
		Interval: interval,
		Callback: callback,
	}

	ws.addSubscription(handler)

	ws.klinesMu.Lock()
	ws.klinesCallbacks[id] = callback
	ws.klinesMu.Unlock()

	topic := fmt.Sprintf("kline.%s.%s", interval, symbol)
	if err := ws.subscribe([]string{topic}); err != nil {
		ws.removeSubscription(id)
		ws.klinesMu.Lock()
		delete(ws.klinesCallbacks, id)
		ws.klinesMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to klines", "symbol", symbol, "interval", interval, "id", id)
	return id, nil
}

func (ws *WebSocketService) UnsubscribeFromKlines(symbol, interval string, subscriptionID int) error {
	handler := ws.removeSubscription(subscriptionID)
	if handler == nil {
		return fmt.Errorf("subscription not found")
	}

	ws.klinesMu.Lock()
	delete(ws.klinesCallbacks, subscriptionID)
	ws.klinesMu.Unlock()

	topic := fmt.Sprintf("kline.%s.%s", interval, symbol)
	if err := ws.unsubscribe([]string{topic}); err != nil {
		ws.logger.Warn("Failed to unsubscribe from klines", "error", err)
	}

	ws.logger.Info("Unsubscribed from klines", "symbol", symbol, "interval", interval)
	return nil
}

func (ws *WebSocketService) SubscribeToPositions(callback func(*PositionMessage)) (int, error) {
	id := ws.nextSubscriptionID()

	handler := &SubscriptionHandler{
		ID:       id,
		Channel:  "position",
		Callback: callback,
	}

	ws.addSubscription(handler)

	ws.positionMu.Lock()
	ws.positionCallbacks[id] = callback
	ws.positionMu.Unlock()

	if err := ws.subscribe([]string{"position"}); err != nil {
		ws.removeSubscription(id)
		ws.positionMu.Lock()
		delete(ws.positionCallbacks, id)
		ws.positionMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to positions", "id", id)
	return id, nil
}

func (ws *WebSocketService) UnsubscribeFromPositions(subscriptionID int) error {
	handler := ws.removeSubscription(subscriptionID)
	if handler == nil {
		return fmt.Errorf("subscription not found")
	}

	ws.positionMu.Lock()
	delete(ws.positionCallbacks, subscriptionID)
	ws.positionMu.Unlock()

	if err := ws.unsubscribe([]string{"position"}); err != nil {
		ws.logger.Warn("Failed to unsubscribe from positions", "error", err)
	}

	ws.logger.Info("Unsubscribed from positions")
	return nil
}

func (ws *WebSocketService) SubscribeToAccountBalance(callback func(*AccountBalanceMessage)) (int, error) {
	id := ws.nextSubscriptionID()

	handler := &SubscriptionHandler{
		ID:       id,
		Channel:  "wallet",
		Callback: callback,
	}

	ws.addSubscription(handler)

	ws.balanceMu.Lock()
	ws.balanceCallbacks[id] = callback
	ws.balanceMu.Unlock()

	if err := ws.subscribe([]string{"wallet"}); err != nil {
		ws.removeSubscription(id)
		ws.balanceMu.Lock()
		delete(ws.balanceCallbacks, id)
		ws.balanceMu.Unlock()
		return 0, err
	}

	ws.logger.Info("Subscribed to account balance", "id", id)
	return id, nil
}

func (ws *WebSocketService) UnsubscribeFromAccountBalance(subscriptionID int) error {
	handler := ws.removeSubscription(subscriptionID)
	if handler == nil {
		return fmt.Errorf("subscription not found")
	}

	ws.balanceMu.Lock()
	delete(ws.balanceCallbacks, subscriptionID)
	ws.balanceMu.Unlock()

	if err := ws.unsubscribe([]string{"wallet"}); err != nil {
		ws.logger.Warn("Failed to unsubscribe from account balance", "error", err)
	}

	ws.logger.Info("Unsubscribed from account balance")
	return nil
}
