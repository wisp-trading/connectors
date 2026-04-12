package polymarket

import (
	"context"
	"math/big"

	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (p *polymarket) Redeem(market prediction.Market) (string, error) {
	ctx := context.Background()

	hash, err := p.orderManager.RedeemPosition(ctx, market)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func (p *polymarket) SplitPosition(market prediction.Market, amountUSDC *big.Int) (string, <-chan error, error) {
	return p.orderManager.SplitPosition(context.Background(), market, amountUSDC)
}

func (p *polymarket) MergePositions(market prediction.Market, amountUSDC *big.Int) (string, error) {
	return p.orderManager.MergePositions(context.Background(), market, amountUSDC)
}

func (p *polymarket) GetLockedPositions() ([]prediction.LockedPosition, error) {
	return p.orderManager.GetLockedPositions(context.Background())
}

func (p *polymarket) GetTokensToRedeem(market prediction.Market) ([]prediction.Balance, error) {
	ctx := context.Background()

	tokens, err := p.orderManager.GetMarketBalance(ctx, market)
	if err != nil {
		return nil, err
	}

	balances := make([]prediction.Balance, 0, len(tokens))

	for outcomeID, token := range tokens {
		balance, err := numerical.NewFromString(token.Balance)

		if err != nil {
			p.appLogger.Error("Failed to parse balance for outcome %s: %v", outcomeID, err)
			continue
		}

		balances = append(balances, prediction.Balance{
			OutcomeID: outcomeID,
			Balance:   balance,
		})
	}

	return balances, nil
}
