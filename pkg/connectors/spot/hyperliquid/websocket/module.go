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

// noOpAuthProvider is a no-op implementation for public WebSocket channels
type noOpAuthProvider struct{}

func (n *noOpAuthProvider) GetAuthHeaders(_ context.Context) (http.Header, error) {
	return make(http.Header), nil
}
func (n *noOpAuthProvider) IsAuthenticated() bool        { return true }
func (n *noOpAuthProvider) Refresh(_ context.Context) error { return nil }
func (n *noOpAuthProvider) GetTokenExpiry() time.Time    { return time.Now().Add(24 * time.Hour) }

func newSpotAuthManager(logger logging.ApplicationLogger) security.AuthManager {
	return security.NewAuthManager(&noOpAuthProvider{}, logger)
}

func newSpotValidationConfig() security.ValidationConfig {
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

func newSpotConnectionConfig() connection.Config {
	cfg := connection.DefaultConfig()
	cfg.URL = "wss://api.hyperliquid.xyz/ws"
	cfg.EnableHealthMonitoring = true
	cfg.EnableHealthPings = true
	cfg.HealthCheckInterval = 30 * time.Second
	return cfg
}

func newSpotBaseServiceConfig() base.Config {
	return base.Config{
		URL:            "wss://api.hyperliquid.xyz/ws",
		ReconnectDelay: 5 * time.Second,
		MaxReconnects:  10,
		PingInterval:   30 * time.Second,
		PongTimeout:    10 * time.Second,
		MaxMessageSize: 65536,
	}
}

// WebSocketModule provides all spot WebSocket dependencies.
// Uses "hl_spot_*" named tags to avoid conflicts with the perps module.
var WebSocketModule = fx.Module("hyperliquid_spot_websocket",
	fx.Provide(
		fx.Annotate(
			newSpotAuthManager,
			fx.ResultTags(`name:"hl_spot_auth"`),
		),
		fx.Annotate(
			newSpotValidationConfig,
			fx.ResultTags(`name:"hl_spot_validation"`),
		),
		fx.Annotate(
			func(vc security.ValidationConfig) security.MessageValidator {
				return security.NewMessageValidator(vc)
			},
			fx.ParamTags(`name:"hl_spot_validation"`),
			fx.ResultTags(`name:"hl_spot_validator"`),
		),
		fx.Annotate(
			func() security.RateLimiter { return security.NewRateLimiter(1000, 100) },
			fx.ResultTags(`name:"hl_spot_rate_limiter"`),
		),
		fx.Annotate(
			func() performance.Metrics { return performance.NewMetrics() },
			fx.ResultTags(`name:"hl_spot_metrics"`),
		),
		fx.Annotate(
			func() performance.CircuitBreaker { return performance.NewCircuitBreaker(3, 30*time.Second) },
			fx.ResultTags(`name:"hl_spot_circuit_breaker"`),
		),
		fx.Annotate(
			newSpotConnectionConfig,
			fx.ResultTags(`name:"hl_spot_conn_config"`),
		),
		fx.Annotate(
			connection.NewGorillaDialer,
			fx.ParamTags(`name:"hl_spot_conn_config"`),
			fx.ResultTags(`name:"hl_spot_dialer"`),
		),
		fx.Annotate(
			func(
				cfg connection.Config,
				auth security.AuthManager,
				metrics performance.Metrics,
				logger logging.ApplicationLogger,
				dialer connection.WebSocketDialer,
			) connection.ConnectionManager {
				return connection.NewConnectionManager(cfg, auth, metrics, logger, dialer)
			},
			fx.ParamTags(
				`name:"hl_spot_conn_config"`,
				`name:"hl_spot_auth"`,
				`name:"hl_spot_metrics"`,
				``,
				`name:"hl_spot_dialer"`,
			),
			fx.ResultTags(`name:"hl_spot_conn_manager"`),
		),
		fx.Annotate(
			func() connection.ReconnectionStrategy {
				return connection.NewExponentialBackoffStrategy(5*time.Second, 60*time.Second, 10)
			},
			fx.ResultTags(`name:"hl_spot_reconnect_strategy"`),
		),
		fx.Annotate(
			func(
				cm connection.ConnectionManager,
				strategy connection.ReconnectionStrategy,
				logger logging.ApplicationLogger,
			) connection.ReconnectManager {
				return connection.NewReconnectManager(cm, strategy, logger)
			},
			fx.ParamTags(
				`name:"hl_spot_conn_manager"`,
				`name:"hl_spot_reconnect_strategy"`,
			),
			fx.ResultTags(`name:"hl_spot_reconnect_manager"`),
		),
		fx.Annotate(
			newSpotBaseServiceConfig,
			fx.ResultTags(`name:"hl_spot_base_config"`),
		),
		fx.Annotate(
			func(
				cfg base.Config,
				logger logging.ApplicationLogger,
				validator security.MessageValidator,
				rateLimiter security.RateLimiter,
				metrics performance.Metrics,
				cb performance.CircuitBreaker,
			) base.BaseService {
				return base.NewBaseService(cfg, logger, validator, rateLimiter, metrics, cb)
			},
			fx.ParamTags(
				`name:"hl_spot_base_config"`,
				``,
				`name:"hl_spot_validator"`,
				`name:"hl_spot_rate_limiter"`,
				`name:"hl_spot_metrics"`,
				`name:"hl_spot_circuit_breaker"`,
			),
			fx.ResultTags(`name:"hl_spot_base_service"`),
		),
		fx.Annotate(
			NewSpotRealTimeService,
			fx.ParamTags(
				`name:"hl_spot_conn_manager"`,
				`name:"hl_spot_reconnect_manager"`,
				`name:"hl_spot_base_service"`,
				``,
				``,
			),
		),
	),
)
