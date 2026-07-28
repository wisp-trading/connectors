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

	It("should register and resolve tokens on the connector", func() {
		conn := runner.GetOnchainConnector()
		baseAddr := connector_test.GetOnchainBaseToken()
		if baseAddr == "" {
			Skip("set UNISWAP_V3_BASE_TOKEN")
		}
		sym := connector_test.GetOnchainBaseSymbol()
		Expect(conn.RegisterToken(sym, baseAddr, 18)).To(Succeed())
		addr, dec, ok := conn.ResolveToken(sym)
		Expect(ok).To(BeTrue())
		Expect(addr).ToNot(BeEmpty())
		Expect(dec).To(Equal(uint8(18)))
		// WETH seeded from config when present
		if _, _, wok := conn.ResolveToken("WETH"); wok {
			connector_test.LogSuccess("WETH resolved from config")
		}
		connector_test.LogSuccess("token registry %s → %s", sym, addr)
	})

	It("should dry-run BUY into onchain store via executor and surface via SDK", func() {
		assertSwapToStore(runner, connector.OrderSideBuy)
	})

	It("should dry-run SELL into onchain store via executor and surface via SDK", func() {
		assertSwapToStore(runner, connector.OrderSideSell)
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

	It("should build signals via wisp.Onchain().Signal without executing", func() {
		baseAddr := connector_test.GetOnchainBaseToken()
		if baseAddr == "" {
			Skip("set UNISWAP_V3_BASE_TOKEN")
		}
		baseSym := connector_test.GetOnchainBaseSymbol()
		_ = runner.GetOnchainConnector().RegisterToken(baseSym, baseAddr, 18)
		pair := portfolio.NewPair(portfolio.NewAsset(baseSym), portfolio.NewAsset("WETH"))
		qty := numerical.NewFromFloat(0.001)

		sig, err := runner.GetWisp().Onchain().Signal(runner.StrategyName()).
			Buy(pair, runner.ExchangeName(), qty).
			Build()
		Expect(err).ToNot(HaveOccurred())
		Expect(sig.GetActions()).To(HaveLen(1))
		Expect(sig.GetActions()[0].Pair.Symbol()).To(Equal(pair.Symbol()))
		connector_test.LogSuccess("signal built strategy=%s actions=1", runner.StrategyName())
	})
})

func assertSwapToStore(runner *connector_test.OnchainTestRunner, side connector.OrderSide) {
	conn := runner.GetOnchainConnector()
	baseSym := connector_test.GetOnchainBaseSymbol()
	baseAddr := connector_test.GetOnchainBaseToken()
	if baseAddr == "" {
		Skip("set UNISWAP_V3_BASE_TOKEN (and optional UNISWAP_V3_BASE_SYMBOL) for swap e2e")
	}
	if _, _, ok := conn.ResolveToken("WETH"); !ok {
		Skip("UNISWAP_V3_WETH required so connector can register WETH")
	}
	Expect(conn.RegisterToken(baseSym, baseAddr, 18)).To(Succeed())
	pair := portfolio.NewPair(portfolio.NewAsset(baseSym), portfolio.NewAsset("WETH"))

	qty, err := numerical.NewFromString("0.001")
	Expect(err).ToNot(HaveOccurred())

	if q, qerr := conn.QuoteMarket(pair, side, qty); qerr == nil {
		connector_test.LogSuccess("QuoteMarket side=%s fee=%d out=%s", side, q.FeeTier, q.AmountOut.String())
	} else {
		connector_test.LogWarning("QuoteMarket soft-fail (dry_run still ok): %v", qerr)
	}

	builder := runner.GetWisp().Onchain().Signal(runner.StrategyName())
	if side == connector.OrderSideSell {
		builder = builder.Sell(pair, runner.ExchangeName(), qty)
	} else {
		builder = builder.Buy(pair, runner.ExchangeName(), qty)
	}
	sig, err := builder.Build()
	Expect(err).ToNot(HaveOccurred())

	before := len(runner.GetOnchainStore().GetOrders())
	result := &execution.ExecutionResult{OrderIDs: make([]string, 0)}
	execErr := runner.GetOnchainExecutor().ExecuteOnchainSignal(sig, &execution.ExecutionContext{}, result)
	Expect(execErr).ToNot(HaveOccurred(), "dry_run swap must succeed via onchain executor")
	Expect(result.OrderIDs).ToNot(BeEmpty())

	orders := runner.GetOnchainStore().GetOrders()
	Expect(len(orders)).To(BeNumerically(">", before), "store must gain an order after executor")

	found := false
	for _, o := range orders {
		if o.ID == result.OrderIDs[0] {
			found = true
			Expect(o.Pair.Symbol()).To(Equal(pair.Symbol()))
			Expect(o.Side).To(Equal(side))
			break
		}
	}
	Expect(found).To(BeTrue(), "store must contain exact order id from connector")

	positions := runner.GetWisp().Onchain().Positions()
	Expect(positions).ToNot(BeEmpty(), "wisp.Onchain().Positions() must surface store orders")
	connector_test.LogSuccess("✓ side=%s connector → executor → store → wisp.Onchain()", side)
}
