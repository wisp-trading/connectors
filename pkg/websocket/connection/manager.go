package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/live-trading/pkg/websocket/performance"
	"github.com/backtesting-org/live-trading/pkg/websocket/security"
	"github.com/gorilla/websocket"
)

type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateFailed
	StateStopped // User commanded stop - final state, never reconnect
)

func (cs ConnectionState) String() string {
	switch cs {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateFailed:
		return "failed"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// connectionManager handles WebSocket connection lifecycle with standard ping/pong
type connectionManager struct {
	config         Config
	authManager    security.AuthManager
	metrics        performance.Metrics
	circuitBreaker performance.CircuitBreaker
	logger         logging.ApplicationLogger
	dialer         WebSocketDialer // Abstracted for testability

	conn       WebSocketConn
	state      ConnectionState
	stateMutex sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	// Goroutine lifecycle management
	stopCh   chan struct{}  // Closed when user commands disconnect
	stopOnce sync.Once      // Ensures stopCh only closed once
	wg       sync.WaitGroup // Tracks all goroutines

	lastActivity  time.Time
	activityMutex sync.RWMutex

	onConnect    func() error
	onDisconnect func() error
	onMessage    func([]byte) error
	onError      func(error)
}

func NewConnectionManager(
	config Config,
	authManager security.AuthManager,
	metrics performance.Metrics,
	logger logging.ApplicationLogger,
	dialer WebSocketDialer,
) ConnectionManager {
	return &connectionManager{
		config:         config,
		authManager:    authManager,
		metrics:        metrics,
		circuitBreaker: performance.NewCircuitBreaker(3, 30*time.Second),
		logger:         logger,
		dialer:         dialer,
		state:          StateDisconnected,
		stopCh:         make(chan struct{}),
	}
}

func (cm *connectionManager) SetCallbacks(
	onConnect func() error,
	onDisconnect func() error,
	onMessage func([]byte) error,
	onError func(error),
) {
	cm.onConnect = onConnect
	cm.onDisconnect = onDisconnect
	cm.onMessage = onMessage
	cm.onError = onError
}

func (cm *connectionManager) Connect(
	ctx context.Context,
	websocketUrl *string,
) error {
	cm.stateMutex.Lock()
	defer cm.stateMutex.Unlock()

	if websocketUrl != nil {
		cm.config.URL = *websocketUrl
	}

	if cm.state == StateConnected || cm.state == StateConnecting {
		return fmt.Errorf("already connected or connecting")
	}

	cm.setState(StateConnecting)
	cm.ctx, cm.cancel = context.WithCancel(ctx)

	return cm.circuitBreaker.Execute(func() error {
		return cm.doConnect()
	})
}

func (cm *connectionManager) doConnect() error {
	u, err := url.Parse(cm.config.URL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	if u.Scheme != "wss" {
		return fmt.Errorf("insecure WebSocket scheme: %s (must be wss)", u.Scheme)
	}

	headers, err := cm.authManager.GetSecureHeaders(cm.ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth headers: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(cm.ctx, cm.config.ConnectTimeout)
	defer cancel()

	conn, _, err := cm.dialer.DialContext(connectCtx, u.String(), headers)
	if err != nil {
		cm.setState(StateFailed)
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	// Set read timeout
	if err := conn.SetReadDeadline(time.Now().Add(cm.config.ReadTimeout)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	cm.conn = conn
	cm.setState(StateConnected)
	cm.updateLastActivity()

	// Start core connection handlers
	go cm.readMessages()

	// Optional: Basic health monitoring (configurable)
	if cm.config.EnableHealthMonitoring {
		go cm.simpleHealthMonitor()
	}

	if cm.onConnect != nil {
		if err := cm.onConnect(); err != nil {
			cm.logger.Error("Connect callback failed: %v", err)
			return err
		}
	}

	cm.logger.Info("WebSocket connected successfully to %s", cm.config.URL)
	return nil
}

func (cm *connectionManager) Disconnect() error {
	cm.stateMutex.Lock()

	// If already stopped, just return
	if cm.state == StateStopped {
		cm.stateMutex.Unlock()
		return nil
	}

	// Set state to Stopped (user commanded - never reconnect)
	cm.setState(StateStopped)
	cm.stateMutex.Unlock()

	cm.logger.Info("User commanded disconnect - stopping all goroutines")

	// Signal all goroutines to stop
	cm.stopOnce.Do(func() {
		close(cm.stopCh)
	})

	// Cancel context
	if cm.cancel != nil {
		cm.cancel()
	}

	// Close connection
	var err error
	if cm.conn != nil {
		err = cm.conn.Close()
		cm.conn = nil
	}

	// Wait for all goroutines to exit
	cm.wg.Wait()

	cm.logger.Info("WebSocket disconnected by user - all goroutines stopped")
	return err
}

func (cm *connectionManager) SendMessage(message []byte) error {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	if cm.state != StateConnected || cm.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	if err := cm.conn.SetWriteDeadline(time.Now().Add(cm.config.WriteTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	return cm.conn.WriteMessage(websocket.TextMessage, message)
}

// Send is an alias for SendMessage
func (cm *connectionManager) Send(data []byte) error {
	return cm.SendMessage(data)
}

func (cm *connectionManager) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	if cm.state != StateConnected || cm.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	// Generic debug logging (not exchange-specific)
	cm.logger.Debug("Sending WebSocket message: %s", string(data))

	if err := cm.conn.SetWriteDeadline(time.Now().Add(cm.config.WriteTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	return cm.conn.WriteMessage(websocket.TextMessage, data)
}

func (cm *connectionManager) SendPing() error {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	if cm.state != StateConnected || cm.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	if err := cm.conn.SetWriteDeadline(time.Now().Add(cm.config.WriteTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	cm.logger.Debug("Sending WebSocket ping control frame")
	return cm.conn.WriteMessage(websocket.PingMessage, nil)
}

func (cm *connectionManager) GetState() ConnectionState {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()
	return cm.state
}

func (cm *connectionManager) GetConnectionStats() map[string]interface{} {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	cm.activityMutex.RLock()
	lastActivity := cm.lastActivity
	cm.activityMutex.RUnlock()

	stats := map[string]interface{}{
		"state":         cm.state.String(),
		"connected":     cm.state == StateConnected,
		"last_activity": lastActivity,
		"url":           cm.config.URL,
	}

	if cm.metrics != nil {
		for k, v := range cm.metrics.GetStats() {
			stats[k] = v
		}
	}

	return stats
}

func (cm *connectionManager) IsHealthy() bool {
	if cm.GetState() != StateConnected {
		return false
	}

	cm.activityMutex.RLock()
	lastActivity := cm.lastActivity
	cm.activityMutex.RUnlock()

	// Consider connection healthy if we've had activity recently
	return time.Since(lastActivity) <= cm.config.HealthCheckTimeout
}

func (cm *connectionManager) setState(state ConnectionState) {
	cm.state = state
	cm.logger.Debug("Connection state changed to: %s", state.String())
}

func (cm *connectionManager) updateLastActivity() {
	cm.activityMutex.Lock()
	defer cm.activityMutex.Unlock()
	cm.lastActivity = time.Now()
}

func (cm *connectionManager) readMessages() {
	cm.wg.Add(1)
	defer cm.wg.Done()

	defer func() {
		if r := recover(); r != nil {
			cm.logger.Error("WebSocket read panic: %v", r)
			if cm.GetState() != StateStopped {
				cm.handleConnectionError()
			}
		}
	}()

	for {
		select {
		case <-cm.stopCh:
			cm.logger.Info("Read loop stopping - user disconnect")
			return
		case <-cm.ctx.Done():
			cm.logger.Debug("Read loop cancelled by context")
			return
		default:
		}

		if cm.GetState() == StateStopped {
			return
		}

		if cm.conn == nil {
			cm.logger.Debug("Connection is nil, exiting read loop")
			return
		}

		cm.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		_, message, err := cm.conn.ReadMessage()

		if err != nil {
			if cm.GetState() == StateStopped {
				cm.logger.Debug("Expected read error after user disconnect")
				return
			}

			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				cm.logger.Info("WebSocket closed normally by server")
			} else {
				cm.logger.Error("WebSocket read error: %v", err)
			}

			cm.handleConnectionError()
			return
		}

		cm.updateLastActivity()

		if cm.metrics != nil {
			cm.metrics.IncrementReceived()
		}

		if cm.onMessage != nil {
			if err := cm.onMessage(message); err != nil {
				cm.logger.Debug("Message handler error: %v", err)
				if cm.onError != nil {
					cm.onError(fmt.Errorf("message processing error: %w", err))
				}
			}
		}
	}
}

func (cm *connectionManager) simpleHealthMonitor() {
	cm.wg.Add(1)
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.config.HealthCheckInterval)
	defer ticker.Stop()

	cm.logger.Debug("Starting connection health monitor with %v interval", cm.config.HealthCheckInterval)

	for {
		select {
		case <-cm.stopCh:
			cm.logger.Debug("Health monitor stopping - user disconnect")
			return
		case <-cm.ctx.Done():
			cm.logger.Debug("Health monitor cancelled by context")
			return
		case <-ticker.C:
			if cm.GetState() == StateStopped {
				return
			}

			if cm.GetState() != StateConnected {
				cm.logger.Debug("Health check: not connected")
				return
			}

			cm.activityMutex.RLock()
			timeSinceActivity := time.Since(cm.lastActivity)
			cm.activityMutex.RUnlock()

			if timeSinceActivity > cm.config.HealthCheckTimeout {
				cm.logger.Warn("No activity for %v, connection may be stale", timeSinceActivity)

				// Optionally send a ping to test connection
				if cm.config.EnableHealthPings {
					if err := cm.SendPing(); err != nil {
						cm.logger.Debug("Health ping failed: %v", err)
						if cm.GetState() != StateStopped {
							cm.handleConnectionError()
						}
						return
					}
				}
			} else {
				cm.logger.Debug("Connection healthy: activity %v ago", timeSinceActivity)
			}
		}
	}
}

func (cm *connectionManager) handleConnectionError() {
	cm.stateMutex.Lock()

	if cm.state == StateStopped {
		cm.stateMutex.Unlock()
		cm.logger.Debug("Ignoring connection error - user commanded disconnect")
		return
	}

	cm.setState(StateDisconnected)
	cm.stateMutex.Unlock()

	cm.logger.Error("WebSocket connection error - transitioning to disconnected state")

	if cm.conn != nil {
		cm.conn.Close()
		cm.conn = nil
	}

	if cm.metrics != nil {
		cm.metrics.IncrementConnectionError()
	}

	if cm.onDisconnect != nil {
		cm.onDisconnect()
	}

	if cm.onError != nil {
		cm.onError(fmt.Errorf("WebSocket connection lost"))
	}
}
