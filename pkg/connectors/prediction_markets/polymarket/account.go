package polymarket

import (
	"context"
	"math/big"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

var (
	usdcDecimals = numerical.NewFromInt(1_000_000)            // 6 decimals
	maticDecimals = new(big.Int).SetUint64(1_000_000_000_000_000_000) // 18 decimals
)

func (p *polymarket) GetBalances() ([]connector.AssetBalance, error) {
	ctx := context.Background()
	now := time.Now()
	var balances []connector.AssetBalance

	// USDC balance
	usdcResp, err := p.orderManager.GetBalance(ctx)
	if err != nil {
		return nil, err
	}
	usdcRaw, err := numerical.NewFromString(usdcResp.Balance)
	if err != nil {
		return nil, err
	}
	balances = append(balances, connector.AssetBalance{
		Asset:     portfolio.NewAsset("USDC"),
		Free:      usdcRaw.Div(usdcDecimals),
		UpdatedAt: now,
	})

	// MATIC balance (native gas token)
	maticWei, err := p.orderManager.GetNativeBalance(ctx)
	if err != nil {
		p.appLogger.Error("Failed to fetch MATIC balance: %v", err)
	} else if maticWei != nil && maticWei.Sign() > 0 {
		maticFloat := new(big.Float).Quo(
			new(big.Float).SetInt(maticWei),
			new(big.Float).SetInt(maticDecimals),
		)
		maticDec, _ := numerical.NewFromString(maticFloat.Text('f', 18))
		balances = append(balances, connector.AssetBalance{
			Asset:     portfolio.NewAsset("MATIC"),
			Free:      maticDec,
			UpdatedAt: now,
		})
	}

	return balances, nil
}

func (p *polymarket) GetBalance(asset portfolio.Asset) (*connector.AssetBalance, error) {
	balances, err := p.GetBalances()
	if err != nil {
		return nil, err
	}

	for _, b := range balances {
		if b.Asset.Symbol() == asset.Symbol() {
			return &b, nil
		}
	}

	return &connector.AssetBalance{
		Asset: asset,
		Free:  numerical.NewFromInt(0),
	}, nil
}

