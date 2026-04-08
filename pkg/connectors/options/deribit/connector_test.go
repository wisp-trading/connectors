package deribit

import (
	"context"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/options/deribit/adaptor"
	deribitWS "github.com/wisp-trading/connectors/pkg/connectors/options/deribit/websocket"
	"github.com/wisp-trading/sdk/pkg/types/logging"
	"github.com/wisp-trading/sdk/pkg/types/temporal"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// noOpWSService is a no-op WebSocket service for unit tests that don't exercise WebSocket.
type noOpWSService struct{}

func (n *noOpWSService) Connect(_ context.Context, _, _, _ string) error { return nil }
func (n *noOpWSService) Disconnect() error                               { return nil }
func (n *noOpWSService) IsConnected() bool                               { return false }
func (n *noOpWSService) ErrorChannel() <-chan error                      { return make(chan error) }
func (n *noOpWSService) SubscribeToTicker(_ string, _ func(*deribitWS.TickerData)) error {
	return nil
}
func (n *noOpWSService) UnsubscribeFromTicker(_ string) error { return nil }
func (n *noOpWSService) SubscribeToOrderBook(_ string, _ func(*deribitWS.OrderBookData)) error {
	return nil
}
func (n *noOpWSService) UnsubscribeFromOrderBook(_ string) error { return nil }

var _ deribitWS.Service = (*noOpWSService)(nil)

// noOpTradingLogger implements TradingLogger with no-op methods
type noOpTradingLogger struct{}

func (l *noOpTradingLogger) Info(format string, args ...interface{})                      {}
func (l *noOpTradingLogger) Infof(msg string, args ...interface{})                        {}
func (l *noOpTradingLogger) MarketCondition(msg string, args ...interface{})              {}
func (l *noOpTradingLogger) Opportunity(strategy, asset, msg string, args ...interface{}) {}
func (l *noOpTradingLogger) Success(strategy, asset, msg string, args ...interface{})     {}
func (l *noOpTradingLogger) Failed(strategy, asset, msg string, args ...interface{})      {}
func (l *noOpTradingLogger) OrderLifecycle(msg, asset string, args ...interface{})        {}
func (l *noOpTradingLogger) DataCollection(exchange, msg string, args ...interface{})     {}
func (l *noOpTradingLogger) Debug(strategy, asset, msg string, args ...interface{})       {}

// systemTimeProvider implements TimeProvider using real time.Time
type systemTimeProvider struct{}

func (p *systemTimeProvider) Now() time.Time                         { return time.Now() }
func (p *systemTimeProvider) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (p *systemTimeProvider) NewTimer(d time.Duration) temporal.Timer {
	return &systemTimer{time.NewTimer(d)}
}
func (p *systemTimeProvider) Since(t time.Time) time.Duration { return time.Since(t) }
func (p *systemTimeProvider) NewTicker(d time.Duration) temporal.Ticker {
	return &systemTicker{time.NewTicker(d)}
}
func (p *systemTimeProvider) Sleep(d time.Duration) { time.Sleep(d) }

type systemTimer struct {
	t *time.Timer
}

func (st *systemTimer) C() <-chan time.Time        { return st.t.C }
func (st *systemTimer) Reset(d time.Duration) bool { return st.t.Reset(d) }
func (st *systemTimer) Stop() bool                 { return st.t.Stop() }

type systemTicker struct {
	t *time.Ticker
}

func (st *systemTicker) C() <-chan time.Time   { return st.t.C }
func (st *systemTicker) Reset(d time.Duration) { st.t.Reset(d) }
func (st *systemTicker) Stop()                 { st.t.Stop() }

var _ = Describe("Deribit Options Connector", func() {
	var (
		mockClient    adaptor.Client
		appLogger     logging.ApplicationLogger
		tradingLogger logging.TradingLogger
		timeProvider  temporal.TimeProvider
		conn          interface{} // Use interface to test through public API
	)

	BeforeEach(func() {
		mockClient = adaptor.NewClient()
		appLogger = logging.NewNoOpLogger()
		tradingLogger = &noOpTradingLogger{}
		timeProvider = &systemTimeProvider{}

		conn = NewDeribitOptions(mockClient, appLogger, tradingLogger, timeProvider, &noOpWSService{})
	})

	Describe("Initialization", func() {
		It("should not be initialized initially", func() {
			connector := conn.(*deribitOptions)
			Expect(connector.IsInitialized()).To(BeFalse())
		})

		It("should initialize successfully with valid config", func() {
			cfg := &Config{
				ClientID:     "test-id",
				ClientSecret: "test-secret",
			}

			connector := conn.(*deribitOptions)
			err := connector.Initialize(cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(connector.IsInitialized()).To(BeTrue())
		})

		It("should reject nil config", func() {
			connector := conn.(*deribitOptions)
			err := connector.Initialize(nil)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Connector Info", func() {
		It("should return correct connector info", func() {
			connector := conn.(*deribitOptions)

			info := connector.GetConnectorInfo()
			Expect(info).NotTo(BeNil())
			Expect(connector.SupportsTradingOperations()).To(BeTrue())
			Expect(connector.SupportsRealTimeData()).To(BeTrue())
		})
	})

	Describe("Config creation", func() {
		It("should create a new config instance", func() {
			connector := conn.(*deribitOptions)

			cfg := connector.NewConfig()
			Expect(cfg).NotTo(BeNil())
			Expect(cfg).To(BeAssignableToTypeOf(&Config{}))
		})
	})
})
