package pyth

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	priceFeedTypes "github.com/wisp-trading/sdk/pkg/markets/price_feeds/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

const (
	pythHermesWS = "wss://hermes.pyth.network/ws"
)

// Service manages the Pyth Hermes WebSocket connection and emits price updates.
// The connector is transport-only; an ingestor consumes updates and persists them.
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
	Subscribe(feedID string) <-chan priceFeedTypes.PriceFeedUpdate
}

type service struct {
	ws          *websocket.Conn
	mu          sync.RWMutex
	connected   bool
	subscribers map[string][]chan priceFeedTypes.PriceFeedUpdate
	errorCh     chan error
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewService creates a new Pyth price feed service (connector only, no storage).
func NewService() Service {
	return &service{
		subscribers: make(map[string][]chan priceFeedTypes.PriceFeedUpdate),
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

// Subscribe returns a channel that receives price updates for a feed.
// Ingestors subscribe to consume updates and persist them to storage.
func (s *service) Subscribe(feedID string) <-chan priceFeedTypes.PriceFeedUpdate {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan priceFeedTypes.PriceFeedUpdate, 10)
	s.subscribers[feedID] = append(s.subscribers[feedID], ch)
	return ch
}

// readLoop processes incoming WebSocket messages and broadcasts to subscribers
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
			s.handlePriceUpdate(envelope.ID, envelope.Result)
		}
	}
}

// handlePriceUpdate broadcasts updates to all subscribers
func (s *service) handlePriceUpdate(feedID string, result interface{}) {
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
		// Set the feed ID on the update
		update.ID = feedID

		// Broadcast to subscribers (ingestors consume these)
		feedUpdate := priceFeedTypes.PriceFeedUpdate{
			FeedID:    priceFeedTypes.PriceFeedID("pyth:" + update.ID),
			Price:     update.Price,
			Timestamp: update.Timestamp,
			Source:    connector.ExchangeName("pyth"),
		}

		if subs, ok := s.subscribers[feedID]; ok {
			for _, ch := range subs {
				select {
				case ch <- feedUpdate:
				default:
				}
			}
		}
	}
}
