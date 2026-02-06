package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/wisp-trading/connectors/pkg/websocket/base"
	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	"github.com/wisp-trading/connectors/pkg/websocket/performance"
	"github.com/wisp-trading/connectors/pkg/websocket/security"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"go.uber.org/fx"
)

const (
	// BybitPerpWSURL is the WebSocket URL for Bybit Perpetual (Public)
	BybitPerpWSURL = "wss://stream.bybit.com/v5/public/linear"
	// BybitPerpPrivateWSURL is the WebSocket URL for Bybit Perpetual (Private)
	BybitPerpPrivateWSURL = "wss://stream.bybit.com/v5/private"
)

// noOpAuthProvider is a no-op implementation for public WebSocket channels
// TODO: Implement proper Bybit auth when adding private channels
type noOpAuthProvider struct{}

func (n *noOpAuthProvider) GetAuthHeaders(_ context.Context) (http.Header, error) {
	return make(http.Header), nil
}

func (n *noOpAuthProvider) IsAuthenticated() bool {
	return true
}

func (n *noOpAuthProvider) Refresh(_ context.Context) error {
	return nil
}

func (n *noOpAuthProvider) GetTokenExpiry() time.Time {
	return time.Now().Add(24 * time.Hour)
}

// NewAuthManager creates auth manager (no-op for now, TODO: implement Bybit auth)
func NewAuthManager(logger logging.ApplicationLogger) security.AuthManager {
	authProvider := &noOpAuthProvider{}
	return security.NewAuthManager(authProvider, logger)
}

// NewValidationConfig creates validation configuration for Bybit messages
func NewValidationConfig() security.ValidationConfig {
	return security.ValidationConfig{
		MaxMessageSize: 131072, // Bybit can send larger messages (128KB)
		AllowedTypes: map[string]bool{
			"snapshot":    true, // orderbook snapshot
			"delta":       true, // orderbook delta
			"publicTrade": true, // trades
			"kline":       true, // klines
			"position":    true, // positions (private)
			"wallet":      true, // wallet/balance (private)
		},
		TypeField: "topic", // Bybit uses "topic" field
	}
}

// NewMessageValidator creates message validator
func NewMessageValidator(valConfig security.ValidationConfig) security.MessageValidator {
	return security.NewMessageValidator(valConfig)
}

// NewRateLimiter creates rate limiter (Bybit allows ~10 messages/sec)
func NewRateLimiter() security.RateLimiter {
	return security.NewRateLimiter(600, 60) // 600 per minute = 10/sec
}

// NewMetrics creates metrics instance
func NewMetrics() performance.Metrics {
	return performance.NewMetrics()
}

// NewCircuitBreaker creates circuit breaker
func NewCircuitBreaker() performance.CircuitBreaker {
	return performance.NewCircuitBreaker(5, 30*time.Second)
}

// NewConnectionConfig creates connection configuration
func NewConnectionConfig() connection.Config {
	cfg := connection.DefaultConfig()
	cfg.URL = BybitPerpWSURL
	cfg.EnableHealthMonitoring = true
	cfg.EnableHealthPings = true
	cfg.HealthCheckInterval = 20 * time.Second // Bybit recommends 20s ping interval
	return cfg
}

// NewConnectionManager creates connection manager
func NewConnectionManager(
	config connection.Config,
	authManager security.AuthManager,
	metrics performance.Metrics,
	logger logging.ApplicationLogger,
	dialer connection.WebSocketDialer,
) connection.ConnectionManager {
	return connection.NewConnectionManager(config, authManager, metrics, logger, dialer)
}

// NewReconnectionStrategy creates reconnection strategy
func NewReconnectionStrategy() connection.ReconnectionStrategy {
	return connection.NewExponentialBackoffStrategy(
		3*time.Second,  // Initial delay
		60*time.Second, // Max delay
		10,             // Max attempts
	)
}

// NewReconnectManager creates reconnect manager
func NewReconnectManager(
	connManager connection.ConnectionManager,
	strategy connection.ReconnectionStrategy,
	logger logging.ApplicationLogger,
) connection.ReconnectManager {
	return connection.NewReconnectManager(connManager, strategy, logger)
}

