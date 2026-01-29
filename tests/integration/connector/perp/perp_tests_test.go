package perp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

var _ = Describe("Perp Connector Tests", func() {
	var runner *connector_test.PerpTestRunner

	BeforeEach(func() {
		var err error
		runner, err = connector_test.NewPerpTestRunner(
			connector_test.GetTestPerpConnectorName(),
			connector_test.GetPerpConnectorConfig(),
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
		func() string { return runner.GetPerpSymbol(connector_test.CreateAsset("ETH")) },
	)

	connector_test.AccountBehavior(
		func() connector_test.BaseTestRunner { return runner },
	)

	connector_test.WebSocketLifecycleBehavior(
		func() connector_test.BaseTestRunner { return runner },
	)

	// PERP-SPECIFIC TESTS

	Describe("Funding Rates", func() {
		Context("FetchFundingRate", func() {
			It("should fetch funding rate for asset", func() {
				conn := runner.GetPerpConnector()
				asset := connector_test.CreateAsset("ETH")

				fr, err := conn.FetchFundingRate(asset)
				Expect(err).ToNot(HaveOccurred())
				Expect(fr).ToNot(BeNil())

				connector_test.LogSuccess("Funding rate for ETH: %s", fr.CurrentRate.String())
			})
		})

		Context("FetchCurrentFundingRates", func() {
			It("should fetch all current funding rates", func() {
				conn := runner.GetPerpConnector()

				rates, err := conn.FetchCurrentFundingRates()
				Expect(err).ToNot(HaveOccurred())
				Expect(rates).ToNot(BeEmpty())

				connector_test.LogSuccess("Fetched funding rates for %d assets", len(rates))
			})
		})
	})

	Describe("Positions", func() {
		Context("GetPositions", func() {
			It("should fetch positions", func() {
				conn := runner.GetPerpConnector()

				positions, err := conn.GetPositions()
				Expect(err).ToNot(HaveOccurred())
				Expect(positions).ToNot(BeNil())

				connector_test.LogSuccess("Positions: %d open", len(positions))
			})
		})
	})

	Describe("Perp WebSocket Subscriptions", func() {
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

		Context("Position Subscription", func() {
			It("should subscribe to positions", func() {
				wsConn := runner.GetWebSocketConnector()
				asset := connector_test.CreateAsset("ETH")

				err := wsConn.SubscribePositions(asset)
				Expect(err).ToNot(HaveOccurred())

				posCh := wsConn.PositionUpdates()
				Expect(posCh).ToNot(BeNil())

				connector_test.LogSuccess("Position subscription active")
			})
		})

		Context("Funding Rate Subscription", func() {
			It("should subscribe to funding rates", func() {
				wsConn := runner.GetWebSocketConnector()
				asset := connector_test.CreateAsset("ETH")

				err := wsConn.SubscribeFundingRates(asset)
				Expect(err).ToNot(HaveOccurred())

				frCh := wsConn.FundingRateUpdates()
				Expect(frCh).ToNot(BeNil())

				connector_test.LogSuccess("Funding rate subscription active")
			})
		})

		Context("OrderBook Subscription", func() {
			It("should subscribe and receive updates", func() {
				wsConn := runner.GetWebSocketConnector()
				asset := connector_test.CreateAsset("ETH")

				err := wsConn.SubscribeOrderBook(asset)
				Expect(err).ToNot(HaveOccurred())

				channels := wsConn.GetOrderBookChannels()
				Expect(channels).ToNot(BeEmpty())

				connector_test.LogSuccess("OrderBook subscription active for ETH")
			})
		})

		Context("Klines Subscription", func() {
			It("should subscribe and receive updates", func() {
				wsConn := runner.GetWebSocketConnector()
				asset := connector_test.CreateAsset("ETH")

				err := wsConn.SubscribeKlines(asset, "1m")
				Expect(err).ToNot(HaveOccurred())

				channels := wsConn.GetKlineChannels()
				Expect(channels).ToNot(BeEmpty())

				connector_test.LogSuccess("Klines subscription active for ETH")
			})
		})
	})
})
