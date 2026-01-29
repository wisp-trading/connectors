package real_time

import (
	"fmt"
	"reflect"
	"sync"

	bybit "github.com/bybit-exchange/bybit.go.api"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/temporal"
)

type Config struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

type RealTimeService interface {
	Initialize(config *Config) error
	Connect() error
	Disconnect() error
	SubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error
	UnsubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error
	SubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error
	UnsubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error
	SubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error
	UnsubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error
	SubscribeAccountBalance() error
	UnsubscribeAccountBalance() error
	SubscribeKlines(asset portfolio.Asset, interval string) error
	UnsubscribeKlines(asset portfolio.Asset, interval string) error
}

type realTimeService struct {
	websocket     *bybit.WebSocket
	logger        logging.ApplicationLogger
	timeProvider  temporal.TimeProvider
	mu            sync.RWMutex
	subscriptions map[string]bool
}

func NewRealTimeService(
	logger logging.ApplicationLogger,
	timeProvider temporal.TimeProvider,
) RealTimeService {
	return &realTimeService{
		logger:        logger,
		timeProvider:  timeProvider,
		subscriptions: make(map[string]bool),
	}
}

func (r *realTimeService) Initialize(config *Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket != nil {
		return fmt.Errorf("real-time service already initialized")
	}

	r.websocket = bybit.NewBybitPrivateWebSocket(config.BaseURL, config.APIKey, config.APISecret, func(message string) error {
		return nil
	})

	return nil
}

func (r *realTimeService) Connect() error {
	r.mu.RLock()
	ws := r.websocket
	r.mu.RUnlock()

	if ws == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	ws.Connect()
	return nil
}

func (r *realTimeService) Disconnect() error {
	r.mu.RLock()
	ws := r.websocket
	r.mu.RUnlock()

	if ws == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	// Bybit SDK doesn't have a Close method, connection is managed automatically
	return nil
}

// sendUnsubscribe sends an unsubscribe message to the Bybit WebSocket
// Format: {"op": "unsubscribe", "args": ["channel.symbol"], "req_id": "..."}
func (r *realTimeService) sendUnsubscribe(channels []string) error {
	if r.websocket == nil {
		return fmt.Errorf("websocket not initialized")
	}

	// Build unsubscribe message manually and send it
	// Bybit format: {"req_id": "xxx", "op": "unsubscribe", "args": ["orderbook.50.BTCUSDT"]}
	reqID := fmt.Sprintf("unsub_%d", r.timeProvider.Now().Unix())

	unsubMessage := map[string]interface{}{
		"req_id": reqID,
		"op":     "unsubscribe",
		"args":   channels,
	}

	// Use reflection to access the private sendAsJson method
	// This is the only way since the SDK doesn't expose unsubscribe properly
	wsValue := reflect.ValueOf(r.websocket)
	if wsValue.Kind() == reflect.Ptr {
		wsValue = wsValue.Elem()
	}

	sendMethod := wsValue.MethodByName("sendAsJson")
	if !sendMethod.IsValid() {
		// Try lowercase
		sendMethod = wsValue.MethodByName("SendAsJson")
	}

	if sendMethod.IsValid() {
		results := sendMethod.Call([]reflect.Value{reflect.ValueOf(unsubMessage)})
		if len(results) > 0 && !results[0].IsNil() {
			return fmt.Errorf("failed to send unsubscribe: %v", results[0].Interface())
		}
		return nil
	}

	return fmt.Errorf("unable to send unsubscribe message - SDK method not accessible")
}

func (r *realTimeService) SubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	symbol := asset.Symbol() + "USDT"
	key := "orderbook:" + symbol

	if r.subscriptions[key] {
		return nil
	}

	// Subscribe via WebSocket - orderbook.{depth}.{symbol}
	// Using depth 50 for detailed order book
	_, err := r.websocket.SendSubscription([]string{fmt.Sprintf("orderbook.50.%s", symbol)})
	if err != nil {
		return fmt.Errorf("failed to subscribe to order book: %w", err)
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to order book", "symbol", symbol)
	return nil
}

func (r *realTimeService) UnsubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	symbol := asset.Symbol() + "USDT"
	key := "orderbook:" + symbol

	if !r.subscriptions[key] {
		return nil
	}

	// Send unsubscribe message to Bybit WebSocket
	channels := []string{fmt.Sprintf("orderbook.50.%s", symbol)}
	if err := r.sendUnsubscribe(channels); err != nil {
		r.logger.Warn("Failed to send unsubscribe message", "error", err, "symbol", symbol)
		// Continue to remove from local tracking even if unsubscribe fails
	}

	delete(r.subscriptions, key)
	r.logger.Info("Unsubscribed from order book", "symbol", symbol)
	return nil
}

