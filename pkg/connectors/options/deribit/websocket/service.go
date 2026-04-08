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

// Service manages the Deribit WebSocket connection and subscription lifecycle.
// Deribit uses JSON-RPC 2.0 over WebSocket — auth is done at the application
// level (not HTTP headers) by sending public/auth immediately after connecting.
type Service interface {
	// Connect opens the WebSocket, authenticates, and enables heartbeats.
	Connect(ctx context.Context, wsURL, clientID, clientSecret string) error
	Disconnect() error
	IsConnected() bool
	ErrorChannel() <-chan error

	// SubscribeToTicker subscribes to ticker.{instrument}.100ms.
	SubscribeToTicker(instrument string, callback func(*TickerData)) error
	UnsubscribeFromTicker(instrument string) error

	// SubscribeToOrderBook subscribes to book.{instrument}.none.20.100ms.
	// Deribit sends a snapshot first, then incremental diffs.
	SubscribeToOrderBook(instrument string, callback func(*OrderBookData)) error
	UnsubscribeFromOrderBook(instrument string) error
}

// service is the concrete implementation.
type service struct {
	connManager  connection.ConnectionManager
	reconnectMgr connection.ReconnectManager
	baseService  base.BaseService
	logger       logging.ApplicationLogger

	clientID     string
	clientSecret string

	// in-flight RPC calls: id → response channel
	pendingMu sync.Mutex
	pending   map[int64]chan *DeribitWSMessage
	nextID    int64

	// ticker subscriptions: instrument → []callback
	tickerMu   sync.RWMutex
	tickerSubs map[string][]func(*TickerData)

	// order book subscriptions: instrument → callback
	obMu   sync.RWMutex
	obSubs map[string]func(*OrderBookData)

	errorCh chan error

	ctx    context.Context
	cancel context.CancelFunc
}

// NewService creates a Deribit WebSocket service.
func NewService(
	connManager connection.ConnectionManager,
	reconnectMgr connection.ReconnectManager,
	baseService base.BaseService,
	logger logging.ApplicationLogger,
) Service {
	s := &service{
		connManager:  connManager,
		reconnectMgr: reconnectMgr,
		baseService:  baseService,
		logger:       logger,
		pending:    make(map[int64]chan *DeribitWSMessage),
		tickerSubs: make(map[string][]func(*TickerData)),
		obSubs:     make(map[string]func(*OrderBookData)),
		errorCh:    make(chan error, 64),
	}

	connManager.SetCallbacks(s.onConnect, s.onDisconnect, s.onMessage, s.onError)
	reconnectMgr.SetCallbacks(s.onReconnectStart, s.onReconnectFail, s.onReconnectSuccess)

	return s
}

// ============================================================================
// Public API
// ============================================================================

