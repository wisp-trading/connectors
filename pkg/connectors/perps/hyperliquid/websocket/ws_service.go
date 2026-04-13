package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sonirico/go-hyperliquid"
	"github.com/wisp-trading/connectors/pkg/websocket/base"
	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	"github.com/wisp-trading/sdk/pkg/types/logging"
)

// WebSocketService manages the WebSocket connection using the robust pkg/websocket infrastructure
type WebSocketService struct {
	connManager  connection.ConnectionManager
	reconnectMgr connection.ReconnectManager
	baseService  base.BaseService
	logger       logging.ApplicationLogger
	parser       MessageParser

	// Subscription tracking
	subscriptionsMu sync.RWMutex
	subscriptions   map[int]*SubscriptionHandler // Map subscription ID -> handler

	// Subscription index for direct routing (no iteration)
	// Key format: "channel:coin:interval" (e.g., "l2Book:BTC:", "candle:ETH:1m")
	subscriptionIndex map[string][]*SubscriptionHandler
	indexMu           sync.RWMutex

	// Message routing
	messageHandlers map[string]func([]byte) error // Channel -> handler
	handlersMu      sync.RWMutex

	// Parsed callbacks
	orderBookCallbacks    map[int]func(*OrderBookMessage)
	orderBookMu           sync.RWMutex
	tradesCallbacks       map[int]func([]TradeMessage)
	tradesMu              sync.RWMutex
	klinesCallbacks       map[int]func(*KlineMessage)
	klinesMu              sync.RWMutex
	fundingRatesCallbacks map[int]func(*FundingRateMessage)
	fundingRatesMu        sync.RWMutex

	// Error channel
	errorCh chan error

	// State
	ctx    context.Context
	cancel context.CancelFunc
}

// SubscriptionHandler tracks an active subscription
type SubscriptionHandler struct {
	ID       int
	Channel  string
	Coin     string
	Interval string
	Callback func(hyperliquid.WSMessage)
}

