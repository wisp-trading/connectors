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
	// GateSpotWSURL is the WebSocket URL for Gate.io Spot
	GateSpotWSURL = "wss://api.gateio.ws/ws/v4/"
)

// noOpAuthProvider is a no-op implementation for public WebSocket channels
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

// NewAuthManager creates auth manager (no-op for public channels)
func NewAuthManager(logger logging.ApplicationLogger) security.AuthManager {
	authProvider := &noOpAuthProvider{}
	return security.NewAuthManager(authProvider, logger)
}

// NewValidationConfig creates validation configuration
func NewValidationConfig() security.ValidationConfig {
	return security.ValidationConfig{
		MaxMessageSize: 65536,
		AllowedTypes: map[string]bool{
			"spot.order_book":   true,
			"spot.trades":       true,
			"spot.candlesticks": true,
			"spot.balances":     true,
			"spot.orders":       true,
		},
		TypeField: "channel",
	}
}

// NewMessageValidator creates message validator
func NewMessageValidator(valConfig security.ValidationConfig) security.MessageValidator {
	return security.NewMessageValidator(valConfig)
}

// NewRateLimiter creates rate limiter
func NewRateLimiter() security.RateLimiter {
	return security.NewRateLimiter(1000, 100)
}

// NewMetrics creates metrics instance
func NewMetrics() performance.Metrics {
	return performance.NewMetrics()
}

// NewCircuitBreaker creates circuit breaker
func NewCircuitBreaker() performance.CircuitBreaker {
	return performance.NewCircuitBreaker(3, 30*time.Second)
}

// NewConnectionConfig creates connection configuration
func NewConnectionConfig() connection.Config {
	cfg := connection.DefaultConfig()
	cfg.URL = "wss://api.gateio.ws/ws/v4/"
	cfg.EnableHealthMonitoring = true
	cfg.EnableHealthPings = true
	cfg.HealthCheckInterval = 30 * time.Second
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
		5*time.Second,
		60*time.Second,
		10,
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
		URL:            GateSpotWSURL,
		ReconnectDelay: 5 * time.Second,
		MaxReconnects:  10,
		PingInterval:   30 * time.Second,
		PongTimeout:    10 * time.Second,
		MaxMessageSize: 65536,
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

// WebSocketModule provides all real-time WebSocket dependencies
var WebSocketModule = fx.Module("gate_spot_websocket",
	fx.Provide(
		fx.Annotate(
			NewAuthManager,
			fx.ResultTags(`name:"gate_spot_auth_manager"`),
		),
		fx.Annotate(
			NewValidationConfig,
			fx.ResultTags(`name:"gate_spot_validation"`),
		),
		fx.Annotate(
			NewMessageValidator,
			fx.ParamTags(`name:"gate_spot_validation"`),
			fx.ResultTags(`name:"gate_spot_validator"`),
		),
		fx.Annotate(
			NewRateLimiter,
			fx.ResultTags(`name:"gate_spot_rate_limiter"`),
		),
		fx.Annotate(
			NewMetrics,
			fx.ResultTags(`name:"gate_spot_metrics"`),
		),
		fx.Annotate(
			NewCircuitBreaker,
			fx.ResultTags(`name:"gate_spot_circuit_breaker"`),
		),
		fx.Annotate(
			NewConnectionConfig,
			fx.ResultTags(`name:"gate_spot_connection_config"`),
		),
		fx.Annotate(
			connection.NewGorillaDialer,
			fx.ParamTags(`name:"gate_spot_connection_config"`),
			fx.ResultTags(`name:"gate_spot_dialer"`),
		),
		fx.Annotate(
			NewConnectionManager,
			fx.ParamTags(
				`name:"gate_spot_connection_config"`,
				`name:"gate_spot_auth_manager"`,
				`name:"gate_spot_metrics"`,
				``,
				`name:"gate_spot_dialer"`,
			),
			fx.ResultTags(`name:"gate_spot_connection_manager"`),
		),
		fx.Annotate(
			NewReconnectionStrategy,
			fx.ResultTags(`name:"gate_spot_strategy"`),
		),
		fx.Annotate(
			NewReconnectManager,
			fx.ParamTags(
				`name:"gate_spot_connection_manager"`,
				`name:"gate_spot_strategy"`,
				``,
			),
			fx.ResultTags(`name:"gate_spot_reconnect_manager"`),
		),
		fx.Annotate(
			NewBaseServiceConfig,
			fx.ResultTags(`name:"gate_spot_base_config"`),
		),
		fx.Annotate(
			NewBaseService,
			fx.ParamTags(
				`name:"gate_spot_base_config"`,
				``,
				`name:"gate_spot_validator"`,
				`name:"gate_spot_rate_limiter"`,
				`name:"gate_spot_metrics"`,
				`name:"gate_spot_circuit_breaker"`,
			),
			fx.ResultTags(`name:"gate_spot_base_service"`),
		),
		fx.Annotate(
			NewWebSocketService,
			fx.ParamTags(
				`name:"gate_spot_connection_manager"`,
				`name:"gate_spot_reconnect_manager"`,
				`name:"gate_spot_base_service"`,
				``,
			),
			fx.As(new(RealTimeService)),
		),
	),
)
