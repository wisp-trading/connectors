package prediction_markets_test

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

// splitMergeAmount is the USDC amount used for split/merge tests: $1.00 in 6-decimal units.
// This is the smallest meaningful amount and keeps real on-chain costs minimal.
var splitMergeAmount = big.NewInt(1_000_000)

// findNegRiskMarket returns the first active NegRisk market available on the connector.
// It fetches a small page of active markets and scans for one with NegRisk=true.
func findNegRiskMarket(conn prediction.Connector) (prediction.Market, error) {
	limit := 50
	active := true
	markets, err := conn.Markets(&prediction.MarketsFilter{
		Limit:  &limit,
		Active: &active,
	})
	if err != nil {
		return prediction.Market{}, fmt.Errorf("failed to fetch markets: %w", err)
	}

	for _, m := range markets {
		if m.NegRisk {
			return m, nil
		}
	}
	return prediction.Market{}, fmt.Errorf("no active NegRisk market found in first %d results", limit)
}

var _ = Describe("CTF Split and Merge Integration Tests", func() {
	var runner *connector_test.PredictionMarketTestRunner

	BeforeEach(func() {
		// Skip if the Polygon RPC URL has not been configured — on-chain calls require it.
		if os.Getenv("POLYGON_RPC_URL") == "" {
			Skip("POLYGON_RPC_URL not set — skipping on-chain CTF tests")
		}

		var err error
		runner, err = connector_test.NewPredictionMarketTestRunner(
			connector_test.GetTestPredictionMarketConnectorName(),
			connector_test.GetPredictionMarketConnectorConfig(),
		)
		Expect(err).ToNot(HaveOccurred(), "failed to initialise test runner")
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	Describe("SplitPosition", func() {
		It("mints YES+NO tokens on Polygon and returns a non-empty tx hash", func() {
			conn := runner.GetPredictionMarketConnector()

			market, err := findNegRiskMarket(conn)
			Expect(err).ToNot(HaveOccurred(), "could not find a NegRisk market to test against")

			fmt.Fprintf(GinkgoWriter, "SplitPosition: market=%s id=%s amount=%s USDC\n",
				market.Slug, market.MarketID, splitMergeAmount.String())

			txHash, err := conn.SplitPosition(market, splitMergeAmount)
			Expect(err).ToNot(HaveOccurred(), "SplitPosition should succeed")
			Expect(txHash).ToNot(BeEmpty(), "should receive a transaction hash")
			Expect(strings.HasPrefix(txHash, "0x")).To(BeTrue(), "tx hash should be hex-prefixed")
			Expect(txHash).To(HaveLen(66), "tx hash should be 32 bytes (66 hex chars including 0x)")

			fmt.Fprintf(GinkgoWriter, "SplitPosition tx: %s\n", txHash)
		})
	})

	Describe("MergePositions", func() {
		It("burns YES+NO tokens on Polygon and returns a non-empty tx hash", func() {
			conn := runner.GetPredictionMarketConnector()

			market, err := findNegRiskMarket(conn)
			Expect(err).ToNot(HaveOccurred(), "could not find a NegRisk market to test against")

			fmt.Fprintf(GinkgoWriter, "MergePositions: market=%s id=%s amount=%s USDC\n",
				market.Slug, market.MarketID, splitMergeAmount.String())

			txHash, err := conn.MergePositions(market, splitMergeAmount)
			Expect(err).ToNot(HaveOccurred(), "MergePositions should succeed")
			Expect(txHash).ToNot(BeEmpty(), "should receive a transaction hash")
			Expect(strings.HasPrefix(txHash, "0x")).To(BeTrue(), "tx hash should be hex-prefixed")
			Expect(txHash).To(HaveLen(66), "tx hash should be 32 bytes (66 hex chars including 0x)")

			fmt.Fprintf(GinkgoWriter, "MergePositions tx: %s\n", txHash)
		})
	})

	Describe("Split then Merge round-trip", func() {
		It("splits $1 into YES+NO tokens then merges back to $1 USDC", func() {
			conn := runner.GetPredictionMarketConnector()

			market, err := findNegRiskMarket(conn)
			Expect(err).ToNot(HaveOccurred(), "could not find a NegRisk market to test against")

			fmt.Fprintf(GinkgoWriter, "Round-trip: market=%s id=%s\n", market.Slug, market.MarketID)

			// ── Split ────────────────────────────────────────────────────────────
			splitTx, err := conn.SplitPosition(market, splitMergeAmount)
			Expect(err).ToNot(HaveOccurred(), "SplitPosition should succeed")
			Expect(splitTx).ToNot(BeEmpty())
			fmt.Fprintf(GinkgoWriter, "Split tx:  %s\n", splitTx)

			// Allow time for the split tx to be mined before merging.
			time.Sleep(5 * time.Second)

			// ── Merge ────────────────────────────────────────────────────────────
			mergeTx, err := conn.MergePositions(market, splitMergeAmount)
			Expect(err).ToNot(HaveOccurred(), "MergePositions should succeed after split")
			Expect(mergeTx).ToNot(BeEmpty())
			fmt.Fprintf(GinkgoWriter, "Merge tx:  %s\n", mergeTx)

			Expect(splitTx).ToNot(Equal(mergeTx), "split and merge should be distinct transactions")
		})
	})
})