// NewWebSocketService creates a new WebSocket service using pkg/websocket infrastructure
// All dependencies are injected via DI - no instantiation with new()
func NewWebSocketService(
	connManager connection.ConnectionManager,
	reconnectMgr connection.ReconnectManager,
	baseService base.BaseService,
	logger logging.ApplicationLogger,
	parser MessageParser,
) (RealTimeService, error) {
	ws := &WebSocketService{
		connManager:           connManager,
		reconnectMgr:          reconnectMgr,
		baseService:           baseService,
		logger:                logger,
		parser:                parser,
		subscriptions:         make(map[int]*SubscriptionHandler),
		subscriptionIndex:     make(map[string][]*SubscriptionHandler),
		messageHandlers:       make(map[string]func([]byte) error),
		orderBookCallbacks:    make(map[int]func(*OrderBookMessage)),
		tradesCallbacks:       make(map[int]func([]TradeMessage)),
		klinesCallbacks:       make(map[int]func(*KlineMessage)),
		fundingRatesCallbacks: make(map[int]func(*FundingRateMessage)),
		errorCh:               make(chan error, 100),
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

	return ws, nil
}

// Connect establishes the WebSocket connection with automatic reconnection
func (ws *WebSocketService) Connect(websocketUrl *string) error {
	// Create a background context that will NEVER be cancelled
	// This allows the connection to stay alive independent of caller's context
	ws.ctx = context.Background()
	ws.cancel = nil

	ws.logger.Info("🔌 Connecting to WebSocket: %s", ws.connManager.GetState())

	// Connect - pass the websocket URL and a never-cancelling context
	if err := ws.connManager.Connect(ws.ctx, nil, websocketUrl); err != nil {
		ws.logger.Error("❌ Failed to connect to WebSocket: %v", err)
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	ws.logger.Info("✅ WebSocket connected successfully")

	// START the reconnection manager - this is critical!
	// Without this, the connection will close and never reconnect
	// StartReconnection spawns a goroutine to watch for disconnections
	ws.reconnectMgr.StartReconnection(ws.ctx)

	return nil
}

// Close disconnects the WebSocket
func (ws *WebSocketService) Close() error {
	ws.logger.Info("Closing WebSocket connection")

	return ws.connManager.Disconnect()
}

// IsConnected returns whether the WebSocket is currently connected
func (ws *WebSocketService) IsConnected() bool {
	return ws.connManager.GetState() == connection.StateConnected
}

// GetMetrics returns connection and message metrics
func (ws *WebSocketService) GetMetrics() map[string]interface{} {
	stats := ws.connManager.GetConnectionStats()

	// Add subscription count
	ws.subscriptionsMu.RLock()
	stats["active_subscriptions"] = len(ws.subscriptions)
	ws.subscriptionsMu.RUnlock()

	return stats
}

// GetErrorChannel returns the error channel for consumers
func (ws *WebSocketService) GetErrorChannel() <-chan error {
	return ws.errorCh
}

// onConnect is called when the connection is established
func (ws *WebSocketService) onConnect() error {
	ws.logger.Info("✅ WebSocket connected")
	return nil
}

// onDisconnect is called when the connection is lost
func (ws *WebSocketService) onDisconnect() error {
	ws.logger.Error("🔌❌ WebSocket disconnected - connection lost, will attempt reconnection")
	// Re-subscription will happen on reconnect via resubscribeAll
	return nil
}

// onMessage processes incoming WebSocket messages
func (ws *WebSocketService) onMessage(data []byte) error {
	// Parse the message to determine its channel
	var msgWrapper struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msgWrapper); err != nil {
		ws.logger.Warn("❌ Failed to unmarshal message wrapper: %v | Raw: %s", err, string(data))
		return nil // Don't error on unparseable messages
	}

	// Handle subscription confirmation messages (no routing needed)
	if msgWrapper.Channel == "subscriptionResponse" {
		ws.logger.Info("✅ Subscription confirmed: %s", string(msgWrapper.Data))
		return nil
	}

	// Find handler for this channel
	ws.handlersMu.RLock()
	handler, exists := ws.messageHandlers[msgWrapper.Channel]
	ws.handlersMu.RUnlock()

	if !exists {
		ws.logger.Warn("⚠️  No handler registered for channel: '%s'", msgWrapper.Channel)
		return nil
	}

	ws.logger.Debug("✅ Routing to handler for channel '%s'", msgWrapper.Channel)

	// Call handler
	if err := handler(data); err != nil {
		ws.logger.Warn("Handler error for channel %s: %v", msgWrapper.Channel, err)
		select {
		case ws.errorCh <- fmt.Errorf("message handler error for %s: %w", msgWrapper.Channel, err):
		default:
		}
	}

	return nil
}

// onError handles errors from the connection manager
func (ws *WebSocketService) onError(err error) {
	ws.logger.Error("❌ WebSocket error: %v", err)
	select {
	case ws.errorCh <- err:
	default:
		ws.logger.Warn("Error channel full, dropping error")
	}
}

// onReconnectStart is called when reconnection attempt starts
func (ws *WebSocketService) onReconnectStart(attempt int) {
	ws.logger.Info("🔄 Reconnection attempt %d", attempt)
}

// onReconnectFail is called when a reconnection attempt fails
func (ws *WebSocketService) onReconnectFail(attempt int, err error) {
	ws.logger.Warn("❌ Reconnection attempt %d failed: %v", attempt, err)
	select {
	case ws.errorCh <- fmt.Errorf("reconnection failed (attempt %d): %w", attempt, err):
	default:
	}
}

// onReconnectSuccess is called when reconnection succeeds
func (ws *WebSocketService) onReconnectSuccess(attempt int) {
	ws.logger.Info("✅ Reconnection successful (attempt %d)", attempt)
	// Re-establish subscriptions
	ws.resubscribeAll()
}

// resubscribeAll re-subscribes to all tracked subscriptions
func (ws *WebSocketService) resubscribeAll() {
	ws.subscriptionsMu.RLock()
	subscriptions := make([]*SubscriptionHandler, 0, len(ws.subscriptions))
	for _, sub := range ws.subscriptions {
		subscriptions = append(subscriptions, sub)
	}
	ws.subscriptionsMu.RUnlock()

	ws.logger.Info("Re-subscribing to %d subscriptions after reconnect", len(subscriptions))

	for _, sub := range subscriptions {
		// Re-subscribe based on channel type
		switch sub.Channel {
		case "l2Book":
			ws.logger.Debug("Re-subscribing to orderbook: %s", sub.Coin)
			// The subscription will be re-sent to server
		case "trades":
			ws.logger.Debug("Re-subscribing to trades: %s", sub.Coin)
		case "candle":
			ws.logger.Debug("Re-subscribing to candles: %s %s", sub.Coin, sub.Interval)
		case "webData2":
			ws.logger.Debug("Re-subscribing to webData2")
		case "activeAssetCtx":
			ws.logger.Debug("Re-subscribing to funding rates: %s", sub.Coin)
		}
	}
}

// subscribeToChannel is the internal method that handles raw subscriptions
func (ws *WebSocketService) subscribeToChannel(channel, coin, interval string, callback func(hyperliquid.WSMessage)) (int, error) {
	fmt.Printf("🟢 subscribeToChannel CALLED: channel=%s, coin=%s, interval=%s\n", channel, coin, interval)

	subID := generateSubscriptionID()
	fmt.Printf("🟢 Generated subID=%d for %s/%s/%s\n", subID, channel, coin, interval)

	sub := &SubscriptionHandler{
		ID:       subID,
		Channel:  channel,
		Coin:     coin,
		Interval: interval,
		Callback: callback,
	}

	// Store subscription by ID
	ws.subscriptionsMu.Lock()
	ws.subscriptions[subID] = sub
	fmt.Printf("🟢 Stored subscription subID=%d in subscriptions map (total: %d)\n", subID, len(ws.subscriptions))
	ws.subscriptionsMu.Unlock()

	// Add to index for O(1) routing
	indexKey := buildIndexKey(channel, coin, interval)
	ws.indexMu.Lock()
	ws.subscriptionIndex[indexKey] = append(ws.subscriptionIndex[indexKey], sub)
	fmt.Printf("🟢 Added to index with key '%s' (total for this key: %d)\n", indexKey, len(ws.subscriptionIndex[indexKey]))
	ws.indexMu.Unlock()

	// Register message handler for this channel if not already registered
	ws.handlersMu.Lock()
	if _, exists := ws.messageHandlers[channel]; !exists {
		ws.messageHandlers[channel] = func(data []byte) error {
			return ws.routeMessageToSubscriptions(channel, data)
		}
	}

	ws.handlersMu.Unlock()

	err := ws.sendSubscription(channel, coin, interval)
	if err != nil {
		return 0, err
	}

	return subID, nil
}

// buildIndexKey creates a lookup key for subscription routing
// Format: "channel:coin:interval" (e.g., "l2Book:BTC:", "candle:ETH:1m")
func buildIndexKey(channel, coin, interval string) string {
	return fmt.Sprintf("%s:%s:%s", channel, coin, interval)
}

// extractOrderBookCoin extracts coin from l2Book message data
// l2Book messages use "coin" field directly
func (ws *WebSocketService) extractOrderBookCoin(data json.RawMessage) string {
	var msgData map[string]interface{}
	if err := json.Unmarshal(data, &msgData); err != nil {
		fmt.Printf("🔴 extractOrderBookCoin: Failed to unmarshal: %v\n", err)
		return ""
	}

	if coinVal, ok := msgData["coin"].(string); ok {
		return coinVal
	}

	return ""
}

// extractCandleMetadata extracts symbol and interval from candle message data
// Candle messages use "s" for symbol and "i" for interval
func (ws *WebSocketService) extractCandleMetadata(data json.RawMessage) (coin, interval string) {
	var msgData map[string]interface{}
	if err := json.Unmarshal(data, &msgData); err != nil {
		fmt.Printf("🔴 extractCandleMetadata: Failed to unmarshal: %v\n", err)
		return "", ""
	}

	if symbolVal, ok := msgData["s"].(string); ok {
		coin = symbolVal
	}

	if intervalVal, ok := msgData["i"].(string); ok {
		interval = intervalVal
	}

	return coin, interval
}

// extractActiveAssetCtxCoin extracts coin from activeAssetCtx message data
// activeAssetCtx messages use "coin" field directly
func (ws *WebSocketService) extractActiveAssetCtxCoin(data json.RawMessage) string {
	var msgData map[string]interface{}
	if err := json.Unmarshal(data, &msgData); err != nil {
		fmt.Printf("🔴 extractActiveAssetCtxCoin: Failed to unmarshal: %v\n", err)
		return ""
	}

	if coinVal, ok := msgData["coin"].(string); ok {
		return coinVal
	}

	return ""
}

// routeMessageToSubscriptions routes incoming messages to matching subscriptions using O(1) index lookup
func (ws *WebSocketService) routeMessageToSubscriptions(channel string, data []byte) error {
	var msgWrapper struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msgWrapper); err != nil {
		fmt.Printf("🔴 Failed to unmarshal message for routing: %v\n", err)
		ws.logger.Warn("Failed to unmarshal message for routing: %v", err)
		return nil
	}

	// Extract metadata based on channel type
	var coin, interval string
	switch channel {
	case "l2Book":
		coin = ws.extractOrderBookCoin(msgWrapper.Data)
	case "candle":
		coin, interval = ws.extractCandleMetadata(msgWrapper.Data)
	case "activeAssetCtx":
		coin = ws.extractActiveAssetCtxCoin(msgWrapper.Data)
	default:
		fmt.Printf("🔴 Unknown channel type '%s' for metadata extraction\n", channel)
	}

	// Build index key for O(1) lookup
	indexKey := buildIndexKey(channel, coin, interval)

	// Direct O(1) lookup in index
	ws.indexMu.RLock()
	subscriptions := ws.subscriptionIndex[indexKey]
	ws.indexMu.RUnlock()

	if len(subscriptions) == 0 {
		fmt.Printf("🔴 NO SUBSCRIPTIONS FOUND for key '%s'\n", indexKey)
		ws.logger.Debug("⚠️  No subscriptions for %s", indexKey)
		return nil
	}

	// Parse as hyperliquid.WSMessage
	msg := hyperliquid.WSMessage{
		Channel: msgWrapper.Channel,
		Data:    msgWrapper.Data,
	}

	// Call all matching callbacks
	for _, sub := range subscriptions {
		if sub.Callback != nil {
			sub.Callback(msg)
		}
	}

	return nil
}

