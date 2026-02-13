package clob

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

func calculateAmounts(order prediction.LimitOrder) (maker, taker string) {
	price := order.Price.InexactFloat64()

	if order.Side == connector.OrderSideBuy {
		// BUY: maker gives USDC, taker gives tokens
		if order.SpendAmount != nil {
			// User said "I want to spend $X"
			usdcAmount := order.SpendAmount.InexactFloat64()
			tokensAmount := (usdcAmount * price) / (1 - price)

			maker = fmt.Sprintf("%.0f", usdcAmount*1_000_000)
			taker = fmt.Sprintf("%.0f", tokensAmount*1_000_000)
		} else {
			// User said "I want to receive Y tokens"
			tokensAmount := order.ReceiveAmount.InexactFloat64()
			usdcAmount := (tokensAmount * (1 - price)) / price

			maker = fmt.Sprintf("%.0f", usdcAmount*1_000_000)
			taker = fmt.Sprintf("%.0f", tokensAmount*1_000_000)
		}
	} else {
		// SELL: maker gives tokens, taker gives USDC
		if order.SpendAmount != nil {
			// User said "I want to sell X tokens"
			tokensAmount := order.SpendAmount.InexactFloat64()
			usdcAmount := tokensAmount * price

			maker = fmt.Sprintf("%.0f", tokensAmount*1_000_000)
			taker = fmt.Sprintf("%.0f", usdcAmount*1_000_000)
		} else {
			// User said "I want to receive $Y"
			usdcAmount := order.ReceiveAmount.InexactFloat64()
			tokensAmount := usdcAmount / price

			maker = fmt.Sprintf("%.0f", tokensAmount*1_000_000)
			taker = fmt.Sprintf("%.0f", usdcAmount*1_000_000)
		}
	}

	return maker, taker
}
