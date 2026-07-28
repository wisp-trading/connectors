package connector

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

// CreatePair creates a portfolio.Pair for testing (base-USDT).
func CreatePair(symbol string) portfolio.Pair {
	base := portfolio.NewAsset(symbol)
	quote := portfolio.NewAsset("USDT")
	return portfolio.NewPair(base, quote)
}

// MarketDataBehavior defines shared market data tests for spot/perp.
//
// Critical path asserted here:
//
//	connector.Fetch* returns data
//	→ WatchPair + CollectNow (batch ingestor)
//	→ MarketStore holds the data
//	→ wisp.Spot()/Perp() facade reads the same values
//
// Earlier versions only checked connector return values while claiming "populate store".
func MarketDataBehavior(getRunner func() PairMarketTestRunner, getPair func() portfolio.Pair) {

	Describe("Market Data (Shared)", func() {

		Context("FetchPrice → store → SDK", func() {
			It("should fetch current price, persist via ingestor, and serve via SDK", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				pair := getPair()
				exchange := runner.ExchangeName()

				// 1) Raw connector works
				price, err := conn.FetchPrice(pair)
				Expect(err).ToNot(HaveOccurred())
				Expect(price).ToNot(BeNil())
				Expect(price.Price.IsPositive()).To(BeTrue())
				LogSuccess("Connector FetchPrice %s@%s = %s", pair.Symbol(), exchange, price.Price.String())

				// 2) Ingestor path: watch + collect into store
				runner.WatchPair(pair)
				runner.CollectNow()

				// 3) Store must hold the price
				stored := runner.StorePrice(pair)
				Expect(stored).ToNot(BeNil(), "MarketStore must contain price after CollectNow — ingestor must write connector data")
				Expect(stored.Price.IsPositive()).To(BeTrue(), "stored price must be positive")
				LogSuccess("Store GetPairPrice %s@%s = %s", pair.Symbol(), exchange, stored.Price.String())

				// 4) SDK facade must read the same store
				sdkPrice, found := runner.SDKPrice(pair)
				Expect(found).To(BeTrue(), "SDK Price() must find store data — connector → ingestor → store → facade")
				Expect(sdkPrice.IsPositive()).To(BeTrue())
				Expect(sdkPrice.Equal(stored.Price)).To(BeTrue(),
					"SDK price %s must equal store price %s", sdkPrice.String(), stored.Price.String())
				LogSuccess("SDK facade Price() = %s (matches store)", sdkPrice.String())
			})
		})

		Context("FetchKlines → store → SDK", func() {
			It("should fetch klines, persist via ingestor, and serve via SDK", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				pair := getPair()

				klines, err := conn.FetchKlines(pair, "1m", 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(klines).ToNot(BeEmpty())
				for _, k := range klines {
					Expect(k.Open > 0 || k.Close > 0).To(BeTrue(), "kline OHLC should be populated")
				}
				LogSuccess("Connector FetchKlines %s: %d bars", pair.Symbol(), len(klines))

				runner.WatchPair(pair)
				runner.CollectNow()

				stored := runner.StoreKlines(pair, "1m", 10)
				Expect(stored).ToNot(BeEmpty(), "MarketStore must contain klines after CollectNow")
				LogSuccess("Store GetKlines 1m: %d bars", len(stored))

				sdkKlines := runner.SDKKlines(pair, "1m", 10)
				Expect(sdkKlines).ToNot(BeEmpty(), "SDK Klines() must return store data")
				// Same series length (or store may retain more from default limits — at least non-empty match)
				Expect(len(sdkKlines)).To(BeNumerically(">=", 1))
				LogSuccess("SDK facade Klines() = %d bars", len(sdkKlines))
			})
		})

		Context("FetchOrderBook → store → SDK", func() {
			It("should fetch order book, persist via ingestor, and serve via SDK", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				pair := getPair()

				ob, err := conn.FetchOrderBook(pair, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(ob).ToNot(BeNil())
				Expect(ob.Bids).ToNot(BeEmpty())
				Expect(ob.Asks).ToNot(BeEmpty())
				LogSuccess("Connector FetchOrderBook: %d bids, %d asks", len(ob.Bids), len(ob.Asks))

				runner.WatchPair(pair)
				runner.CollectNow()

				stored := runner.StoreOrderBook(pair)
				Expect(stored).ToNot(BeNil(), "MarketStore must contain order book after CollectNow")
				Expect(stored.Bids).ToNot(BeEmpty(), "stored bids must be non-empty")
				Expect(stored.Asks).ToNot(BeEmpty(), "stored asks must be non-empty")
				LogSuccess("Store GetOrderBook: %d bids, %d asks", len(stored.Bids), len(stored.Asks))

				sdkOB, found := runner.SDKOrderBook(pair)
				Expect(found).To(BeTrue(), "SDK OrderBook() must find store data")
				Expect(sdkOB.Bids).ToNot(BeEmpty())
				Expect(sdkOB.Asks).ToNot(BeEmpty())
				LogSuccess("SDK facade OrderBook() = %d bids, %d asks", len(sdkOB.Bids), len(sdkOB.Asks))
			})
		})

		Context("FetchRecentTrades", func() {
			// Note: batch pair ingestors do not currently write recent trades into MarketStore.
			// This test only asserts the connector API; do not claim store population here.
			It("should fetch recent trades from the connector", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.MarketDataReader)
				pair := getPair()

				trades, err := conn.FetchRecentTrades(pair, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(trades).ToNot(BeNil())
				for _, trade := range trades {
					Expect(trade.Price.IsPositive()).To(BeTrue(), "Trade price should be positive")
					Expect(trade.Quantity.IsPositive()).To(BeTrue(), "Trade quantity should be positive")
				}
				LogSuccess("Connector FetchRecentTrades: %d trades (store path not wired for trades)", len(trades))
			})
		})
	})
}