func (s *service) Connect(ctx context.Context, wsURL, clientID, clientSecret string) error {
	s.clientID = clientID
	s.clientSecret = clientSecret
	s.ctx, s.cancel = context.WithCancel(ctx)

	s.logger.Infof("Connecting to Deribit WebSocket: %s", wsURL)

	if err := s.connManager.Connect(s.ctx, nil, &wsURL); err != nil {
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	_ = s.reconnectMgr.StartReconnection(s.ctx)

	s.logger.Infof("Deribit WebSocket connected")
	return nil
}

func (s *service) Disconnect() error {
	if s.cancel != nil {
		s.cancel()
	}
	return s.connManager.Disconnect()
}

func (s *service) IsConnected() bool {
	return s.connManager.GetState() == connection.StateConnected
}

func (s *service) ErrorChannel() <-chan error {
	return s.errorCh
}

// SubscribeToTicker subscribes to ticker.{instrument}.100ms.
// Calling this a second time for the same instrument replaces the callback.
func (s *service) SubscribeToTicker(instrument string, callback func(*TickerData)) error {
	channel := fmt.Sprintf("ticker.%s.100ms", instrument)

	s.tickerMu.Lock()
	_, alreadySubscribed := s.tickerSubs[instrument]
	s.tickerSubs[instrument] = []func(*TickerData){callback}
	s.tickerMu.Unlock()

	// Only send subscribe to Deribit when the instrument is new.
	if !alreadySubscribed {
		if err := s.sendSubscribe([]string{channel}); err != nil {
			s.tickerMu.Lock()
			delete(s.tickerSubs, instrument)
			s.tickerMu.Unlock()
			return fmt.Errorf("subscribe to %s failed: %w", channel, err)
		}
		s.logger.Infof("Subscribed to Deribit channel: %s", channel)
	}

	return nil
}

// UnsubscribeFromTicker removes all callbacks for the instrument and sends
// public/unsubscribe to Deribit.
func (s *service) UnsubscribeFromTicker(instrument string) error {
	channel := fmt.Sprintf("ticker.%s.100ms", instrument)

	s.tickerMu.Lock()
	delete(s.tickerSubs, instrument)
	s.tickerMu.Unlock()

	if err := s.sendUnsubscribe([]string{channel}); err != nil {
		return fmt.Errorf("unsubscribe from %s failed: %w", channel, err)
	}
	s.logger.Infof("Unsubscribed from Deribit channel: %s", channel)
	return nil
}

// SubscribeToOrderBook subscribes to book.{instrument}.none.20.100ms.
func (s *service) SubscribeToOrderBook(instrument string, callback func(*OrderBookData)) error {
	channel := fmt.Sprintf("book.%s.none.20.100ms", instrument)

	s.obMu.Lock()
	_, alreadySubscribed := s.obSubs[instrument]
	s.obSubs[instrument] = callback
	s.obMu.Unlock()

	if !alreadySubscribed {
		if err := s.sendSubscribe([]string{channel}); err != nil {
			s.obMu.Lock()
			delete(s.obSubs, instrument)
			s.obMu.Unlock()
			return fmt.Errorf("subscribe to %s failed: %w", channel, err)
		}
		s.logger.Infof("Subscribed to Deribit channel: %s", channel)
	}

	return nil
}

// UnsubscribeFromOrderBook removes the order book subscription for an instrument.
func (s *service) UnsubscribeFromOrderBook(instrument string) error {
	channel := fmt.Sprintf("book.%s.none.20.100ms", instrument)

	s.obMu.Lock()
	delete(s.obSubs, instrument)
	s.obMu.Unlock()

	if err := s.sendUnsubscribe([]string{channel}); err != nil {
		return fmt.Errorf("unsubscribe from %s failed: %w", channel, err)
	}
	s.logger.Infof("Unsubscribed from Deribit channel: %s", channel)
	return nil
}

// ============================================================================
// Connection lifecycle callbacks
// ============================================================================

func (s *service) onConnect() error {
	s.logger.Infof("Deribit WebSocket connection established, authenticating")

	if err := s.authenticate(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if err := s.enableHeartbeat(30); err != nil {
		// Non-fatal: heartbeats are best-effort
		s.logger.Warnf("Failed to enable Deribit heartbeat: %v", err)
	}

	return nil
}

func (s *service) onDisconnect() error {
	s.logger.Infof("Deribit WebSocket disconnected")
	return nil
}

func (s *service) onMessage(raw []byte) error {
	return s.baseService.ProcessMessage(raw, s.handleMessage)
}

func (s *service) onError(err error) {
	s.logger.Errorf("Deribit WebSocket error: %v", err)
	select {
	case s.errorCh <- err:
	default:
	}
}

// ============================================================================
// Reconnection callbacks
// ============================================================================

func (s *service) onReconnectStart(attempt int) {
	s.logger.Infof("Deribit WebSocket reconnection attempt %d", attempt)
}

func (s *service) onReconnectFail(attempt int, err error) {
	s.logger.Warnf("Deribit WebSocket reconnection attempt %d failed: %v", attempt, err)
}

func (s *service) onReconnectSuccess(attempt int) {
	s.logger.Infof("Deribit WebSocket reconnected after %d attempts, resubscribing", attempt)
	s.resubscribeAll()
}

// resubscribeAll restores all active subscriptions after a reconnect.
func (s *service) resubscribeAll() {
	var channels []string

	s.tickerMu.RLock()
	for inst := range s.tickerSubs {
		channels = append(channels, fmt.Sprintf("ticker.%s.100ms", inst))
	}
	s.tickerMu.RUnlock()

	s.obMu.RLock()
	for inst := range s.obSubs {
		channels = append(channels, fmt.Sprintf("book.%s.none.20.100ms", inst))
	}
	s.obMu.RUnlock()

	if len(channels) == 0 {
		return
	}

	if err := s.sendSubscribe(channels); err != nil {
		s.logger.Errorf("Failed to resubscribe after reconnect: %v", err)
	} else {
		s.logger.Infof("Resubscribed to %d channels after reconnect", len(channels))
	}
}

// ============================================================================
// Message handling
// ============================================================================

func (s *service) handleMessage(raw []byte) error {
	var msg DeribitWSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("failed to parse Deribit WS message: %w", err)
	}

	switch msg.Method {
	case "subscription":
		return s.handleSubscription(&msg)
	case "heartbeat":
		return s.handleHeartbeat(&msg)
	case "":
		// RPC response (has id field)
		if msg.ID != nil {
			s.dispatchPending(*msg.ID, &msg)
		}
	}

	return nil
}

func (s *service) handleSubscription(msg *DeribitWSMessage) error {
	if msg.Params == nil {
		return nil
	}

	channel := msg.Params.Channel

	switch {
	case len(channel) > 7 && channel[:7] == "ticker.":
		return s.handleTickerUpdate(channel, msg.Params.Data)
	case len(channel) > 5 && channel[:5] == "book.":
		return s.handleOrderBookUpdate(channel, msg.Params.Data)
	}

	return nil
}

func (s *service) handleTickerUpdate(channel string, data json.RawMessage) error {
	// Extract instrument name from "ticker.{instrument}.{interval}"
	// Strip "ticker." prefix and ".100ms" (or ".raw") suffix
	inner := channel[7:]
	lastDot := len(inner) - 1
	for lastDot >= 0 && inner[lastDot] != '.' {
		lastDot--
	}
	if lastDot < 0 {
		return fmt.Errorf("unexpected ticker channel format: %s", channel)
	}
	instrument := inner[:lastDot]

	var tickerData TickerData
	if err := json.Unmarshal(data, &tickerData); err != nil {
		return fmt.Errorf("failed to parse ticker data for %s: %w", instrument, err)
	}

	s.tickerMu.RLock()
	callbacks := s.tickerSubs[instrument]
	s.tickerMu.RUnlock()

	for _, cb := range callbacks {
		cb(&tickerData)
	}

	return nil
}

func (s *service) handleOrderBookUpdate(channel string, data json.RawMessage) error {
	// channel: "book.{instrument}.none.20.100ms" — extract instrument name.
	// Strip the "book." prefix, then take everything up to the first dot
	// (Deribit instrument names use dashes, not dots, so the first dot marks the end).
	inner := channel[5:]
	end := len(inner)
	for i, c := range inner {
		if c == '.' {
			end = i
			break
		}
	}
	instrument := inner[:end]

	var obData OrderBookData
	if err := json.Unmarshal(data, &obData); err != nil {
		return fmt.Errorf("failed to parse order book data for %s: %w", instrument, err)
	}

	s.obMu.RLock()
	cb, ok := s.obSubs[instrument]
	s.obMu.RUnlock()

	if ok {
		cb(&obData)
	}

	return nil
}

func (s *service) handleHeartbeat(msg *DeribitWSMessage) error {
	if msg.Params == nil || msg.Params.Type != "test_request" {
		return nil
	}

	// Respond to heartbeat with public/test
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.newID(),
		"method":  "public/test",
		"params":  map[string]interface{}{},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat response: %w", err)
	}
	return s.connManager.Send(data)
}

