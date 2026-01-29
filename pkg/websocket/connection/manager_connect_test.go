package connection_test

import (
	"context"
	"errors"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	mockconn "github.com/wisp-trading/connectors/mocks/github.com/wisp-trading/connectors/pkg/websocket/connection"
	mockperf "github.com/wisp-trading/connectors/mocks/github.com/wisp-trading/connectors/pkg/websocket/performance"
	mocksec "github.com/wisp-trading/connectors/mocks/github.com/wisp-trading/connectors/pkg/websocket/security"
	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	logger "github.com/wisp-trading/sdk/pkg/types/logging"
)

var _ = Describe("ConnectionManager - Connect Operations", func() {
	var (
		mgr         connection.ConnectionManager
		mockAuth    *mocksec.AuthManager
		mockMetrics *mockperf.Metrics
		mockDialer  *mockconn.WebSocketDialer
		mockConn    *mockconn.WebSocketConn
		mockLogger  logger.ApplicationLogger
		ctx         context.Context
		cancel      context.CancelFunc
		config      connection.Config
	)

	BeforeEach(func() {
		mockAuth = mocksec.NewAuthManager(GinkgoT())
		mockMetrics = mockperf.NewMetrics(GinkgoT())
		mockDialer = mockconn.NewWebSocketDialer(GinkgoT())
		mockConn = mockconn.NewWebSocketConn(GinkgoT())
		mockLogger = logger.NewNoOpLogger()
		ctx, cancel = context.WithCancel(context.Background())

		mockMetrics.On("GetStats").Return(map[string]interface{}{}).Maybe()
		mockMetrics.On("IncrementConnectionError").Return().Maybe()
		mockMetrics.On("IncrementSent").Return().Maybe()
		mockMetrics.On("IncrementReceived").Return().Maybe()
		mockMetrics.On("RecordConnectionDuration", mock.Anything).Return().Maybe()

		mockDialer.On("DialContext", mock.Anything, mock.Anything, mock.Anything).
			Return(mockConn, (*http.Response)(nil), nil).Maybe()

		mockConn.On("SetReadDeadline", mock.Anything).Return(nil).Maybe()
		mockConn.On("SetWriteDeadline", mock.Anything).Return(nil).Maybe()
		mockConn.On("Close").Return(nil).Maybe()
		mockConn.On("ReadMessage").Return(0, []byte{}, errors.New("connection closed")).Maybe()
		mockConn.On("WriteMessage", mock.Anything, mock.Anything).Return(nil).Maybe()

		config = connection.Config{
			URL:                    "wss://test.example.com/ws",
			ConnectTimeout:         5 * time.Second,
			HandshakeTimeout:       5 * time.Second,
			ReadTimeout:            30 * time.Second,
			WriteTimeout:           10 * time.Second,
			MaxMessageSize:         1024 * 1024,
			ReadBufferSize:         4096,
			WriteBufferSize:        4096,
			HealthCheckInterval:    10 * time.Second,
			HealthCheckTimeout:     30 * time.Second,
			EnableHealthMonitoring: false,
			EnableHealthPings:      false,
		}

		mgr = connection.NewConnectionManager(config, mockAuth, mockMetrics, mockLogger, mockDialer)
	})

	AfterEach(func() {
		cancel()
		if mgr != nil {
			_ = mgr.Disconnect()
		}
	})

	Describe("Connect", func() {
		Context("when auth fails", func() {
			BeforeEach(func() {
				mockAuth.ExpectedCalls = nil
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(nil, errors.New("auth failed"))
			})

			It("should return error and stay Disconnected", func() {
				err := mgr.Connect(ctx, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("auth"))
			})
		})

		Context("when connection succeeds", func() {
			BeforeEach(func() {
				mockAuth.ExpectedCalls = nil
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(http.Header{}, nil)
			})

			It("should transition to Connected state", func() {
				err := mgr.Connect(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(mgr.GetState()).To(Equal(connection.StateConnected))
			})
		})
	})
})
