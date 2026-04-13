package spot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

// websocket_test.go — Verifies that the SDK's realtime data pipeline works
// end-to-end for spot connectors.
//
// The SDK automatically creates realtime ingestors for WebSocket-capable
// connectors. These tests verify that data arrives through the public
// Spot() API — whether via batch (30s cycle) or realtime (WebSocket).
// The strategy never manages WebSocket connections directly.

var _ = Describe("Spot — Realtime Data Pipeline", func() {
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

	// ────────────────────────────────────────────────────────────────
	// Data arrival — price should appear after watching a pair
	// ────────────────────────────────────────────────────────────────

	Context("Data arrival", func() {
		It("should receive price data after watching a pair", func() {
			pair := testPair()
			exchange := runner.ExchangeName()

			runner.WatchPair(pair)

			// Data should arrive within 60s — either via an immediate batch
			// collection (fired on coordinator startup) or via the realtime
			// WebSocket ingestor if the connector supports it.
			Eventually(func() bool {
				price, ok := runner.Spot().Price(exchange, pair)
				return ok && price.IsPositive()
			}, "60s", "1s").Should(BeTrue(),
				"Price data should arrive via the SDK pipeline")

			price, _ := runner.Spot().Price(exchange, pair)
			connector_test.LogSuccess("Price arrived: %s", price.String())
		})

		It("should receive order book data after watching a pair", func() {
			pair := testPair()
			exchange := runner.ExchangeName()

			runner.WatchPair(pair)

			Eventually(func() bool {
				ob, ok := runner.Spot().OrderBook(exchange, pair)
				return ok && ob != nil && len(ob.Bids) > 0 && len(ob.Asks) > 0
			}, "60s", "1s").Should(BeTrue(),
				"OrderBook data should arrive via the SDK pipeline")

			ob, _ := runner.Spot().OrderBook(exchange, pair)
			connector_test.LogSuccess("OrderBook arrived: %d bids, %d asks",
				len(ob.Bids), len(ob.Asks))
		})
	})
})
