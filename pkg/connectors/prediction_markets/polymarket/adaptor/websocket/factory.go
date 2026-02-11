package websocket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
	"github.com/wisp-trading/connectors/pkg/websocket/base"
	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	"github.com/wisp-trading/connectors/pkg/websocket/performance"
	"github.com/wisp-trading/connectors/pkg/websocket/security"
	"github.com/wisp-trading/sdk/pkg/types/logging"
)

// WebSocketServiceFactory creates WebSocket services with runtime configuration
type WebSocketServiceFactory interface {
	CreateWebSocketService(cfg *config.Config) (PolymarketWebsocket, error)
}

type webSocketServiceFactory struct {
	logger logging.ApplicationLogger
}

// NewWebSocketServiceFactory creates a new factory for creating WebSocket services
// This is injected by fx at build time (no config needed)
func NewWebSocketServiceFactory(logger logging.ApplicationLogger) WebSocketServiceFactory {
	return &webSocketServiceFactory{
		logger: logger,
	}
}

// CreateWebSocketService creates a fully-configured WebSocket service with the given config
// This is called at runtime from connector.Initialize() after config is available
func (f *webSocketServiceFactory) CreateWebSocketService(cfg *config.Config) (PolymarketWebsocket, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	f.logger.Info("Creating Polymarket WebSocket service with config")

	// Create auth provider with credentials from config
	authProvider := &authProvider{
		apiKey:     cfg.APIKey,
		apiSecret:  cfg.APISecret,
		passphrase: cfg.Passphrase,
	}

	// Create auth manager
	authManager := security.NewAuthManager(authProvider, f.logger)

	// Create message validator
	validator := NewMessageValidator(DefaultValidationConfig())

	// Create rate limiter
	rateLimiter := security.NewRateLimiter(1000, 100)

	// Create metrics
	metrics := performance.NewMetrics()

	// Create circuit breaker
	circuitBreaker := performance.NewCircuitBreaker(3, 30*time.Second)

	// Create connection config with URL from config
	connConfig := connection.DefaultConfig()
	connConfig.URL = cfg.WebSocketURL
	connConfig.EnableHealthMonitoring = true
	connConfig.EnableHealthPings = true
	connConfig.HealthCheckInterval = 30 * time.Second

	// Create dialer
	dialer := connection.NewGorillaDialer(connConfig)

	// Create connection manager
	connManager := connection.NewConnectionManager(
		connConfig,
		authManager,
		metrics,
		f.logger,
		dialer,
	)

	// Create reconnection strategy
	reconnectStrategy := connection.NewExponentialBackoffStrategy(
		5*time.Second,
		60*time.Second,
		10,
	)

	// Create reconnect manager
	reconnectMgr := connection.NewReconnectManager(
		connManager,
		reconnectStrategy,
		f.logger,
	)

	// Create base service config
	baseConfig := base.Config{
		URL:            cfg.WebSocketURL,
		ReconnectDelay: 5 * time.Second,
		MaxReconnects:  10,
		PingInterval:   30 * time.Second,
		PongTimeout:    10 * time.Second,
		MaxMessageSize: 65536,
	}

	// Create base service
	baseService := base.NewBaseService(
		baseConfig,
		f.logger,
		validator,
		rateLimiter,
		metrics,
		circuitBreaker,
	)

	// Create and return WebSocket service
	ws := NewWebSocketService(
		connManager,
		reconnectMgr,
		baseService,
		f.logger,
	)

	f.logger.Info("✅ Polymarket WebSocket service created successfully")

	return ws, nil
}

// authProvider implements CLOB authentication for WebSocket connections
type authProvider struct {
	apiKey     string
	apiSecret  string
	passphrase string
}

func (a *authProvider) GetAuthHeaders(_ context.Context) (http.Header, error) {
	headers := make(http.Header)
	headers.Set("POLY_API_KEY", a.apiKey)
	headers.Set("POLY_SECRET", a.apiSecret)
	headers.Set("POLY_PASSPHRASE", a.passphrase)
	return headers, nil
}

func (a *authProvider) IsAuthenticated() bool {
	return a.apiKey != "" && a.apiSecret != "" && a.passphrase != ""
}

func (a *authProvider) Refresh(_ context.Context) error {
	// CLOB API keys don't expire, no refresh needed
	return nil
}

func (a *authProvider) GetTokenExpiry() time.Time {
	// Return far future date since keys don't expire
	return time.Now().Add(365 * 24 * time.Hour)
}
