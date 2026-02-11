package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/config"
	"github.com/wisp-trading/connectors/pkg/websocket/base"
	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	"github.com/wisp-trading/connectors/pkg/websocket/performance"
	"github.com/wisp-trading/connectors/pkg/websocket/security"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"go.uber.org/fx"
)

const (
	// PolymarketWSURL is the default WebSocket URL for Polymarket CLOB
	PolymarketWSURL = "wss://ws-subscriptions-clob.polymarket.com/ws/market"
)

// AuthProvider implements CLOB authentication for WebSocket connections
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

// newAuthManager creates auth manager for Polymarket CLOB
func newAuthManager(cfg *config.Config, logger logging.ApplicationLogger) security.AuthManager {
	authProvider := &authProvider{
		apiKey:     cfg.APIKey,
		apiSecret:  cfg.APISecret,
		passphrase: cfg.Passphrase,
	}
	return security.NewAuthManager(authProvider, logger)
}

// newValidationConfig creates validation configuration
func newValidationConfig() security.ValidationConfig {
	return security.ValidationConfig{
		MaxMessageSize: 65536,
		AllowedTypes: map[string]bool{
			"market":     true,
			"book":       true,
			"trade":      true,
			"user_order": true,
			"user_trade": true,
			"last_price": true,
			"ticker":     true,
		},
		TypeField: "event_type",
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

// newConnectionConfig creates connection configuration
func newConnectionConfig(cfg *config.Config) connection.Config {
	connCfg := connection.DefaultConfig()
	connCfg.URL = cfg.WebSocketURL
	connCfg.EnableHealthMonitoring = true
	connCfg.EnableHealthPings = true
	connCfg.HealthCheckInterval = 30 * time.Second
	return connCfg
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

// NewReconnectionStrategy creates reconnection strategy
func NewReconnectionStrategy() connection.ReconnectionStrategy {
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
func newBaseServiceConfig(cfg *config.Config) base.Config {
	return base.Config{
		URL:            cfg.WebSocketURL,
		ReconnectDelay: 5 * time.Second,
		MaxReconnects:  10,
		PingInterval:   30 * time.Second,
		PongTimeout:    10 * time.Second,
		MaxMessageSize: 65536,
	}
}

// newBaseService creates base service
func newBaseService(
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

// WebSocketModule provides all real-time WebSocket dependencies for Polymarket
var WebSocketModule = fx.Module("polymarket_websocket",
	fx.Provide(
		fx.Annotate(
			newAuthManager,
			fx.ResultTags(`name:"polymarket_auth_manager"`),
		),
		fx.Annotate(
			newValidationConfig,
			fx.ResultTags(`name:"polymarket_validation"`),
		),
		fx.Annotate(
			newMessageValidator,
			fx.ParamTags(`name:"polymarket_validation"`),
			fx.ResultTags(`name:"polymarket_validator"`),
		),
		fx.Annotate(
			newRateLimiter,
			fx.ResultTags(`name:"polymarket_rate_limiter"`),
		),
		fx.Annotate(
			newMetrics,
			fx.ResultTags(`name:"polymarket_metrics"`),
		),
		fx.Annotate(
			newCircuitBreaker,
			fx.ResultTags(`name:"polymarket_circuit_breaker"`),
		),
		fx.Annotate(
			newConnectionConfig,
			fx.ResultTags(`name:"polymarket_connection_config"`),
		),
		fx.Annotate(
			connection.NewGorillaDialer,
			fx.ParamTags(`name:"polymarket_connection_config"`),
			fx.ResultTags(`name:"polymarket_dialer"`),
		),
		fx.Annotate(
			newConnectionManager,
			fx.ParamTags(
				`name:"polymarket_connection_config"`,
				`name:"polymarket_auth_manager"`,
				`name:"polymarket_metrics"`,
				``,
				`name:"polymarket_dialer"`,
			),
			fx.ResultTags(`name:"polymarket_connection_manager"`),
		),
		fx.Annotate(
			NewReconnectionStrategy,
			fx.ResultTags(`name:"polymarket_strategy"`),
		),
		fx.Annotate(
			newReconnectManager,
			fx.ParamTags(
				`name:"polymarket_connection_manager"`,
				`name:"polymarket_strategy"`,
				``,
			),
			fx.ResultTags(`name:"polymarket_reconnect_manager"`),
		),
		fx.Annotate(
			newBaseServiceConfig,
			fx.ResultTags(`name:"polymarket_base_config"`),
		),
		fx.Annotate(
			newBaseService,
			fx.ParamTags(
				`name:"polymarket_base_config"`,
				``,
				`name:"polymarket_validator"`,
				`name:"polymarket_rate_limiter"`,
				`name:"polymarket_metrics"`,
				`name:"polymarket_circuit_breaker"`,
			),
			fx.ResultTags(`name:"polymarket_base_service"`),
		),
		fx.Annotate(
			NewWebSocketService,
			fx.ParamTags(
				`name:"polymarket_connection_manager"`,
				`name:"polymarket_reconnect_manager"`,
				`name:"polymarket_base_service"`,
				``,
			),
			fx.As(new(PolymarketWebsocket)),
		),
	),
)