// sendSubscription sends a subscription message to Hyperliquid
func (ws *WebSocketService) sendSubscription(channel, coin, interval string) error {
	fmt.Printf("🟢 sendSubscription CALLED: channel=%s, coin=%s, interval=%s\n", channel, coin, interval)

	subMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": channel,
			"coin": coin,
		},
	}

	if interval != "" {
		subMsg["subscription"].(map[string]interface{})["interval"] = interval
	}

	data, err := json.Marshal(subMsg)
	if err != nil {
		fmt.Printf("🔴 Failed to marshal subscription: %v\n", err)
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	fmt.Printf("🟢 Sending subscription message: %s\n", string(data))
	err = ws.connManager.Send(data)
	if err != nil {
		fmt.Printf("🔴 connManager.Send FAILED: %v\n", err)
		return err
	}

	fmt.Printf("✅ Subscription message sent successfully\n")
	return nil
}

// Parsing helper functions that use the injected parser

func (ws *WebSocketService) parseOrderBook(msg hyperliquid.WSMessage) (*OrderBookMessage, error) {
	return ws.parser.ParseOrderBook(msg)
}

func (ws *WebSocketService) parseTrades(msg hyperliquid.WSMessage) ([]TradeMessage, error) {
	return ws.parser.ParseTrades(msg)
}