func (r *realTimeService) SubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	symbol := asset.Symbol() + "USDT"
	key := "trades:" + symbol

	if r.subscriptions[key] {
		return nil
	}

	// Subscribe via WebSocket - publicTrade.{symbol}
	_, err := r.websocket.SendSubscription([]string{fmt.Sprintf("publicTrade.%s", symbol)})
	if err != nil {
		return fmt.Errorf("failed to subscribe to trades: %w", err)
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to trades", "symbol", symbol)
	return nil
}

func (r *realTimeService) UnsubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	symbol := asset.Symbol() + "USDT"
	key := "trades:" + symbol

	if !r.subscriptions[key] {
		return nil
	}

	// Send unsubscribe message to Bybit WebSocket
	channels := []string{fmt.Sprintf("publicTrade.%s", symbol)}
	if err := r.sendUnsubscribe(channels); err != nil {
		r.logger.Warn("Failed to send unsubscribe message", "error", err, "symbol", symbol)
	}

	delete(r.subscriptions, key)
	r.logger.Info("Unsubscribed from trades", "symbol", symbol)
	return nil
}

func (r *realTimeService) SubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	symbol := asset.Symbol() + "USDT"
	key := "positions:" + symbol

	if r.subscriptions[key] {
		return nil
	}

	// Subscribe via WebSocket - "position" private channel (subscribes to all positions)
	_, err := r.websocket.SendSubscription([]string{"position"})
	if err != nil {
		return fmt.Errorf("failed to subscribe to positions: %w", err)
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to positions", "symbol", symbol)
	return nil
}

func (r *realTimeService) UnsubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	symbol := asset.Symbol() + "USDT"
	key := "positions:" + symbol

	if !r.subscriptions[key] {
		return nil
	}

	// Send unsubscribe message to Bybit WebSocket
	channels := []string{"position"}
	if err := r.sendUnsubscribe(channels); err != nil {
		r.logger.Warn("Failed to send unsubscribe message", "error", err, "symbol", symbol)
	}

	delete(r.subscriptions, key)
	r.logger.Info("Unsubscribed from positions", "symbol", symbol)
	return nil
}

func (r *realTimeService) SubscribeAccountBalance() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	key := "balance"

	if r.subscriptions[key] {
		return nil
	}

	// Subscribe via WebSocket - "wallet" private channel
	_, err := r.websocket.SendSubscription([]string{"wallet"})
	if err != nil {
		return fmt.Errorf("failed to subscribe to account balance: %w", err)
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to account balance")
	return nil
}

func (r *realTimeService) UnsubscribeAccountBalance() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.subscriptions["balance"] {
		return nil
	}

	// Send unsubscribe message to Bybit WebSocket
	channels := []string{"wallet"}
	if err := r.sendUnsubscribe(channels); err != nil {
		r.logger.Warn("Failed to send unsubscribe message", "error", err)
	}

	delete(r.subscriptions, "balance")
	r.logger.Info("Unsubscribed from account balance")
	return nil
}

func (r *realTimeService) SubscribeKlines(asset portfolio.Asset, interval string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	symbol := asset.Symbol() + "USDT"
	key := "klines:" + symbol + ":" + interval

	if r.subscriptions[key] {
		return nil
	}

	// Subscribe via WebSocket - kline.{interval}.{symbol}
	_, err := r.websocket.SendSubscription([]string{fmt.Sprintf("kline.%s.%s", interval, symbol)})
	if err != nil {
		return fmt.Errorf("failed to subscribe to klines: %w", err)
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to klines", "symbol", symbol, "interval", interval)
	return nil
}

func (r *realTimeService) UnsubscribeKlines(asset portfolio.Asset, interval string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	symbol := asset.Symbol() + "USDT"
	key := "klines:" + symbol + ":" + interval

	if !r.subscriptions[key] {
		return nil
	}

	// Send unsubscribe message to Bybit WebSocket
	channels := []string{fmt.Sprintf("kline.%s.%s", interval, symbol)}
	if err := r.sendUnsubscribe(channels); err != nil {
		r.logger.Warn("Failed to send unsubscribe message", "error", err, "symbol", symbol, "interval", interval)
	}

	delete(r.subscriptions, key)
	r.logger.Info("Unsubscribed from klines", "symbol", symbol, "interval", interval)
	return nil
}
