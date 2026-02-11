package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/fx"

	pmwebsocket "github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"github.com/wisp-trading/connectors/pkg/websocket/base"
	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	"github.com/wisp-trading/connectors/pkg/websocket/performance"
	"github.com/wisp-trading/connectors/pkg/websocket/security"
	"github.com/wisp-trading/sdk/pkg/types/logging"
)

// TestWebSocketServer wraps a test websocket server
type TestWebSocketServer struct {
	Server     *httptest.Server
	ServerConn *websocket.Conn
	Upgrader   websocket.Upgrader
	URL        string
}

// NewTestWebSocketServer creates a test websocket server
func NewTestWebSocketServer() *TestWebSocketServer {
	testServer := &TestWebSocketServer{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	testServer.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		testServer.ServerConn, err = testServer.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
	}))

	testServer.URL = "ws" + strings.TrimPrefix(testServer.Server.URL, "http")
	return testServer
}

func (t *TestWebSocketServer) Close() {
	if t.ServerConn != nil {
		t.ServerConn.Close()
	}
	if t.Server != nil {
		t.Server.Close()
	}
}

func (t *TestWebSocketServer) SendMessage(msg []byte) error {
	if t.ServerConn == nil {
		return nil
	}
	return t.ServerConn.WriteMessage(websocket.TextMessage, msg)
}

// testAuthProvider for testing
type testAuthProvider struct{}

func (t *testAuthProvider) GetAuthHeaders(_ context.Context) (http.Header, error) {
	return http.Header{}, nil
}

func (t *testAuthProvider) IsAuthenticated() bool           { return true }
func (t *testAuthProvider) Refresh(_ context.Context) error { return nil }
func (t *testAuthProvider) GetTokenExpiry() time.Time       { return time.Now().Add(24 * time.Hour) }

// TestWebSocketModule provides test dependencies
var TestWebSocketModule = fx.Options(
	fx.Provide(
		NewTestWebSocketServer,
		logging.NewNoOpLogger,
		func(logger logging.ApplicationLogger) security.AuthManager {
			return security.NewAuthManager(&testAuthProvider{}, logger)
		},
		func() security.ValidationConfig {
			return security.ValidationConfig{
				MaxMessageSize: 65536,
				AllowedTypes:   map[string]bool{"book": true, "subscribed": true, "unsubscribed": true},
				TypeField:      "event_type",
			}
		},
		security.NewMessageValidator,
		func() security.RateLimiter {
			return security.NewRateLimiter(1000, 100)
		},
		performance.NewMetrics,
		func() performance.CircuitBreaker {
			return performance.NewCircuitBreaker(3, 30*time.Second)
		},
		func(testServer *TestWebSocketServer) connection.Config {
			connCfg := connection.DefaultConfig()
			connCfg.URL = testServer.URL
			connCfg.EnableHealthMonitoring = false
			connCfg.EnableHealthPings = false
			connCfg.SkipTLSVerify = true
			return connCfg
		},
		connection.NewGorillaDialer,
		connection.NewConnectionManager,
		func() connection.ReconnectionStrategy {
			return connection.NewExponentialBackoffStrategy(5*time.Second, 60*time.Second, 10)
		},
		connection.NewReconnectManager,
		func(testServer *TestWebSocketServer) base.Config {
			return base.Config{
				URL:            testServer.URL,
				MaxMessageSize: 65536,
			}
		},
		base.NewBaseService,
		pmwebsocket.NewWebSocketService,
	),
)
