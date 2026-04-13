package spot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

// spot_test.go — Verifies that the SDK lifecycle boots cleanly and the
// connector is reachable through the public Spot() API.

var _ = Describe("Spot — Initialisation", func() {
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

	It("should boot the SDK and expose Spot() without error", func() {
		spot := runner.Spot()
		Expect(spot).ToNot(BeNil())

		connector_test.LogSuccess("SDK booted — Spot() available for %s",
			runner.ExchangeName())
	})

	It("should allow watching a pair via the public API", func() {
		pair := connector_test.CreatePairWithQuote(
			connector_test.GetSpotSymbol(),
			connector_test.GetSpotQuote(),
		)

		// WatchPair should not panic or error
		runner.WatchPair(pair)

		connector_test.LogSuccess("WatchPair(%s) accepted via SDK",
			pair.Base().Symbol()+"/"+pair.Quote().Symbol())
	})
})
