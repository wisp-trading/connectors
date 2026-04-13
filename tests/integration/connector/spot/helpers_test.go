package spot_test

import (
	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

// testPair returns the default spot test pair constructed from environment
// config. Centralised here so every test file uses one definition.
func testPair() connector_test.Pair {
	return connector_test.CreatePairWithQuote(
		connector_test.GetSpotSymbol(),
		connector_test.GetSpotQuote(),
	)
}
