package prediction_markets_test

import (
	"fmt"
	"math/big"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// splitMergeAmount is the USDC amount used for split/merge tests: $1.00 in 6-decimal units.
// This is the smallest meaningful amount and keeps real on-chain costs minimal.
var splitMergeAmount = big.NewInt(1_000_000)

// splitMergeMarketSlug is a long-running market used as the test fixture.
// SplitPosition and MergePositions only require a valid condition ID; any
// active market will do. Use GetMarket (single fast API call) rather than
// paginating Markets() to avoid multi-second Gamma API delays.
const splitMergeMarketSlug = "will-jesus-christ-return-before-2027"

// minMATICBalance is the minimum MATIC balance required for gas (0.01 MATIC).
var minMATICBalance = numerical.NewFromFloat(0.01)

// minUSDCBalance is the minimum USDC balance required for split/merge ($1.00).
var minUSDCBalance = numerical.NewFromFloat(1.00)

// balancePreflightCheck uses the connector's GetBalances() to verify the wallet
// has enough MATIC for gas and USDC for the test. Skips the test if insufficient.
func balancePreflightCheck(conn prediction.Connector) {
	balances, err := conn.GetBalances()
	if err != nil {
		Skip(fmt.Sprintf("failed to fetch balances: %v", err))
	}

	var maticBalance, usdcBalance numerical.Decimal
	for _, b := range balances {
		switch b.Asset.Symbol() {
		case "MATIC":
			maticBalance = b.Free
		case "USDC":
			usdcBalance = b.Free
		}
	}

	fmt.Fprintf(GinkgoWriter, "MATIC: %s  USDC: $%s\n",
		maticBalance.StringFixed(6), usdcBalance.StringFixed(2))

	if maticBalance.LessThan(minMATICBalance) {
		Skip(fmt.Sprintf("insufficient MATIC: have %s, need %s", maticBalance.StringFixed(6), minMATICBalance.StringFixed(6)))
	}
	if usdcBalance.LessThan(minUSDCBalance) {
		Skip(fmt.Sprintf("insufficient USDC: have $%s, need $%s", usdcBalance.StringFixed(2), minUSDCBalance.StringFixed(2)))
	}
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
			balancePreflightCheck(conn)

			market, err := conn.GetMarket(splitMergeMarketSlug)
			Expect(err).ToNot(HaveOccurred(), "could not fetch test market")

			fmt.Fprintf(GinkgoWriter, "SplitPosition: market=%s amount=$1.00\n", market.Slug)

			txHash, ready, err := conn.SplitPosition(market, splitMergeAmount)
			Expect(err).ToNot(HaveOccurred(), "SplitPosition should succeed")
			Expect(txHash).ToNot(BeEmpty(), "should receive a transaction hash")
			Expect(strings.HasPrefix(txHash, "0x")).To(BeTrue(), "tx hash should be hex-prefixed")
			Expect(txHash).To(HaveLen(66), "tx hash should be 32 bytes (66 hex chars including 0x)")

			fmt.Fprintf(GinkgoWriter, "SplitPosition tx: %s (awaiting confirmation)\n", txHash)
			Expect(<-ready).ToNot(HaveOccurred(), "split tx should be mined and CLOB notified")
		})
	})

	Describe("MergePositions", func() {
		It("burns YES+NO tokens on Polygon and returns a non-empty tx hash", func() {
			conn := runner.GetPredictionMarketConnector()
			balancePreflightCheck(conn)

			market, err := conn.GetMarket(splitMergeMarketSlug)
			Expect(err).ToNot(HaveOccurred(), "could not fetch test market")

			fmt.Fprintf(GinkgoWriter, "MergePositions: market=%s amount=$1.00\n", market.Slug)

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
			balancePreflightCheck(conn)

			market, err := conn.GetMarket(splitMergeMarketSlug)
			Expect(err).ToNot(HaveOccurred(), "could not fetch test market")

			fmt.Fprintf(GinkgoWriter, "Round-trip: market=%s\n", market.Slug)

			// ── Split ────────────────────────────────────────────────────────────
			splitTx, ready, err := conn.SplitPosition(market, splitMergeAmount)
			Expect(err).ToNot(HaveOccurred(), "SplitPosition should succeed")
			Expect(splitTx).ToNot(BeEmpty())
			fmt.Fprintf(GinkgoWriter, "Split tx:  %s (awaiting confirmation)\n", splitTx)

			// Wait for mining + CLOB notify before merging.
			Expect(<-ready).ToNot(HaveOccurred(), "split tx should be mined and CLOB notified")
			fmt.Fprintf(GinkgoWriter, "Split confirmed\n")

			// ── Merge ────────────────────────────────────────────────────────────
			mergeTx, err := conn.MergePositions(market, splitMergeAmount)
			Expect(err).ToNot(HaveOccurred(), "MergePositions should succeed after split")
			Expect(mergeTx).ToNot(BeEmpty())
			fmt.Fprintf(GinkgoWriter, "Merge tx:  %s\n", mergeTx)

			Expect(splitTx).ToNot(Equal(mergeTx), "split and merge should be distinct transactions")
		})
	})
})
