package prediction_markets_test

import (
	"fmt"
	"math/big"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

var _ = Describe("Recover locked positions", func() {
	var runner *connector_test.PredictionMarketTestRunner

	BeforeEach(func() {
		if os.Getenv("POLYGON_RPC_URL") == "" {
			Skip("POLYGON_RPC_URL not set")
		}
		var err error
		runner, err = connector_test.NewPredictionMarketTestRunner(
			connector_test.GetTestPredictionMarketConnectorName(),
			connector_test.GetPredictionMarketConnectorConfig(),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	It("merges all locked YES+NO token pairs back to USDC", func() {
		conn := runner.GetPredictionMarketConnector()

		positions, err := conn.GetLockedPositions()
		Expect(err).ToNot(HaveOccurred())

		if len(positions) == 0 {
			fmt.Fprintf(GinkgoWriter, "No locked positions found — nothing to recover\n")
			return
		}

		fmt.Fprintf(GinkgoWriter, "Found %d locked position(s)\n", len(positions))

		for _, pos := range positions {
			usdc, _ := new(big.Float).Quo(
				new(big.Float).SetInt(pos.MergeableAmount),
				big.NewFloat(1_000_000),
			).Float64()
			fmt.Fprintf(GinkgoWriter, "  market %s: merging $%.2f USDC\n",
				pos.Market.MarketID, usdc)

			txHash, err := conn.MergePositions(pos.Market, pos.MergeableAmount)
			if err != nil {
				fmt.Fprintf(GinkgoWriter, "  market %s: MergePositions failed: %v\n",
					pos.Market.MarketID, err)
				continue
			}
			fmt.Fprintf(GinkgoWriter, "  market %s: tx %s\n", pos.Market.MarketID, txHash)
			time.Sleep(3 * time.Second)
		}
	})
})
