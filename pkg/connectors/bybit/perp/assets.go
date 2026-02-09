package perp

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
)

func (b *bybit) FetchRiskFundBalance(symbol string) (*perp.RiskFundBalance, error) {
	return nil, fmt.Errorf("FetchRiskFundBalance not implemented for Bybit")
}
