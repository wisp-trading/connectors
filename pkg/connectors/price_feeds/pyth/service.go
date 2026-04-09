package pyth

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	priceFeedTypes "github.com/wisp-trading/sdk/pkg/markets/price_feeds/types"
)

const (
	pythHermesWS = "wss://hermes.pyth.network/ws"
)

// Service manages the Pyth Hermes WebSocket connection
// It receives price updates and passes them to a Store for persistence
type Service interface {
	// Connect establishes the WebSocket connection and subscribes to feeds
	Connect(ctx context.Context, feedIDs []string) error
	// Disconnect closes the WebSocket connection
	Disconnect() error
	// IsConnected returns whether the connection is active
	IsConnected() bool
	// ErrorChannel returns a channel of errors from the connection
	ErrorChannel() <-chan error
	// Subscribe returns a channel that receives price updates for the given feed
	Subscribe(feedID string) <-chan priceFeedTypes.PriceSnapshot
}

type service struct {
	ws          *websocket.Conn
	store       priceFeedTypes.PriceFeedsStore
	mu          sync.RWMutex
	connected   bool
	subscribers map[string][]chan priceFeedTypes.PriceSnapshot
	errorCh     chan error
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewService creates a new Pyth price feed service
func NewService(store priceFeedTypes.PriceFeedsStore) Service {
	return &service{
		store:       store,
		subscribers: make(map[string][]chan priceFeedTypes.PriceSnapshot),
		errorCh:     make(chan error, 64),
	}
}

// Connect establishes the WebSocket connection
func (s *service) Connect(ctx context.Context, feedIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return fmt.Errorf("already connected")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)

	// Dial WebSocket
	ws, _, err := websocket.DefaultDialer.DialContext(ctx, pythHermesWS, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	s.ws = ws
	s.connected = true

	// Subscribe to feeds
	subReq := PythSubscribeReq{
		Type: "subscribe",
		IDs:  feedIDs,
	}

	data, err := json.Marshal(subReq)
	if err != nil {
		ws.Close()
		s.connected = false
		return fmt.Errorf("marshal subscribe request: %w", err)
	}

	if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
		ws.Close()
		s.connected = false
		return fmt.Errorf("send subscribe request: %w", err)
	}

	// Start read loop in background
	go s.readLoop()

	return nil
}

// Disconnect closes the WebSocket connection
func (s *service) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}

	if s.ws != nil {
		s.ws.Close()
	}

	s.connected = false
	return nil
}

// IsConnected returns whether the connection is active
func (s *service) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// ErrorChannel returns a channel of errors from the connection
func (s *service) ErrorChannel() <-chan error {
	return s.errorCh
}

// Subscribe returns a channel that receives price updates
func (s *service) Subscribe(feedID string) <-chan priceFeedTypes.PriceSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan priceFeedTypes.PriceSnapshot, 10)
	s.subscribers[feedID] = append(s.subscribers[feedID], ch)
	return ch
}

// readLoop processes incoming WebSocket messages
func (s *service) readLoop() {
	defer func() {
		s.mu.Lock()
		s.connected = false
		s.mu.Unlock()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		_, msg, err := s.ws.ReadMessage()
		if err != nil {
			select {
			case s.errorCh <- fmt.Errorf("websocket read: %w", err):
			default:
			}
			return
		}

		var envelope PythMessage
		if err := json.Unmarshal(msg, &envelope); err != nil {
			continue
		}

		// Handle subscription updates
		if envelope.Type == "price_update" {
			s.handlePriceUpdate(envelope.Result)
		}
	}
}

// handlePriceUpdate processes a price update from Pyth
// Persists to store and broadcasts to subscribers
func (s *service) handlePriceUpdate(result interface{}) {
	data, err := json.Marshal(result)
	if err != nil {
		return
	}

	var updates PriceUpdateMsg
	if err := json.Unmarshal(data, &updates); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, update := range updates.Parsed {
		// Record to store for persistence
		snap := priceFeedTypes.PriceSnapshot{
			FeedID:    priceFeedTypes.PriceFeedID("pyth"),
			Price:     update.Price,
			Timestamp: update.Timestamp,
		}
		_ = s.store.RecordPrice(snap)

		// Broadcast to subscribers
		if subs, ok := s.subscribers["pyth"]; ok {

			for _, ch := range subs {
				select {
				case ch <- snap:
				default:
				}
			}
		}
	}
}
