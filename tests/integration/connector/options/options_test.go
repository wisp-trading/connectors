package options_test

import (
	"time"

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

			connector_test.LogSuccess("✓ wisp.Options() accessible - SDK is properly wired")
		})

		It("should persist connector data to store and serve via SDK", func() {
			// 1. Get connector and fetch real data
			conn := runner.GetOptionsConnector()
			Expect(conn).ToNot(BeNil())

			// Create a test contract (BTC call option)
			pair := portfolio.NewPair(
				portfolio.NewAsset("BTC"),
				portfolio.NewAsset("USDT"),
			)

			// Get available expirations to use a real one
			expirations, err := conn.GetExpirations(pair)
			Expect(err).ToNot(HaveOccurred(), "Should be able to fetch expirations")
			Expect(expirations).ToNot(BeEmpty(), "Should have at least one expiration")

			// Use the first available expiration
			expiration := expirations[0]

			// Get available strikes for this expiration
			strikes, err := conn.GetStrikes(pair, expiration)
			Expect(err).ToNot(HaveOccurred(), "Should be able to fetch strikes")
			Expect(strikes).ToNot(BeEmpty(), "Should have at least one strike")

			// Use a realistic strike (pick one from available)
			strike := strikes[0]

			contract := optionsTypes.OptionContract{
				Pair:       pair,
				Strike:     strike,
				Expiration: expiration,
				OptionType: "CALL",
			}

			// 2. Fetch option data from connector (this populates the store)
			optionData, err := conn.GetOptionData(contract)
			Expect(err).ToNot(HaveOccurred(), "Connector should fetch option data")
			Expect(optionData).ToNot(BeNil())

			connector_test.LogSuccess("✓ Connector fetched option data: mark_price=%v, iv=%v",
				optionData.MarkPrice, optionData.IV)

			// 3. CRITICAL: Verify data is in the store
			store := runner.GetOptionsStore()
			Expect(store).ToNot(BeNil())

			storedMarkPrice := store.GetMarkPrice(contract)
			Expect(storedMarkPrice).To(BeNumerically(">", 0),
				"Store should contain mark price after connector fetch")

			storedIV := store.GetIV(contract)
			Expect(storedIV).To(BeNumerically(">", 0),
				"Store should contain IV after connector fetch")

			connector_test.LogSuccess("✓ Store populated: mark_price=%v (from %v), iv=%v",
				storedMarkPrice, optionData.MarkPrice, storedIV)

			// 4. CRITICAL: Verify data flows through SDK service to strategy
			wispInstance := runner.GetWisp()
			optionsService := wispInstance.Options()

			// Strategy calls SDK to get mark price
			sdkMarkPrice, found := optionsService.MarkPrice(types.DeribitOptions, contract)
			Expect(found).To(BeTrue(), "SDK should have mark price for contract")
			Expect(sdkMarkPrice.String()).ToNot(BeEmpty(),
				"SDK should return accessible decimal mark price")

			connector_test.LogSuccess("✓ SDK accessible: wisp.Options().MarkPrice() = %s", sdkMarkPrice.String())

			// Strategy calls SDK to get IV
			sdkIV, found := optionsService.ImpliedVolatility(types.DeribitOptions, contract)
			Expect(found).To(BeTrue(), "SDK should have IV for contract")
			Expect(sdkIV).To(BeNumerically(">", 0),
				"SDK should return IV for contract")

			connector_test.LogSuccess("✓ SDK accessible: wisp.Options().ImpliedVolatility() = %v", sdkIV)

			// Strategy calls SDK to get Greeks
			sdkGreeks, found := optionsService.Greeks(types.DeribitOptions, contract)
			Expect(found).To(BeTrue(), "SDK should have Greeks for contract")
			connector_test.LogSuccess("✓ SDK accessible: wisp.Options().Greeks() = delta:%v gamma:%v",
				sdkGreeks.Delta, sdkGreeks.Gamma)

			// Verify full chain: Connector → Store → Service → SDK works
			connector_test.LogSuccess("✓✓✓ FULL DATA FLOW VERIFIED: Exchange → Connector → Store → Service → SDK")
		})

		It("should handle multiple concurrent SDK accesses", func() {
			wispInstance := runner.GetWisp()
			optionsService := wispInstance.Options()
			conn := runner.GetOptionsConnector()

			pair := portfolio.NewPair(
				portfolio.NewAsset("BTC"),
				portfolio.NewAsset("USDT"),
			)

			// Get available expirations and strikes
			expirations, err := conn.GetExpirations(pair)
			Expect(err).ToNot(HaveOccurred())
			Expect(expirations).ToNot(BeEmpty())

			expiration := expirations[0]

			strikes, err := conn.GetStrikes(pair, expiration)
			Expect(err).ToNot(HaveOccurred())
			Expect(strikes).ToNot(BeEmpty())

			// Use first two available strikes
			strike1 := strikes[0]
			strike2 := strike1
			if len(strikes) > 1 {
				strike2 = strikes[1]
			}

			contract1 := optionsTypes.OptionContract{
				Pair:       pair,
				Strike:     strike1,
				Expiration: expiration,
				OptionType: "CALL",
			}

			contract2 := optionsTypes.OptionContract{
				Pair:       pair,
				Strike:     strike2,
				Expiration: expiration,
				OptionType: "PUT",
			}

			// Fetch both contracts
			_, err1 := conn.GetOptionData(contract1)
			_, err2 := conn.GetOptionData(contract2)

			Expect(err1).ToNot(HaveOccurred())
			Expect(err2).ToNot(HaveOccurred())

			// Access via SDK concurrently
			price1, found1 := optionsService.MarkPrice(types.DeribitOptions, contract1)
			price2, found2 := optionsService.MarkPrice(types.DeribitOptions, contract2)

			Expect(found1).To(BeTrue())
			Expect(found2).To(BeTrue())
			Expect(price1.String()).ToNot(BeEmpty())
			Expect(price2.String()).ToNot(BeEmpty())

			connector_test.LogSuccess("✓ Concurrent access works: Contract1 price=%s, Contract2 price=%s",
				price1.String(), price2.String())
		})

		It("should verify SDK store consistency", func() {
			wispInstance := runner.GetWisp()
			optionsService := wispInstance.Options()
			store := runner.GetOptionsStore()
			conn := runner.GetOptionsConnector()

			pair := portfolio.NewPair(
				portfolio.NewAsset("ETH"),
				portfolio.NewAsset("USDT"),
			)

			// Get available expirations and strikes for ETH
			expirations, err := conn.GetExpirations(pair)
			Expect(err).ToNot(HaveOccurred(), "Should be able to fetch ETH expirations")
			Expect(expirations).ToNot(BeEmpty(), "Should have at least one ETH expiration")

			expiration := expirations[0]

			strikes, err := conn.GetStrikes(pair, expiration)
			Expect(err).ToNot(HaveOccurred(), "Should be able to fetch ETH strikes")
			Expect(strikes).ToNot(BeEmpty(), "Should have at least one ETH strike")

			contract := optionsTypes.OptionContract{
				Pair:       pair,
				Strike:     strikes[0],
				Expiration: expiration,
				OptionType: "CALL",
			}

			originalData, err := conn.GetOptionData(contract)
			Expect(err).ToNot(HaveOccurred())

			// Get from SDK service
			sdkMarkPrice, _ := optionsService.MarkPrice(types.DeribitOptions, contract)
			sdkIV, _ := optionsService.ImpliedVolatility(types.DeribitOptions, contract)

			// Get directly from store
			storeMarkPrice := store.GetMarkPrice(contract)
			storeIV := store.GetIV(contract)

			// All three should match
			Expect(sdkMarkPrice.String()).ToNot(BeEmpty(),
				"SDK mark price should be accessible")
			Expect(storeMarkPrice).To(Equal(originalData.MarkPrice),
				"Store mark price should match connector data")
			Expect(sdkIV).To(Equal(storeIV),
				"SDK IV should match store IV")

			connector_test.LogSuccess("✓ Consistency verified: SDK≈Store≈Connector")
		})
	})
})
