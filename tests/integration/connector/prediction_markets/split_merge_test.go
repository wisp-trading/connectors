package prediction_markets_test

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

// splitMergeAmount is the USDC amount used for split/merge tests: $1.00 in 6-decimal units.
// This is the smallest meaningful amount and keeps real on-chain costs minimal.
var splitMergeAmount = big.NewInt(1_000_000)

// splitMergeMarketSlug is a long-running market used as the test fixture.
// SplitPosition and MergePositions only require a valid condition ID; any
// active market will do. Use GetMarket (single fast API call) rather than
// paginating Markets() to avoid multi-second Gamma API delays.
const splitMergeMarketSlug = "will-jesus-christ-return-before-2027"

// usdcContractAddress is the USDC ERC-20 on Polygon mainnet.
const usdcContractAddress = "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"

// minMATICForGas is the minimum MATIC balance (in wei) required in the EOA.
// 0.01 MATIC is comfortably above what a single CTF call costs (~0.001–0.003 MATIC).
var minMATICForGas = new(big.Int).Mul(big.NewInt(10_000_000_000_000_000), big.NewInt(1)) // 0.01 MATIC

// ctfPreflightCheck dials Polygon, derives the EOA address from the private key,
// and verifies it has enough MATIC for gas and enough on-chain USDC to split.
// Returns the EOA address for logging. Calls GinkgoWriter and Skip as needed.
func ctfPreflightCheck() common.Address {
	rpcURL := os.Getenv("POLYGON_RPC_URL")
	privateKeyHex := strings.TrimPrefix(os.Getenv("POLYMARKET_PRIVATE_KEY"), "0x")

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		Skip(fmt.Sprintf("cannot dial Polygon RPC %q: %v", rpcURL, err))
	}
	defer client.Close()

	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		Skip(fmt.Sprintf("cannot parse POLYMARKET_PRIVATE_KEY: %v", err))
	}
	eoa := crypto.PubkeyToAddress(key.PublicKey)

	ctx := context.Background()

	// ── MATIC balance ────────────────────────────────────────────────────────
	maticWei, err := client.BalanceAt(ctx, eoa, nil)
	if err != nil {
		Skip(fmt.Sprintf("cannot fetch MATIC balance for EOA %s: %v", eoa.Hex(), err))
	}
	maticFloat := new(big.Float).Quo(
		new(big.Float).SetInt(maticWei),
		new(big.Float).SetInt(big.NewInt(1e18)),
	)
	fmt.Fprintf(GinkgoWriter, "EOA %s — MATIC balance: %s\n", eoa.Hex(), maticFloat.Text('f', 6))

	if maticWei.Cmp(minMATICForGas) < 0 {
		Skip(fmt.Sprintf(
			"EOA %s has only %s MATIC — needs ≥0.01 MATIC for gas. Send MATIC to this address on Polygon.",
			eoa.Hex(), maticFloat.Text('f', 6),
		))
	}

	// ── USDC balance (ERC-20 balanceOf via eth_call) ─────────────────────────
	// balanceOf(address) selector: 0x70a08231
	data := append([]byte{0x70, 0xa0, 0x82, 0x31}, common.LeftPadBytes(eoa.Bytes(), 32)...)
	usdcAddr := common.HexToAddress(usdcContractAddress)
	result, err := client.CallContract(ctx, ethereum.CallMsg{To: &usdcAddr, Data: data}, nil)
	if err == nil && len(result) == 32 {
		usdcRaw := new(big.Int).SetBytes(result)
		usdcFloat := new(big.Float).Quo(
			new(big.Float).SetInt(usdcRaw),
			new(big.Float).SetInt(big.NewInt(1_000_000)),
		)
		fmt.Fprintf(GinkgoWriter, "EOA %s — USDC balance: $%s\n", eoa.Hex(), usdcFloat.Text('f', 2))

		if usdcRaw.Cmp(splitMergeAmount) < 0 {
			Skip(fmt.Sprintf(
				"EOA %s has only $%s USDC on-chain — needs ≥$1.00. "+
					"Withdraw USDC from Polymarket to this EOA (not the Safe wallet).",
				eoa.Hex(), usdcFloat.Text('f', 2),
			))
		}
	}

	return eoa
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
			eoa := ctfPreflightCheck()
			conn := runner.GetPredictionMarketConnector()

			market, err := conn.GetMarket(splitMergeMarketSlug)
			Expect(err).ToNot(HaveOccurred(), "could not fetch test market")

			fmt.Fprintf(GinkgoWriter, "SplitPosition: EOA=%s market=%s amount=$1.00\n",
				eoa.Hex(), market.Slug)

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
			eoa := ctfPreflightCheck()
			conn := runner.GetPredictionMarketConnector()

			market, err := conn.GetMarket(splitMergeMarketSlug)
			Expect(err).ToNot(HaveOccurred(), "could not fetch test market")

			fmt.Fprintf(GinkgoWriter, "MergePositions: EOA=%s market=%s amount=$1.00\n",
				eoa.Hex(), market.Slug)

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
			eoa := ctfPreflightCheck()
			conn := runner.GetPredictionMarketConnector()

			market, err := conn.GetMarket(splitMergeMarketSlug)
			Expect(err).ToNot(HaveOccurred(), "could not fetch test market")

			fmt.Fprintf(GinkgoWriter, "Round-trip: EOA=%s market=%s\n", eoa.Hex(), market.Slug)

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