// AccountBehavior defines shared account tests (connector-level only).
// Account balances are not written into the pair MarketStore by batch ingestors.
func AccountBehavior(getRunner func() BaseTestRunner) {

	Describe("Account Data (Shared)", func() {

		Context("GetAccountBalance", func() {
			It("should fetch account balance from the connector", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector().(connector.AccountReader)

				balance, err := conn.GetBalance(portfolio.NewAsset("USDC"))
				Expect(err).ToNot(HaveOccurred())
				Expect(balance).ToNot(BeNil())
				Expect(balance.Asset.Symbol()).ToNot(BeEmpty())

				LogSuccess("Account Balance: %s %s", balance.Total.String(), balance.Asset.Symbol())
				Expect(balance.Total.String()).ToNot(BeEmpty(), "Total balance should be set")
				Expect(balance.Free.String()).ToNot(BeEmpty(), "Free balance should be set")
			})
		})
	})
}

// WebSocketLifecycleBehavior defines shared WebSocket lifecycle tests.
func WebSocketLifecycleBehavior(getRunner func() BaseTestRunner) {

	Describe("WebSocket Lifecycle (Shared)", func() {

		Context("StartWebSocket", func() {
			It("should establish connection", func() {
				runner := getRunner()
				if !runner.HasWebSocketSupport() {
					Skip("Connector does not support WebSocket")
				}

				wsConn := runner.GetWebSocketCapable()
				err := wsConn.StartWebSocket()
				Expect(err).ToNot(HaveOccurred())

				Eventually(wsConn.IsWebSocketConnected, "10s", "500ms").
					Should(BeTrue(), "WebSocket should connect")

				LogSuccess("WebSocket connected")
			})
		})

		Context("StopWebSocket", func() {
			It("should disconnect cleanly", func() {
				runner := getRunner()
				if !runner.HasWebSocketSupport() {
					Skip("Connector does not support WebSocket")
				}

				wsConn := runner.GetWebSocketCapable()
				err := wsConn.StartWebSocket()
				Expect(err).ToNot(HaveOccurred())
				Eventually(wsConn.IsWebSocketConnected, "10s").Should(BeTrue())

				err = wsConn.StopWebSocket()
				Expect(err).ToNot(HaveOccurred())

				Eventually(wsConn.IsWebSocketConnected, "5s", "500ms").
					Should(BeFalse(), "WebSocket should disconnect")

				LogSuccess("WebSocket disconnected")
			})
		})
	})
}

// OptionsBehavior defines shared options market data test behaviors (placeholder readiness).
func OptionsBehavior(getRunner func() BaseTestRunner, getContract func() interface{}) {

	Describe("Options Market Data (Shared)", func() {

		Context("FetchMarkPrice", func() {
			It("should fetch current mark price", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for mark price fetch")
			})
		})

		Context("FetchGreeks", func() {
			It("should fetch Greeks (delta, gamma, theta, vega, rho)", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for Greeks fetch")
			})
		})

		Context("FetchImpliedVolatility", func() {
			It("should fetch implied volatility", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for IV fetch")
			})
		})

		Context("FetchUnderlyingPrice", func() {
			It("should fetch underlying asset price", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for underlying price fetch")
			})
		})

		Context("FetchExpirations", func() {
			It("should list available expiration dates", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for expiration fetch")
			})
		})

		Context("FetchStrikes", func() {
			It("should list available strikes for expiration", func() {
				runner := getRunner()
				conn := runner.GetBaseConnector()
				Expect(conn).NotTo(BeNil())
				LogSuccess("Options connector ready for strikes fetch")
			})
		})
	})
}
