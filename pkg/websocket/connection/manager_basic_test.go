package connection_test

import (
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

var _ = Describe("ConnectionManager - Basic Operations", func() {
	var (
		mgr         connection.ConnectionManager
		mockAuth    *mocksec.AuthManager
		mockMetrics *mockperf.Metrics
		mockDialer  *mockconn.WebSocketDialer
		mockConn    *mockconn.WebSocketConn
		mockLogger  logger.ApplicationLogger
		config      connection.Config
	)

	BeforeEach(func() {
		mockAuth = mocksec.NewAuthManager(GinkgoT())
		mockMetrics = mockperf.NewMetrics(GinkgoT())
		mockDialer = mockconn.NewWebSocketDialer(GinkgoT())
		mockConn = mockconn.NewWebSocketConn(GinkgoT())
		mockLogger = logger.NewNoOpLogger()

		// Setup mock metrics
		mockMetrics.On("GetStats").Return(map[string]interface{}{}).Maybe()
		mockMetrics.On("IncrementConnectionError").Return().Maybe()
		mockMetrics.On("IncrementSent").Return().Maybe()
		mockMetrics.On("IncrementReceived").Return().Maybe()
		mockMetrics.On("RecordConnectionDuration", mock.Anything).Return().Maybe()

		// Setup mock dialer
		mockDialer.On("DialContext", mock.Anything, mock.Anything, mock.Anything).
			Return(mockConn, (*http.Response)(nil), nil).Maybe()

		// Setup mock connection
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
		if mgr != nil {
			_ = mgr.Disconnect()
		}
	})

	Describe("Initial State", func() {
		It("should start in Disconnected state", func() {
			Expect(mgr.GetState()).To(Equal(connection.StateDisconnected))
		})
	})

	Describe("Disconnect - User Command", func() {
		It("should transition to Stopped state", func() {
			err := mgr.Disconnect()
			Expect(err).ToNot(HaveOccurred())
			Expect(mgr.GetState()).To(Equal(connection.StateStopped))
		})

		It("should be idempotent", func() {
			err := mgr.Disconnect()
			Expect(err).ToNot(HaveOccurred())

			err = mgr.Disconnect()
			Expect(err).ToNot(HaveOccurred())

			Expect(mgr.GetState()).To(Equal(connection.StateStopped))
		})

		It("should not call onDisconnect callback", func() {
			disconnectCalled := false
			mgr.SetCallbacks(nil, func() error {
				disconnectCalled = true
				return nil
			}, nil, nil)

			err := mgr.Disconnect()
			Expect(err).ToNot(HaveOccurred())

			Consistently(func() bool { return disconnectCalled }, "200ms").Should(BeFalse())
		})

		It("should succeed when already disconnected", func() {
			Expect(mgr.GetState()).To(Equal(connection.StateDisconnected))

			err := mgr.Disconnect()
			Expect(err).ToNot(HaveOccurred())
			Expect(mgr.GetState()).To(Equal(connection.StateStopped))
		})
	})

	Describe("GetConnectionStats", func() {
		It("should return stats map", func() {
			stats := mgr.GetConnectionStats()
			Expect(stats).ToNot(BeNil())
			Expect(stats).To(HaveKey("state"))
			Expect(stats).To(HaveKey("connected"))
			Expect(stats).To(HaveKey("url"))
		})

		It("should reflect current state", func() {
			stats := mgr.GetConnectionStats()
			Expect(stats["state"]).To(Equal("disconnected"))
			Expect(stats["connected"]).To(BeFalse())

			mgr.Disconnect()
			stats = mgr.GetConnectionStats()
			Expect(stats["state"]).To(Equal("stopped"))
		})
	})

	Describe("IsHealthy", func() {
		It("should return false when disconnected", func() {
			Expect(mgr.IsHealthy()).To(BeFalse())
		})

		It("should return false when stopped", func() {
			mgr.Disconnect()
			Expect(mgr.IsHealthy()).To(BeFalse())
		})
	})
})
