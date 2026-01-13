package connection_test

import (
	"context"
	"errors"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	logger "github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	mockconn "github.com/backtesting-org/live-trading/mocks/github.com/backtesting-org/live-trading/pkg/websocket/connection"
	mockperf "github.com/backtesting-org/live-trading/mocks/github.com/backtesting-org/live-trading/pkg/websocket/performance"
	mocksec "github.com/backtesting-org/live-trading/mocks/github.com/backtesting-org/live-trading/pkg/websocket/security"
	"github.com/backtesting-org/live-trading/pkg/websocket/connection"
)

var _ = Describe("ConnectionManager - Send Operations", func() {
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

	Describe("SendMessage", func() {
		Context("when disconnected", func() {
			It("should return error", func() {
				err := mgr.SendMessage([]byte("test"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not connected"))
			})
		})

		Context("when in Stopped state", func() {
			It("should return error", func() {
				mgr.Disconnect()
				Expect(mgr.GetState()).To(Equal(connection.StateStopped))

				err := mgr.SendMessage([]byte("test"))
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when connected", func() {
			BeforeEach(func() {
				mockAuth.ExpectedCalls = nil
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(http.Header{}, nil)
				err := mgr.Connect(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should send message successfully", func() {
				err := mgr.SendMessage([]byte("test message"))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("SendJSON", func() {
		Context("when disconnected", func() {
			It("should return error", func() {
				testData := map[string]string{"test": "data"}
				err := mgr.SendJSON(testData)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not connected"))
			})
		})

		Context("when connected", func() {
			BeforeEach(func() {
				mockAuth.ExpectedCalls = nil
				mockAuth.On("GetSecureHeaders", mock.Anything).Return(http.Header{}, nil)
				err := mgr.Connect(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should serialize and send JSON", func() {
				testData := map[string]string{"test": "data"}
				err := mgr.SendJSON(testData)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("SendPing", func() {
		Context("when disconnected", func() {
			It("should return error", func() {
				err := mgr.SendPing()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
