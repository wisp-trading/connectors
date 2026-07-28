package perp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"

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

	// Batch: connector → CollectNow → store → SDK
	connector_test.MarketDataBehavior(
		func() connector_test.PairMarketTestRunner { return runner },
		func() portfolio.Pair { return connector_test.CreatePair(connector_test.GetPerpSymbol()) },
	)

	// Realtime: WatchPair → StartRealtime → WS → store → SDK
	connector_test.WebSocketMarketDataStoreBehavior(
		func() connector_test.PairMarketTestRunner { return runner },
		func() portfolio.Pair { return connector_test.CreatePair(connector_test.GetPerpSymbol()) },
	)

	connector_test.AccountBehavior(
		func() connector_test.BaseTestRunner { return runner },
	)

	connector_test.WebSocketLifecycleBehavior(
		func() connector_test.BaseTestRunner { return runner },
	)

	// Perp-specific connector APIs (no pair MarketStore path for funding/positions yet).
	Describe("Funding Rates (connector-only)", func() {
		It("should fetch funding rate for asset", func() {
			conn := runner.GetPerpConnector()
			asset := connector_test.CreatePair(connector_test.GetPerpSymbol())
			fr, err := conn.FetchFundingRate(asset)
			Expect(err).ToNot(HaveOccurred())
			Expect(fr).ToNot(BeNil())
			connector_test.LogSuccess("Funding rate: %s", fr.CurrentRate.String())
		})

		It("should fetch all current funding rates", func() {
			conn := runner.GetPerpConnector()
			rates, err := conn.FetchCurrentFundingRates()
			Expect(err).ToNot(HaveOccurred())
			Expect(rates).ToNot(BeEmpty())
			connector_test.LogSuccess("Fetched funding rates for %d assets", len(rates))
		})
	})

	Describe("Positions (connector-only)", func() {
		It("should fetch positions", func() {
			conn := runner.GetPerpConnector()
			positions, err := conn.GetPositions()
			Expect(err).ToNot(HaveOccurred())
			Expect(positions).ToNot(BeNil())
			connector_test.LogSuccess("Positions: %d open", len(positions))
		})
	})

	Describe("Perp WebSocket extras (connector-only)", func() {
		BeforeEach(func() {
			if !runner.HasWebSocketSupport() {
				Skip("Connector does not support WebSocket")
			}
			wsConn := runner.GetWebSocketConnector()
			Expect(wsConn.StartWebSocket()).To(Succeed())
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

		It("should subscribe to positions", func() {
			wsConn := runner.GetWebSocketConnector()
			asset := connector_test.CreatePair(connector_test.GetPerpSymbol())
			Expect(wsConn.SubscribePositions(asset)).To(Succeed())
			Expect(wsConn.PositionUpdates()).ToNot(BeNil())
			connector_test.LogSuccess("Position subscription active")
		})

		It("should subscribe to funding rates", func() {
			wsConn := runner.GetWebSocketConnector()
			asset := connector_test.CreatePair(connector_test.GetPerpSymbol())
			Expect(wsConn.SubscribeFundingRates(asset)).To(Succeed())
			Expect(wsConn.FundingRateUpdates()).ToNot(BeNil())
			connector_test.LogSuccess("Funding rate subscription active")
		})
	})
})
