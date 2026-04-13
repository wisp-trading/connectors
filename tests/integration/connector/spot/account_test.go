package spot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	"github.com/wisp-trading/sdk/pkg/markets/base/types/stores/market"
)

// account_test.go — Verifies account-level data flows through the SDK.
//
// The SDK's Spot() public API exposes Trades(), Positions(), and PNL() for
// account activity. Balance queries (GetBalance, GetBalances) are connector-
// level operations not yet exposed on the Spot interface — those will be
// tested once the SDK adds a public balance API.

var _ = Describe("Spot — Account Activity via SDK", func() {
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
	// Trades — filtered by exchange and pair
	// ────────────────────────────────────────────────────────────────

	Context("Trades", func() {
		It("should return all trades without error", func() {
			trades := runner.Spot().Trades()
			Expect(trades).ToNot(BeNil())

			connector_test.LogSuccess("Spot().Trades(): %d trades", len(trades))
		})

		It("should support filtering trades by exchange", func() {
			exchange := runner.ExchangeName()
			trades := runner.Spot().Trades(market.ActivityQuery{Exchange: &exchange})
			Expect(trades).ToNot(BeNil())

			connector_test.LogSuccess("Spot().Trades(exchange=%s): %d trades",
				exchange, len(trades))
		})

		It("should support filtering trades by pair", func() {
			pair := testPair()
			trades := runner.Spot().Trades(market.ActivityQuery{Pair: &pair})
			Expect(trades).ToNot(BeNil())

			connector_test.LogSuccess("Spot().Trades(pair=%s): %d trades",
				pair, len(trades))
		})
	})

	// ────────────────────────────────────────────────────────────────
	// Positions — filtered by exchange and pair
	// ────────────────────────────────────────────────────────────────

	Context("Positions", func() {
		It("should return all positions without error", func() {
			positions := runner.Spot().Positions()
			Expect(positions).ToNot(BeNil())

			connector_test.LogSuccess("Spot().Positions(): %d orders", len(positions))
		})

		It("should support filtering positions by exchange", func() {
			exchange := runner.ExchangeName()
			positions := runner.Spot().Positions(market.ActivityQuery{Exchange: &exchange})
			Expect(positions).ToNot(BeNil())

			connector_test.LogSuccess("Spot().Positions(exchange=%s): %d orders",
				exchange, len(positions))
		})
	})

	// ────────────────────────────────────────────────────────────────
	// PNL
	// ────────────────────────────────────────────────────────────────

	Context("PNL", func() {
		It("should return PNL service without error", func() {
			pnl := runner.Spot().PNL()
			Expect(pnl).ToNot(BeNil())

			connector_test.LogSuccess("Spot().PNL() service available")
		})
	})
})
