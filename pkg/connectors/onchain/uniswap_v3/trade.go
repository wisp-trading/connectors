package uniswap_v3

import (
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	onchainconnector "github.com/wisp-trading/sdk/pkg/types/connector/onchain"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// PlaceMarketOrder performs an exact-in UniV3 swap.
// BUY: quantity = quote spent; SELL: quantity = base sold.
func (u *uniswapV3) PlaceMarketOrder(pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*connector.OrderResponse, error) {
	if err := u.requireInit(); err != nil {
		return nil, err
	}
	if quantity.IsZero() || quantity.IsNegative() {
		return nil, fmt.Errorf("quantity must be positive")
	}

	tokenInSym, tokenOutSym := routeSymbols(pair, side)
	inAddr, inDec, ok := u.ResolveToken(tokenInSym)
	if !ok {
		return nil, fmt.Errorf("token not registered: %s (call RegisterToken)", tokenInSym)
	}
	outAddr, _, ok := u.ResolveToken(tokenOutSym)
	if !ok {
		return nil, fmt.Errorf("token not registered: %s (call RegisterToken)", tokenOutSym)
	}

	amountIn, err := toBaseUnits(quantity, inDec)
	if err != nil {
		return nil, err
	}

	quote, err := u.quoteExactIn(inAddr, outAddr, amountIn, u.cfg.DefaultFeeTier)
	if err != nil {
		// dry-run still allows a synthetic quote when RPC quote fails
		if !u.cfg.DryRun {
			return nil, fmt.Errorf("quote: %w", err)
		}
		u.appLogger.Warn("uniswap_v3 dry_run quote failed (%v); simulating fill", err)
		quote = &onchainconnector.Quote{
			AmountIn:  quantity,
			AmountOut: numerical.Zero(),
			FeeTier:   u.cfg.DefaultFeeTier,
		}
	}

	now := u.timeProvider.Now()
	orderID := "dryrun-" + uuid.NewString()

	if u.cfg.DryRun {
		u.tradingLogger.Info(
			"DRY_RUN uniswap_v3 swap side=%s pair=%s amountIn=%s fee=%d",
			side, pair.Symbol(), quantity.String(), u.cfg.DefaultFeeTier,
		)
		return &connector.OrderResponse{
			OrderID:   orderID,
			Symbol:    pair.Symbol(),
			Status:    connector.OrderStatusFilled,
			Side:      side,
			Type:      connector.OrderTypeMarket,
			Quantity:  quantity,
			FilledQty: quantity,
			AvgPrice:  numerical.Zero(),
			Timestamp: now,
		}, nil
	}

	if u.cfg.SwapRouter == "" {
		return nil, fmt.Errorf("swap_router required for live swaps")
	}

	minOut := applySlippage(quote.AmountOut, u.cfg.DefaultSlippage)
	txHash, err := u.executeSwap(inAddr, outAddr, amountIn, minOut, u.cfg.DefaultFeeTier)
	if err != nil {
		return nil, err
	}

	return &connector.OrderResponse{
		OrderID:   txHash,
		Symbol:    pair.Symbol(),
		Status:    connector.OrderStatusFilled,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		FilledQty: quantity,
		Timestamp: time.Now(),
	}, nil
}

func (u *uniswapV3) PlaceLimitOrder(pair portfolio.Pair, side connector.OrderSide, quantity, price numerical.Decimal) (*connector.OrderResponse, error) {
	return nil, fmt.Errorf("uniswap_v3: limit orders not supported in pilot (use market / exact-in)")
}

func (u *uniswapV3) CancelOrder(orderID string, pair ...portfolio.Pair) (*connector.CancelResponse, error) {
	return nil, fmt.Errorf("uniswap_v3: cancel not supported (swaps are atomic)")
}

func (u *uniswapV3) GetOpenOrders(pair ...portfolio.Pair) ([]connector.Order, error) {
	return nil, nil
}

func (u *uniswapV3) GetOrderStatus(orderID string, pair ...portfolio.Pair) (*connector.Order, error) {
	return nil, fmt.Errorf("uniswap_v3: GetOrderStatus not implemented (use receipt for %s)", orderID)
}

func (u *uniswapV3) QuoteMarket(pair portfolio.Pair, side connector.OrderSide, quantity numerical.Decimal) (*onchainconnector.Quote, error) {
	if err := u.requireInit(); err != nil {
		return nil, err
	}
	tokenInSym, tokenOutSym := routeSymbols(pair, side)
	inAddr, inDec, ok := u.ResolveToken(tokenInSym)
	if !ok {
		return nil, fmt.Errorf("token not registered: %s", tokenInSym)
	}
	outAddr, _, ok := u.ResolveToken(tokenOutSym)
	if !ok {
		return nil, fmt.Errorf("token not registered: %s", tokenOutSym)
	}
	amountIn, err := toBaseUnits(quantity, inDec)
	if err != nil {
		return nil, err
	}
	return u.quoteExactIn(inAddr, outAddr, amountIn, u.cfg.DefaultFeeTier)
}

func routeSymbols(pair portfolio.Pair, side connector.OrderSide) (tokenIn, tokenOut string) {
	base := pair.Base().Symbol()
	quote := pair.Quote().Symbol()
	if side == connector.OrderSideSell {
		return base, quote
	}
	// BUY: spend quote to buy base
	return quote, base
}

func toBaseUnits(amount numerical.Decimal, decimals uint8) (*big.Int, error) {
	// numerical.Decimal wraps shopspring; convert via string.
	d, err := decimal.NewFromString(amount.String())
	if err != nil {
		return nil, err
	}
	scale := decimal.New(1, int32(decimals))
	scaled := d.Mul(scale).Truncate(0)
	bi := new(big.Int)
	bi, ok := bi.SetString(scaled.String(), 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount units: %s", scaled.String())
	}
	return bi, nil
}

func applySlippage(amountOut numerical.Decimal, slip float64) *big.Int {
	// amountOut may be zero in dry-run fallback
	if amountOut.IsZero() {
		return big.NewInt(0)
	}
	d, err := decimal.NewFromString(amountOut.String())
	if err != nil {
		return big.NewInt(0)
	}
	// minOut = amountOut * (1 - slip)
	factor := decimal.NewFromFloat(1.0 - slip)
	min := d.Mul(factor).Truncate(0)
	bi := new(big.Int)
	bi, _ = bi.SetString(min.String(), 10)
	if bi == nil {
		return big.NewInt(0)
	}
	return bi
}
