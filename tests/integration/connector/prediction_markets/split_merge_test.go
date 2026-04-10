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

// splitMergeMarketSlug is a long-running market used as the test fixture.
// SplitPosition and MergePositions only require a valid condition ID; any
// active market will do. Use GetMarket (single fast API call) rather than
// paginating Markets() to avoid multi-second Gamma API delays.
const splitMergeMarketSlug = "will-jesus-christ-return-before-2027"

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

			market, err := conn.GetMarket(splitMergeMarketSlug)
			Expect(err).ToNot(HaveOccurred(), "could not fetch test market")

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

			market, err := conn.GetMarket(splitMergeMarketSlug)
			Expect(err).ToNot(HaveOccurred(), "could not fetch test market")

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

			market, err := conn.GetMarket(splitMergeMarketSlug)
			Expect(err).ToNot(HaveOccurred(), "could not fetch test market")

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
