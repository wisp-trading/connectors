package bybit

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
)

func (b *bybit) GetAccountBalance() (*connector.AccountBalance, error) {
	return b.trading.GetAccountBalance()
}

func (b *bybit) GetPositions() ([]connector.Position, error) {
	return b.trading.GetPositions()
}

func (b *bybit) GetTradingHistory(symbol string, limit int) ([]connector.Trade, error) {
	return b.trading.GetTradingHistory(symbol, limit)
}
