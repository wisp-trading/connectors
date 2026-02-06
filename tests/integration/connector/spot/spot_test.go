package spot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

var _ = Describe("Spot Connector Tests", func() {
	var runner *connector_test.SpotTestRunner

	BeforeEach(func() {
		var err error
		runner, err = connector_test.NewSpotTestRunner(
			connector_test.GetTestSpotConnectorName(),
			connector_test.GetSpotConnectorConfig(),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	// Include shared behaviors
	connector_test.MarketDataBehavior(
		func() connector_test.BaseTestRunner { return runner },
		func() portfolio.Pair { return connector_test.CreatePair("ETH") },
	)

	connector_test.AccountBehavior(
		func() connector_test.BaseTestRunner { return runner },
	)

	connector_test.WebSocketLifecycleBehavior(
		func() connector_test.BaseTestRunner { return runner },
	)

	// Spot-specific WebSocket subscriptions
	Describe("Spot WebSocket Subscriptions", func() {
		BeforeEach(func() {
			if !runner.HasWebSocketSupport() {
				Skip("Connector does not support WebSocket")
			}
			wsConn := runner.GetWebSocketConnector()
			err := wsConn.StartWebSocket()
			Expect(err).ToNot(HaveOccurred())
			Eventually(wsConn.IsWebSocketConnected, "10s").Should(BeTrue())
		})

		AfterEach(func() {
			if runner.HasWebSocketSupport() {
				wsConn := runner.GetWebSocketConnector()
				if wsConn.IsWebSocketConnected() {
					_ = wsConn.StopWebSocket()
				}
			}
		})

		Context("OrderBook Subscription", func() {
			It("should subscribe and receive updates", func() {
				wsConn := runner.GetWebSocketConnector()
				asset := connector_test.CreatePair(connector_test.GetSpotSymbol())

				err := wsConn.SubscribeOrderBook(asset)
				Expect(err).ToNot(HaveOccurred())

				channels := wsConn.GetOrderBookChannels()
				Expect(channels).ToNot(BeEmpty())

				connector_test.LogSuccess("OrderBook subscription active for %s", connector_test.GetSpotSymbol())
			})
		})

		Context("Klines Subscription", func() {
			It("should subscribe and receive updates", func() {
				wsConn := runner.GetWebSocketConnector()
				asset := connector_test.CreatePair(connector_test.GetSpotSymbol())

				err := wsConn.SubscribeKlines(asset, "1m")
				Expect(err).ToNot(HaveOccurred())

				channels := wsConn.GetKlineChannels()
				Expect(channels).ToNot(BeEmpty())

				connector_test.LogSuccess("Klines subscription active for %s", connector_test.GetSpotSymbol())
			})
		})

		Context("Account Balance Subscription", func() {
			It("should subscribe to balance updates", func() {
				wsConn := runner.GetWebSocketConnector()

				err := wsConn.SubscribeAccountBalance()
				Expect(err).ToNot(HaveOccurred())

				balanceCh := wsConn.AssetBalanceUpdates()
				Expect(balanceCh).ToNot(BeNil())

				connector_test.LogSuccess("Balance subscription active")
			})
		})
	})
})
