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

// noOpAuthProvider satisfies security.AuthProvider for Deribit.
// Deribit does not use HTTP-level auth headers — authentication is performed
// at the application level via a public/auth JSON-RPC message after connect.
type noOpAuthProvider struct{}

func (n *noOpAuthProvider) GetAuthHeaders(_ context.Context) (http.Header, error) {
	return make(http.Header), nil
}
func (n *noOpAuthProvider) IsAuthenticated() bool        { return true }
func (n *noOpAuthProvider) Refresh(_ context.Context) error { return nil }
func (n *noOpAuthProvider) GetTokenExpiry() time.Time    { return time.Now().Add(24 * time.Hour) }

func newAuthManager(logger logging.ApplicationLogger) security.AuthManager {
	return security.NewAuthManager(&noOpAuthProvider{}, logger)
}

func newValidationConfig() security.ValidationConfig {
	return security.ValidationConfig{
		MaxMessageSize: 131072, // 128 KB — Deribit messages are small but safe upper bound
		AllowedTypes: map[string]bool{
			"subscription": true,
			"heartbeat":    true,
		},
		TypeField: "method",
	}
}

func newMessageValidator(cfg security.ValidationConfig) security.MessageValidator {
	return security.NewMessageValidator(cfg)
}

func newRateLimiter() security.RateLimiter {
	// Deribit allows up to 20 requests/second on WebSocket
	return security.NewRateLimiter(1200, 60)
}

func newMetrics() performance.Metrics {
	return performance.NewMetrics()
}

func newCircuitBreaker() performance.CircuitBreaker {
	return performance.NewCircuitBreaker(5, 30*time.Second)
}

func newConnectionConfig() connection.Config {
	cfg := connection.DefaultConfig()
	// URL is set at runtime via Connect() — the config only carries defaults.
	cfg.EnableHealthMonitoring = true
	cfg.EnableHealthPings = true
	cfg.HealthCheckInterval = 25 * time.Second // Deribit requires heartbeat < 30s
	return cfg
}

func newConnectionManager(
	config connection.Config,
	authManager security.AuthManager,
	metrics performance.Metrics,
	logger logging.ApplicationLogger,
	dialer connection.WebSocketDialer,
) connection.ConnectionManager {
	return connection.NewConnectionManager(config, authManager, metrics, logger, dialer)
}

func newReconnectionStrategy() connection.ReconnectionStrategy {
	return connection.NewExponentialBackoffStrategy(
		3*time.Second,  // initial delay
		60*time.Second, // max delay
		10,             // max attempts
	)
}

func newReconnectManager(
	connManager connection.ConnectionManager,
	strategy connection.ReconnectionStrategy,
	logger logging.ApplicationLogger,
) connection.ReconnectManager {
	return connection.NewReconnectManager(connManager, strategy, logger)
}

func newBaseServiceConfig() base.Config {
	return base.Config{
		ReconnectDelay: 3 * time.Second,
		MaxReconnects:  10,
		PingInterval:   25 * time.Second,
		PongTimeout:    10 * time.Second,
		MaxMessageSize: 131072,
	}
}

func newBaseService(
	config base.Config,
	logger logging.ApplicationLogger,
	validator security.MessageValidator,
	rateLimiter security.RateLimiter,
	metrics performance.Metrics,
	circuitBreaker performance.CircuitBreaker,
) base.BaseService {
	return base.NewBaseService(config, logger, validator, rateLimiter, metrics, circuitBreaker)
}

// WebSocketModule provides all WebSocket dependencies for the Deribit options connector.
var WebSocketModule = fx.Module("deribit_options_websocket",
	fx.Provide(
		fx.Annotate(newAuthManager,
			fx.ResultTags(`name:"deribit_options_auth_manager"`),
		),
		fx.Annotate(newValidationConfig,
			fx.ResultTags(`name:"deribit_options_validation"`),
		),
		fx.Annotate(newMessageValidator,
			fx.ParamTags(`name:"deribit_options_validation"`),
			fx.ResultTags(`name:"deribit_options_validator"`),
		),
		fx.Annotate(newRateLimiter,
			fx.ResultTags(`name:"deribit_options_rate_limiter"`),
		),
		fx.Annotate(newMetrics,
			fx.ResultTags(`name:"deribit_options_metrics"`),
		),
		fx.Annotate(newCircuitBreaker,
			fx.ResultTags(`name:"deribit_options_circuit_breaker"`),
		),
		fx.Annotate(newConnectionConfig,
			fx.ResultTags(`name:"deribit_options_connection_config"`),
		),
		fx.Annotate(connection.NewGorillaDialer,
			fx.ParamTags(`name:"deribit_options_connection_config"`),
			fx.ResultTags(`name:"deribit_options_dialer"`),
		),
		fx.Annotate(newConnectionManager,
			fx.ParamTags(
				`name:"deribit_options_connection_config"`,
				`name:"deribit_options_auth_manager"`,
				`name:"deribit_options_metrics"`,
				``,
				`name:"deribit_options_dialer"`,
			),
			fx.ResultTags(`name:"deribit_options_connection_manager"`),
		),
		fx.Annotate(newReconnectionStrategy,
			fx.ResultTags(`name:"deribit_options_strategy"`),
		),
		fx.Annotate(newReconnectManager,
			fx.ParamTags(
				`name:"deribit_options_connection_manager"`,
				`name:"deribit_options_strategy"`,
				``,
			),
			fx.ResultTags(`name:"deribit_options_reconnect_manager"`),
		),
		fx.Annotate(newBaseServiceConfig,
			fx.ResultTags(`name:"deribit_options_base_config"`),
		),
		fx.Annotate(newBaseService,
			fx.ParamTags(
				`name:"deribit_options_base_config"`,
				``,
				`name:"deribit_options_validator"`,
				`name:"deribit_options_rate_limiter"`,
				`name:"deribit_options_metrics"`,
				`name:"deribit_options_circuit_breaker"`,
			),
			fx.ResultTags(`name:"deribit_options_base_service"`),
		),
		fx.Annotate(NewService,
			fx.ParamTags(
				`name:"deribit_options_connection_manager"`,
				`name:"deribit_options_reconnect_manager"`,
				`name:"deribit_options_base_service"`,
				``,
			),
		),
	),
)
