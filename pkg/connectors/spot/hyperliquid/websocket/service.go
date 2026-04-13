package websocket

// SpotRealTimeService handles WebSocket connections for spot market data.
// The Hyperliquid WS uses the same endpoint for spot and perp — spot
// subscriptions use the "@N" coin identifier format.
type SpotRealTimeService interface {
	Connect(wsURL string) error
	Disconnect() error
	IsConnected() bool
}

type spotRealTimeService struct {
	connected bool
}

// NewSpotRealTimeService creates a new spot WebSocket service
func NewSpotRealTimeService() SpotRealTimeService {
	return &spotRealTimeService{}
}

func (s *spotRealTimeService) Connect(wsURL string) error {
	s.connected = true
	return nil
}

func (s *spotRealTimeService) Disconnect() error {
	s.connected = false
	return nil
}

func (s *spotRealTimeService) IsConnected() bool {
	return s.connected
}