func (ws *WebSocketService) parseKline(msg hyperliquid.WSMessage) (*KlineMessage, error) {
	return ws.parser.ParseKline(msg)
}

// Message handlers for specific channels

func (ws *WebSocketService) handleOrderbookMessage(data []byte) error {
	var msgWrapper struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msgWrapper); err != nil {
		ws.logger.Debug("Failed to unmarshal orderbook message: %v", err)
		return nil
	}

	if msgWrapper.Channel != "l2Book" {
		ws.logger.Warn("Failed to parse orderbook message: expected l2Book channel, got %s", msgWrapper.Channel)
		return nil
	}

	// Call subscribed callbacks
	ws.subscriptionsMu.RLock()
	defer ws.subscriptionsMu.RUnlock()

	for _, sub := range ws.subscriptions {
		if sub.Channel == "l2Book" {
			msg := hyperliquid.WSMessage{Channel: msgWrapper.Channel, Data: msgWrapper.Data}
			sub.Callback(msg)
		}
	}

	return nil
}

func (ws *WebSocketService) handleTradesMessage(data []byte) error {
	var msgWrapper struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msgWrapper); err != nil {
		ws.logger.Debug("Failed to unmarshal trades message: %v", err)
		return nil
	}

	if msgWrapper.Channel != "trades" {
		ws.logger.Warn("Failed to parse trades message: expected trades channel, got %s", msgWrapper.Channel)
		return nil
	}

	// Call subscribed callbacks
	ws.subscriptionsMu.RLock()
	defer ws.subscriptionsMu.RUnlock()

	for _, sub := range ws.subscriptions {
		if sub.Channel == "trades" {
			msg := hyperliquid.WSMessage{Channel: msgWrapper.Channel, Data: msgWrapper.Data}
			sub.Callback(msg)
		}
	}

	return nil
}

func (ws *WebSocketService) handleCandleMessage(data []byte) error {
	var msgWrapper struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msgWrapper); err != nil {
		ws.logger.Debug("Failed to unmarshal candle message: %v", err)
		return nil
	}

	if msgWrapper.Channel != "candle" {
		ws.logger.Warn("Failed to parse candle message: expected candle channel, got %s", msgWrapper.Channel)
		return nil
	}

	// Call subscribed callbacks
	ws.subscriptionsMu.RLock()
	defer ws.subscriptionsMu.RUnlock()

	for _, sub := range ws.subscriptions {
		if sub.Channel == "candle" {
			msg := hyperliquid.WSMessage{Channel: msgWrapper.Channel, Data: msgWrapper.Data}
			sub.Callback(msg)
		}
	}

	return nil
}

// Disconnect closes the connection explicitly
func (ws *WebSocketService) Disconnect() error {
	ws.logger.Info("🛑 Explicit disconnect requested from user")
	return ws.connManager.Disconnect()
}

var (
	subIDCounter int64
	subIDMutex   sync.Mutex
)

func generateSubscriptionID() int {
	subIDMutex.Lock()
	defer subIDMutex.Unlock()
	subIDCounter++
	return int(subIDCounter)
}
