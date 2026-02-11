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
	"github.com/wisp-trading/sdk/pkg/types/temporal"
	"go.uber.org/fx"
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

// newAuthManager creates auth manager (no-op for public channels)
func newAuthManager(logger logging.ApplicationLogger) security.AuthManager {
	authProvider := &noOpAuthProvider{}
	return security.NewAuthManager(authProvider, logger)
}

// newValidationConfig creates validation configuration
func newValidationConfig() security.ValidationConfig {
	return security.ValidationConfig{
		MaxMessageSize: 65536,
		AllowedTypes: map[string]bool{
			"l2Book":   true,
			"trades":   true,
			"candle":   true,
			"webData2": true,
		},
		TypeField: "channel",
	}
}

// newMessageValidator creates message validator
func newMessageValidator(valConfig security.ValidationConfig) security.MessageValidator {
	return security.NewMessageValidator(valConfig)
}

// newRateLimiter creates rate limiter
func newRateLimiter() security.RateLimiter {
	return security.NewRateLimiter(1000, 100)
}

// newMetrics creates metrics instance
func newMetrics() performance.Metrics {
	return performance.NewMetrics()
}

// newCircuitBreaker creates circuit breaker
func newCircuitBreaker() performance.CircuitBreaker {
	return performance.NewCircuitBreaker(3, 30*time.Second)
}

// newMessageParser creates message parser
func newMessageParser(
	logger logging.ApplicationLogger,
	timeProvider temporal.TimeProvider,
) MessageParser {
	return NewParser(logger, timeProvider)
}

// newConnectionConfig creates connection configuration
func newConnectionConfig() connection.Config {
	cfg := connection.DefaultConfig()
	cfg.URL = "wss://api.hyperliquid.xyz/ws"
	cfg.EnableHealthMonitoring = true
	cfg.EnableHealthPings = true
	cfg.HealthCheckInterval = 30 * time.Second
	return cfg
}

// newConnectionManager creates connection manager
func newConnectionManager(
	config connection.Config,
	authManager security.AuthManager,
	metrics performance.Metrics,
	logger logging.ApplicationLogger,
	dialer connection.WebSocketDialer,
) connection.ConnectionManager {
	return connection.NewConnectionManager(config, authManager, metrics, logger, dialer)
}

// newReconnectionStrategy creates reconnection strategy
func newReconnectionStrategy() connection.ReconnectionStrategy {
	return connection.NewExponentialBackoffStrategy(
		5*time.Second,
		60*time.Second,
		10,
	)
}

// newReconnectManager creates reconnect manager
func newReconnectManager(
	connManager connection.ConnectionManager,
	strategy connection.ReconnectionStrategy,
	logger logging.ApplicationLogger,
) connection.ReconnectManager {
	return connection.NewReconnectManager(connManager, strategy, logger)
}

// newBaseServiceConfig creates base service configuration
func newBaseServiceConfig() base.Config {
	return base.Config{
		URL:            "wss://api.hyperliquid.xyz/ws",
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
var WebSocketModule = fx.Module("hyperliquid_websocket",
	fx.Provide(
		fx.Annotate(
			newAuthManager,
			fx.ResultTags(`name:"hyperliquid_auth_manager"`),
		),
		fx.Annotate(
			newValidationConfig,
			fx.ResultTags(`name:"hyperliquid_validation"`),
		),
		fx.Annotate(
			newMessageValidator,
			fx.ParamTags(`name:"hyperliquid_validation"`),
			fx.ResultTags(`name:"hyperliquid_validator"`),
		),
		fx.Annotate(
			newRateLimiter,
			fx.ResultTags(`name:"hyperliquid_rate_limiter"`),
		),
		fx.Annotate(
			newMetrics,
			fx.ResultTags(`name:"hyperliquid_metrics"`),
		),
		fx.Annotate(
			newCircuitBreaker,
			fx.ResultTags(`name:"hyperliquid_circuit_breaker"`),
		),
		fx.Annotate(
			newMessageParser,
			fx.ResultTags(`name:"hyperliquid_parser"`),
		),
		fx.Annotate(
			newConnectionConfig,
			fx.ResultTags(`name:"hyperliquid_connection_config"`),
		),
		fx.Annotate(
			connection.NewGorillaDialer, // Provide dialer
			fx.ParamTags(`name:"hyperliquid_connection_config"`),
			fx.ResultTags(`name:"hyperliquid_dialer"`),
		),
		fx.Annotate(
			newConnectionManager,
			fx.ParamTags(
				`name:"hyperliquid_connection_config"`,
				`name:"hyperliquid_auth_manager"`,
				`name:"hyperliquid_metrics"`,
				``,                          // logger (no tag)
				`name:"hyperliquid_dialer"`, // Inject dialer
			),
			fx.ResultTags(`name:"hyperliquid_connection_manager"`),
		),
		fx.Annotate(
			newReconnectionStrategy,
			fx.ResultTags(`name:"hyperliquid_strategy"`),
		),
		fx.Annotate(
			newReconnectManager,
			fx.ParamTags(
				`name:"hyperliquid_connection_manager"`,
				`name:"hyperliquid_strategy"`,
			),
			fx.ResultTags(`name:"hyperliquid_reconnect_manager"`),
		),
		fx.Annotate(
			newBaseServiceConfig,
			fx.ResultTags(`name:"hyperliquid_base_config"`),
		),
		fx.Annotate(
			NewBaseService,
			fx.ParamTags(
				`name:"hyperliquid_base_config"`,
				``,
				`name:"hyperliquid_validator"`,
				`name:"hyperliquid_rate_limiter"`,
				`name:"hyperliquid_metrics"`,
				`name:"hyperliquid_circuit_breaker"`,
			),
			fx.ResultTags(`name:"hyperliquid_base"`),
		),
		fx.Annotate(
			NewWebSocketService,
			fx.ParamTags(
				`name:"hyperliquid_connection_manager"`,
				`name:"hyperliquid_reconnect_manager"`,
				`name:"hyperliquid_base"`,
				``,
				`name:"hyperliquid_parser"`,
			),
		),
	),
)
