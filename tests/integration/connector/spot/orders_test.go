package spot_test

import (
	"math"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// orders_test.go — Verifies order-related flows via the SDK's Signal builder
// and the Positions/Trades read APIs.
//
// Live trading tests are gated behind ENABLE_SPOT_TRADING_TESTS=true.
// Orders are placed far from market (10% of mid) so they rest on the book
// and never fill, then cancelled immediately after verification.

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
	// Place → Verify → Cancel a limit order (requires ENABLE_SPOT_TRADING_TESTS)
	// ────────────────────────────────────────────────────────────────

	Context("Signal — BuyLimit round-trip", func() {
		It("should place a limit buy far from market, verify it rests, then cancel", func() {
			if !connector_test.IsSpotTradingEnabled() {
				Skip("Live trading tests disabled (set ENABLE_SPOT_TRADING_TESTS=true)")
			}

			pair := testPair()
			exchange := runner.ExchangeName()

			// Watch the pair so the ingestor provides price data.
			runner.WatchPair(pair)

			// Wait for a price to arrive via the automatic pipeline.
			Eventually(func() bool {
				price, ok := runner.Spot().Price(exchange, pair)
				return ok && price.IsPositive()
			}, "60s", "1s").Should(BeTrue(), "Need a live price to place a safe order")

			price, _ := runner.Spot().Price(exchange, pair)

			// Place at 50% of market price — far enough to never fill, but
			// high enough to satisfy Hyperliquid's $10 minimum order value.
			limitPrice := price.Mul(numerical.NewFromFloat(0.50))

			// Compute minimum quantity to meet $10 minimum, then add a margin.
			// minQty = ceil(10 / limitPrice) + 1
			limitPriceF, _ := limitPrice.Float64()
			minQty := math.Ceil(10.0/limitPriceF) + 1
			qty := numerical.NewFromFloat(minQty)

			connector_test.LogInfo("Placing limit BUY %s %s @ %s (market: %s)",
				qty.String(), pair.Base().Symbol(), limitPrice.String(), price.String())

			// ── 1. Build and emit the signal ────────────────────────
			signal, err := runner.Spot().Signal(connector_test.TestStrategyName).
				BuyLimit(pair, exchange, qty, limitPrice).
				Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(signal).ToNot(BeNil())

			result, ok := runner.Emit(signal).AwaitWithTimeout(15 * time.Second)
			Expect(ok).To(BeTrue(), "Execution should complete within 15s")
			Expect(result.Error).ToNot(HaveOccurred(), "Order placement should succeed")
			Expect(result.OrderIDs).ToNot(BeEmpty(), "Should return at least one order ID")

			orderID := result.OrderIDs[0]
			connector_test.LogSuccess("Order placed: ID=%s", orderID)

			// ── 2. Verify the order rests on the book ───────────────
			// GetOpenOrders returns ALL open orders across all pairs.
			orders, err := runner.GetOpenOrders()
			Expect(err).ToNot(HaveOccurred())

			connector_test.LogInfo("GetOpenOrders returned %d orders", len(orders))
			found := false
			for _, o := range orders {
				connector_test.LogInfo("  order: ID=%s pair=%s side=%s price=%s qty=%s",
					o.ID, o.Pair, o.Side, o.Price.String(), o.Quantity.String())
				if o.ID == orderID {
					found = true
				}
			}

			if !found {
				connector_test.LogInfo("Order %s not found in %d open orders — trying cancel anyway to clean up", orderID, len(orders))
				// Cancel to avoid leaving orphaned orders, even if we couldn't verify.
				cancelResp, cancelErr := runner.CancelOrder(orderID, pair)
				if cancelErr != nil {
					connector_test.LogError("Cleanup cancel failed: %v", cancelErr)
				} else {
					connector_test.LogInfo("Cleanup cancel succeeded: %s", cancelResp.Status)
				}
			}
			Expect(found).To(BeTrue(), "Order %s should appear in open orders", orderID)
			connector_test.LogSuccess("Order %s confirmed on book (%d open orders)", orderID, len(orders))

			// ── 3. Cancel the order ─────────────────────────────────
			cancelResp, err := runner.CancelOrder(orderID, pair)
			Expect(err).ToNot(HaveOccurred())
			Expect(cancelResp).ToNot(BeNil())

			connector_test.LogSuccess("Order %s cancelled: status=%s", orderID, cancelResp.Status)

			// ── 4. Verify the order is gone ─────────────────────────
			ordersAfter, err := runner.GetOpenOrders(pair)
			Expect(err).ToNot(HaveOccurred())

			stillOpen := false
			for _, o := range ordersAfter {
				if o.ID == orderID {
					stillOpen = true
					break
				}
			}
			Expect(stillOpen).To(BeFalse(), "Order %s should no longer be on the book after cancel", orderID)
			connector_test.LogSuccess("Confirmed order %s removed from book", orderID)
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