// ============================================================================
// RPC helpers
// ============================================================================

// authenticate sends public/auth and waits for the response.
func (s *service) authenticate() error {
	id := s.newID()
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "public/auth",
		"params": map[string]interface{}{
			"grant_type":    "client_credentials",
			"client_id":     s.clientID,
			"client_secret": s.clientSecret,
		},
	}

	resp, err := s.call(id, req)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("auth error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	s.logger.Infof("Deribit WebSocket authenticated successfully")
	return nil
}

// enableHeartbeat configures Deribit server-side heartbeats at the given interval (seconds).
func (s *service) enableHeartbeat(intervalSeconds int) error {
	id := s.newID()
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "public/set_heartbeat",
		"params": map[string]interface{}{
			"interval": intervalSeconds,
		},
	}

	resp, err := s.call(id, req)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("set_heartbeat error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return nil
}

// sendSubscribe sends a public/subscribe request (fire-and-forget).
func (s *service) sendSubscribe(channels []string) error {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.newID(),
		"method":  "public/subscribe",
		"params": map[string]interface{}{
			"channels": channels,
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal subscribe request: %w", err)
	}
	return s.connManager.Send(data)
}

// sendUnsubscribe sends a public/unsubscribe request (fire-and-forget).
func (s *service) sendUnsubscribe(channels []string) error {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.newID(),
		"method":  "public/unsubscribe",
		"params": map[string]interface{}{
			"channels": channels,
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscribe request: %w", err)
	}
	return s.connManager.Send(data)
}

// call sends a JSON-RPC request and blocks until the response arrives (or ctx cancels).
func (s *service) call(id int64, req map[string]interface{}) (*DeribitWSMessage, error) {
	respCh := make(chan *DeribitWSMessage, 1)

	s.pendingMu.Lock()
	s.pending[id] = respCh
	s.pendingMu.Unlock()

	defer func() {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
	}()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	if err := s.connManager.Send(data); err != nil {
		return nil, fmt.Errorf("failed to send RPC request: %w", err)
	}

	select {
	case resp := <-respCh:
		return resp, nil
	case <-s.ctx.Done():
		return nil, fmt.Errorf("context cancelled waiting for RPC response (id=%d)", id)
	}
}

// dispatchPending routes an RPC response to its waiting call() goroutine.
func (s *service) dispatchPending(id int64, msg *DeribitWSMessage) {
	s.pendingMu.Lock()
	ch, ok := s.pending[id]
	s.pendingMu.Unlock()

	if ok {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *service) newID() int64 {
	return atomic.AddInt64(&s.nextID, 1)
}
