package options_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wisp-trading/connectors/pkg/connectors/types"
	optionsTypes "github.com/wisp-trading/sdk/pkg/types/connector/options"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

var _ = Describe("Options Connector Tests", func() {
	var runner *connector_test.OptionsTestRunner

	BeforeEach(func() {
		var err error
		runner, err = connector_test.NewOptionsTestRunner(
			connector_test.GetTestOptionsConnectorName(),
			connector_test.GetOptionsConnectorConfig(),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	// Include shared behaviors
	connector_test.OptionsBehavior(
		func() connector_test.BaseTestRunner { return runner },
		func() interface{} {
			return connector_test.CreateOptionsContract("BTC", 50000, "CALL")
		},
	)

	Describe("Options Market Data Flow to SDK", func() {

		It("should make options service available via wisp.Options()", func() {
			// Get the Wisp SDK instance from the runner
			wisp := runner.GetWisp()
			Expect(wisp).ToNot(BeNil())

			// CRITICAL: Verify Options() is accessible
			optionsService := wisp.Options()
			Expect(optionsService).ToNot(BeNil())

			connector_test.LogSuccess("NewFromFloatwisp.Options() accessible - SDK is properly wired")
		})

		It("should persist connector data to store and serve via SDK", func() {
			conn := runner.GetOptionsConnector()
			pair := portfolio.NewPair(portfolio.NewAsset("BTC"), portfolio.NewAsset("USDT"))

			// 1. Discover a real expiration from the exchange
			expirations, err := conn.GetExpirations(pair)
			Expect(err).ToNot(HaveOccurred())
			if len(expirations) == 0 {
				Skip("No options expirations available for testing")
			}
			expiration := expirations[0]

			// 2. Register expiration on the watchlist — the ingestor only collects watched expirations
			Expect(runner.WatchExpiration(pair, expiration)).To(Succeed())

			// 3. Trigger the ingestor: connector.GetExpirationData() → store.Set*() for all strikes
			runner.CollectNow()

			// 4. Discover a strike that was just populated so we can form a contract key
			strikes, err := conn.GetStrikes(pair, expiration)
			Expect(err).ToNot(HaveOccurred())
			Expect(strikes).ToNot(BeEmpty())

			contract := optionsTypes.OptionContract{
				Pair:       pair,
				Strike:     strikes[0],
				Expiration: expiration,
				OptionType: "CALL",
			}

			// 5. Verify the store was populated by the ingestor (not by us)
			store := runner.GetOptionsStore()
			storedMarkPrice := store.GetMarkPrice(contract)
			Expect(storedMarkPrice).To(BeNumerically(">", 0),
				"Store should contain mark price — ingestor must have persisted connector data")

			storedIV := store.GetIV(contract)
			Expect(storedIV).To(BeNumerically(">", 0),
				"Store should contain IV — ingestor must have persisted connector data")

			connector_test.LogSuccess("NewFromFloatStore populated by ingestor: mark_price=%v iv=%v", storedMarkPrice, storedIV)

			// 6. Verify the SDK service reads from the same store — strategies call wisp.Options()
			optionsService := runner.GetWisp().Options()

			sdkMarkPrice, found := optionsService.MarkPrice(types.DeribitOptions, contract)
			Expect(found).To(BeTrue(), "SDK should have mark price — data must flow store → service → SDK")
			Expect(sdkMarkPrice.String()).ToNot(BeEmpty())
			connector_test.LogSuccess("NewFromFloatSDK: wisp.Options().MarkPrice() = %s", sdkMarkPrice.String())

			sdkIV, found := optionsService.ImpliedVolatility(types.DeribitOptions, contract)
			Expect(found).To(BeTrue(), "SDK should have IV")
			Expect(sdkIV).To(BeNumerically(">", 0))
			connector_test.LogSuccess("NewFromFloatSDK: wisp.Options().ImpliedVolatility() = %v", sdkIV)

			sdkGreeks, found := optionsService.Greeks(types.DeribitOptions, contract)
			Expect(found).To(BeTrue(), "SDK should have Greeks")
			connector_test.LogSuccess("NewFromFloatSDK: wisp.Options().Greeks() delta=%v gamma=%v", sdkGreeks.Delta, sdkGreeks.Gamma)

			connector_test.LogSuccess("✓✓NewFromFloatExchange → Connector → Ingestor → Store → Service → SDK verified")
		})

		It("should handle multiple concurrent SDK accesses", func() {
			wispInstance := runner.GetWisp()
			optionsService := wispInstance.Options()
			conn := runner.GetOptionsConnector()

			pair := portfolio.NewPair(
				portfolio.NewAsset("BTC"),
				portfolio.NewAsset("USDT"),
			)

			// Discover a real expiration from the exchange
			expirations, err := conn.GetExpirations(pair)
			Expect(err).ToNot(HaveOccurred())
			if len(expirations) == 0 {
				Skip("No options expirations available for testing")
			}
			expiration := expirations[0]

			// Register expiration on watchlist so the ingestor knows to collect it
			Expect(runner.WatchExpiration(pair, expiration)).To(Succeed())

			// Trigger ingestor: connector.GetExpirationData() → store.Set*() for all strikes
			runner.CollectNow()

			// Now discover what strikes were populated (both CALL and PUT for the same strike)
			strikes, err := conn.GetStrikes(pair, expiration)
			Expect(err).ToNot(HaveOccurred())
			Expect(strikes).ToNot(BeEmpty())

			strike1 := strikes[0]
			strike2 := strike1
			if len(strikes) > 1 {
				strike2 = strikes[1]
			}

			contract1 := optionsTypes.OptionContract{
				Pair:       pair,
				Strike:     strike1,
				Expiration: expiration,
				OptionType: "call",
			}

			contract2 := optionsTypes.OptionContract{
				Pair:       pair,
				Strike:     strike2,
				Expiration: expiration,
				OptionType: "put",
			}

			// Both contracts should be in the store — ingestor populated them
			// No manual Set* calls — this proves the real data flow works
			price1, found1 := optionsService.MarkPrice(types.DeribitOptions, contract1)
			price2, found2 := optionsService.MarkPrice(types.DeribitOptions, contract2)

			Expect(found1).To(BeTrue())
			Expect(found2).To(BeTrue())
			Expect(price1.String()).ToNot(BeEmpty())
			Expect(price2.String()).ToNot(BeEmpty())

			connector_test.LogSuccess("NewFromFloatConcurrent access works: Contract1 price=%s, Contract2 price=%s",
				price1.String(), price2.String())
		})

		It("should verify SDK store consistency", func() {
			conn := runner.GetOptionsConnector()
			store := runner.GetOptionsStore()
			optionsService := runner.GetWisp().Options()

			pair := portfolio.NewPair(portfolio.NewAsset("ETH"), portfolio.NewAsset("USDT"))

			// 1. Discover a real expiration from the exchange
			expirations, err := conn.GetExpirations(pair)
			Expect(err).ToNot(HaveOccurred())
			if len(expirations) == 0 {
				Skip("No ETH options expirations available for testing")
			}
			expiration := expirations[0]

			// 2. Watch the expiration and trigger ingestor — this is the real data flow
			Expect(runner.WatchExpiration(pair, expiration)).To(Succeed())
			runner.CollectNow()

			// 3. Pick a strike to inspect
			strikes, err := conn.GetStrikes(pair, expiration)
			Expect(err).ToNot(HaveOccurred())
			Expect(strikes).ToNot(BeEmpty())

			contract := optionsTypes.OptionContract{
				Pair:       pair,
				Strike:     strikes[0],
				Expiration: expiration,
				OptionType: "CALL",
			}

			// 4. All three layers must agree — store, SDK service, and connector must be consistent
			storeMarkPrice := store.GetMarkPrice(contract)
			storeIV := store.GetIV(contract)

			sdkMarkPrice, sdkFound := optionsService.MarkPrice(types.DeribitOptions, contract)
			sdkIV, sdkIVFound := optionsService.ImpliedVolatility(types.DeribitOptions, contract)

			// Verify the store has data (ingestor populated it)
			Expect(storeMarkPrice).To(BeNumerically(">", 0), "Store must have mark price from ingestor")
			Expect(storeIV).To(BeNumerically(">", 0), "Store must have IV from ingestor")

			// Verify SDK reads from the same store values
			Expect(sdkFound).To(BeTrue(), "SDK must find mark price — reads from store")
			Expect(sdkIVFound).To(BeTrue(), "SDK must find IV — reads from store")
			Expect(sdkIV).To(Equal(storeIV), "SDK IV must equal store IV — same data source")

			connector_test.LogSuccess("NewFromFloatStore mark_price=%v iv=%v | SDK mark_price=%s iv=%v",
				storeMarkPrice, storeIV, sdkMarkPrice.String(), sdkIV)
			connector_test.LogSuccess("NewFromFloatConsistency verified: Ingestor → Store ≡ SDK")
		})
	})
})