// NewBaseServiceConfig creates base service configuration
func NewBaseServiceConfig() base.Config {
	return base.Config{
		URL:            BybitPerpWSURL,
		ReconnectDelay: 3 * time.Second,
		MaxReconnects:  10,
		PingInterval:   20 * time.Second, // Bybit recommends 20s
		PongTimeout:    10 * time.Second,
		MaxMessageSize: 131072, // 128KB
	}
}

// NewBaseService creates base service
func NewBaseService(
	config base.Config,
	logger logging.ApplicationLogger,
	validator security.MessageValidator,
	rateLimiter security.RateLimiter,
	metrics performance.Metrics,
	circuitBreaker performance.CircuitBreaker,
) base.BaseService {
	return base.NewBaseService(
		config,
		logger,
		validator,
		rateLimiter,
		metrics,
		circuitBreaker,
	)
}

// WebSocketModule provides all real-time WebSocket dependencies for Bybit
var WebSocketModule = fx.Module("bybit_perp_websocket",
	fx.Provide(
		fx.Annotate(
			NewAuthManager,
			fx.ResultTags(`name:"bybit_perp_auth_manager"`),
		),
		fx.Annotate(
			NewValidationConfig,
			fx.ResultTags(`name:"bybit_perp_validation"`),
		),
		fx.Annotate(
			NewMessageValidator,
			fx.ParamTags(`name:"bybit_perp_validation"`),
			fx.ResultTags(`name:"bybit_perp_validator"`),
		),
		fx.Annotate(
			NewRateLimiter,
			fx.ResultTags(`name:"bybit_perp_rate_limiter"`),
		),
		fx.Annotate(
			NewMetrics,
			fx.ResultTags(`name:"bybit_perp_metrics"`),
		),
		fx.Annotate(
			NewCircuitBreaker,
			fx.ResultTags(`name:"bybit_perp_circuit_breaker"`),
		),
		fx.Annotate(
			NewConnectionConfig,
			fx.ResultTags(`name:"bybit_perp_connection_config"`),
		),
		fx.Annotate(
			connection.NewGorillaDialer,
			fx.ParamTags(`name:"bybit_perp_connection_config"`),
			fx.ResultTags(`name:"bybit_perp_dialer"`),
		),
		fx.Annotate(
			NewConnectionManager,
			fx.ParamTags(
				`name:"bybit_perp_connection_config"`,
				`name:"bybit_perp_auth_manager"`,
				`name:"bybit_perp_metrics"`,
				``,
				`name:"bybit_perp_dialer"`,
			),
			fx.ResultTags(`name:"bybit_perp_connection_manager"`),
		),
		fx.Annotate(
			NewReconnectionStrategy,
			fx.ResultTags(`name:"bybit_perp_strategy"`),
		),
		fx.Annotate(
			NewReconnectManager,
			fx.ParamTags(
				`name:"bybit_perp_connection_manager"`,
				`name:"bybit_perp_strategy"`,
				``,
			),
			fx.ResultTags(`name:"bybit_perp_reconnect_manager"`),
		),
		fx.Annotate(
			NewBaseServiceConfig,
			fx.ResultTags(`name:"bybit_perp_base_config"`),
		),
		fx.Annotate(
			NewBaseService,
			fx.ParamTags(
				`name:"bybit_perp_base_config"`,
				``,
				`name:"bybit_perp_validator"`,
				`name:"bybit_perp_rate_limiter"`,
				`name:"bybit_perp_metrics"`,
				`name:"bybit_perp_circuit_breaker"`,
			),
			fx.ResultTags(`name:"bybit_perp_base_service"`),
		),
		fx.Annotate(
			NewWebSocketService,
			fx.ParamTags(
				`name:"bybit_perp_connection_manager"`,
				`name:"bybit_perp_reconnect_manager"`,
				`name:"bybit_perp_base_service"`,
				``,
			),
			fx.As(new(RealTimeService)),
		),
	),
)
