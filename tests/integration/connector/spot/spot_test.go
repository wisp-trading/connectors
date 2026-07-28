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

	// Batch: connector → CollectNow → store → SDK
	connector_test.MarketDataBehavior(
		func() connector_test.PairMarketTestRunner { return runner },
		func() portfolio.Pair { return connector_test.CreatePair(connector_test.GetSpotSymbol()) },
	)

	// Realtime: WatchPair → StartRealtime → WS → store → SDK
	connector_test.WebSocketMarketDataStoreBehavior(
		func() connector_test.PairMarketTestRunner { return runner },
		func() portfolio.Pair { return connector_test.CreatePair(connector_test.GetSpotSymbol()) },
	)

	connector_test.AccountBehavior(
		func() connector_test.BaseTestRunner { return runner },
	)

	connector_test.WebSocketLifecycleBehavior(
		func() connector_test.BaseTestRunner { return runner },
	)

	// Account WS has no MarketStore path — connector channel only.
	Describe("Spot WebSocket Account (connector-only)", func() {
		It("should subscribe to balance updates", func() {
			if !runner.HasWebSocketSupport() {
				Skip("Connector does not support WebSocket")
			}
			wsConn := runner.GetWebSocketConnector()
			Expect(wsConn.StartWebSocket()).To(Succeed())
			Eventually(wsConn.IsWebSocketConnected, "10s").Should(BeTrue())
			defer func() { _ = wsConn.StopWebSocket() }()

			Expect(wsConn.SubscribeAccountBalance()).To(Succeed())
			Expect(wsConn.AssetBalanceUpdates()).ToNot(BeNil())
			connector_test.LogSuccess("Balance subscription active (no pair MarketStore path)")
		})
	})
})
