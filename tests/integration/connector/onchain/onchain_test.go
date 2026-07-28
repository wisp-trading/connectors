package onchain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/execution"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
)

var _ = Describe("Onchain UniV3 Connector E2E", func() {
	var runner *connector_test.OnchainTestRunner

	BeforeEach(func() {
		if !connector_test.OnchainConfigured() {
			Skip("set UNISWAP_V3_RPC_URL and UNISWAP_V3_CHAIN_ID to run onchain integration tests")
		}
		var err error
		runner, err = connector_test.NewOnchainTestRunner(
			connector_test.GetTestOnchainConnectorName(),
			connector_test.GetOnchainConnectorConfig(),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if runner != nil {
			runner.Cleanup()
		}
	})

	It("should wire wisp.Onchain() through the SDK", func() {
		w := runner.GetWisp()
		Expect(w).ToNot(BeNil())
		Expect(w.Onchain()).ToNot(BeNil())
		connector_test.LogSuccess("wisp.Onchain() accessible")
	})

	It("should register tokens and dry-run a market swap into the onchain store via executor", func() {
		conn := runner.GetOnchainConnector()
		baseSym := connector_test.GetOnchainBaseSymbol()
		baseAddr := connector_test.GetOnchainBaseToken()
		if baseAddr == "" {
			// Without a base token we still prove dry_run path using WETH/WETH is invalid;
			// require base for meaningful e2e.
			Skip("set UNISWAP_V3_BASE_TOKEN (and optional UNISWAP_V3_BASE_SYMBOL) for swap e2e")
		}

		// Register WETH (from config) + base token
		weth := ""
		// Resolve via config: connector seeds WETH on Initialize when cfg.WETH set
		if addr, _, ok := conn.ResolveToken("WETH"); ok {
			weth = addr
		}
		if weth == "" {
			Skip("UNISWAP_V3_WETH required so connector can register WETH")
		}

		Expect(conn.RegisterToken(baseSym, baseAddr, 18)).To(Succeed())
		pair := portfolio.NewPair(portfolio.NewAsset(baseSym), portfolio.NewAsset("WETH"))

		// Optional quote (needs quoter + live pool); failure soft-skips quote assertion only
		qty, err := numerical.NewFromString("0.001")
		Expect(err).ToNot(HaveOccurred())
		if q, qerr := conn.QuoteMarket(pair, connector.OrderSideBuy, qty); qerr == nil {
			Expect(q.AmountIn.IsPositive() || !q.AmountIn.IsNegative()).To(BeTrue())
			connector_test.LogSuccess("QuoteMarket fee=%d amountOut=%s", q.FeeTier, q.AmountOut.String())
		} else {
			connector_test.LogWarning("QuoteMarket skipped/failed (ok for dry_run without pool): %v", qerr)
		}

		// Strategy-facing signal → domain executor → connector PlaceMarketOrder (dry_run) → store
		sig, err := runner.GetWisp().Onchain().Signal(runner.StrategyName()).
			Buy(pair, runner.ExchangeName(), qty).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(sig).ToNot(BeNil())

		result := &execution.ExecutionResult{OrderIDs: make([]string, 0)}
		execErr := runner.GetOnchainExecutor().ExecuteOnchainSignal(sig, &execution.ExecutionContext{}, result)
		Expect(execErr).ToNot(HaveOccurred(), "dry_run swap must succeed via onchain executor")
		Expect(result.OrderIDs).ToNot(BeEmpty(), "executor must record order id from connector response")
		connector_test.LogSuccess("ExecuteOnchainSignal order_ids=%v", result.OrderIDs)

		// Store must hold the placed order (PlaceOrderAndRecord)
		orders := runner.GetOnchainStore().GetOrders()
		Expect(orders).ToNot(BeEmpty(), "onchain MarketStore must contain order after executor path")
		found := false
		for _, o := range orders {
			if o.ID == result.OrderIDs[0] {
				found = true
				Expect(o.Pair.Symbol()).To(Equal(pair.Symbol()))
				Expect(o.Side).To(Equal(connector.OrderSideBuy))
				break
			}
		}
		Expect(found).To(BeTrue(), "store must contain the exact order id returned by the connector")

		// Facade Positions surface reads the same order book
		positions := runner.GetWisp().Onchain().Positions()
		Expect(positions).ToNot(BeEmpty(), "wisp.Onchain().Positions() must surface store orders")
		connector_test.LogSuccess("✓ connector → executor → store → wisp.Onchain() verified (dry_run)")
	})

	It("should reject limit orders on the UniV3 pilot path", func() {
		conn := runner.GetOnchainConnector()
		baseAddr := connector_test.GetOnchainBaseToken()
		if baseAddr == "" {
			Skip("set UNISWAP_V3_BASE_TOKEN")
		}
		baseSym := connector_test.GetOnchainBaseSymbol()
		_ = conn.RegisterToken(baseSym, baseAddr, 18)
		pair := portfolio.NewPair(portfolio.NewAsset(baseSym), portfolio.NewAsset("WETH"))
		qty := numerical.NewFromFloat(0.001)
		price := numerical.NewFromFloat(1)

		_, err := conn.PlaceLimitOrder(pair, connector.OrderSideBuy, qty, price)
		Expect(err).To(HaveOccurred())
		connector_test.LogSuccess("limit order correctly rejected: %v", err)
	})
})
