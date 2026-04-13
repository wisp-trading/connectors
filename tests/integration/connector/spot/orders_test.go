package spot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// orders_test.go — Verifies order-related flows via the SDK's Signal builder
// and the Positions/Trades read APIs.
//
// Live trading tests are gated behind ENABLE_SPOT_TRADING_TESTS=true.
// Orders are placed far from market (50% below mid) so they rest on
// the book and never fill.

var _ = Describe("Spot — Orders via SDK", func() {
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
	// Positions (read-only, always safe)
	// ────────────────────────────────────────────────────────────────

	Context("Positions", func() {
		It("should return positions (possibly empty) without error", func() {
			positions := runner.Spot().Positions()
			Expect(positions).ToNot(BeNil())

			connector_test.LogSuccess("Spot().Positions(): %d orders", len(positions))
		})
	})

	// ────────────────────────────────────────────────────────────────
	// Trades (read-only, always safe)
	// ────────────────────────────────────────────────────────────────

	Context("Trades", func() {
		It("should return trades (possibly empty) without error", func() {
			trades := runner.Spot().Trades()
			Expect(trades).ToNot(BeNil())

			connector_test.LogSuccess("Spot().Trades(): %d trades", len(trades))
		})
	})

	// ────────────────────────────────────────────────────────────────
	// Build a limit order signal via Signal (requires ENABLE_SPOT_TRADING_TESTS)
	// ────────────────────────────────────────────────────────────────

	Context("Signal — BuyLimit", func() {
		It("should build a limit buy signal without error", func() {
			if !connector_test.IsSpotTradingEnabled() {
				Skip("Live trading tests disabled (set ENABLE_SPOT_TRADING_TESTS=true)")
			}

			pair := testPair()
			exchange := runner.ExchangeName()

			// Watch the pair so the ingestor provides price data
			runner.WatchPair(pair)

			// Wait for a price to arrive via the automatic pipeline
			Eventually(func() bool {
				price, ok := runner.Spot().Price(exchange, pair)
				return ok && price.IsPositive()
			}, "60s", "1s").Should(BeTrue(), "Need a live price to place a safe order")

			price, _ := runner.Spot().Price(exchange, pair)
			limitPrice := price.Mul(numerical.NewFromFloat(0.5)) // 50% below market
			qty := numerical.NewFromFloat(0.1)

			connector_test.LogInfo("Building limit BUY %s %s @ %s (market: %s)",
				qty.String(), pair.Base().Symbol(), limitPrice.String(), price.String())

			// Build the signal through the SDK — validates the builder pipeline.
			// Emission (wisp.Emit) is not called here to avoid placing a real
			// order in every test run; that is covered by a separate live test.
			signal, err := runner.Spot().Signal(connector_test.TestStrategyName).
				BuyLimit(pair, exchange, qty, limitPrice).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(signal).ToNot(BeNil())

			connector_test.LogSuccess("Signal built: BuyLimit %s @ %s", qty.String(), limitPrice.String())
		})
	})

	// ────────────────────────────────────────────────────────────────
	// PNL
	// ────────────────────────────────────────────────────────────────

	Context("PNL", func() {
		It("should return PNL service without error", func() {
			pnl := runner.Spot().PNL()
			Expect(pnl).ToNot(BeNil())

			connector_test.LogSuccess("Spot().PNL() available")
		})
	})
})
