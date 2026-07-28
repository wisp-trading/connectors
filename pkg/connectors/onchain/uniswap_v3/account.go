package uniswap_v3

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (u *uniswapV3) GetBalances() ([]connector.AssetBalance, error) {
	if err := u.requireInit(); err != nil {
		return nil, err
	}
	if u.cfg.AccountAddress == "" {
		return nil, fmt.Errorf("account_address not set")
	}
	u.mu.RLock()
	defer u.mu.RUnlock()

	out := make([]connector.AssetBalance, 0, len(u.tokens))
	seen := map[string]struct{}{}
	for sym, meta := range u.tokens {
		// skip address-keyed duplicates
		if strings.HasPrefix(sym, "0x") {
			continue
		}
		if _, ok := seen[meta.address.Hex()]; ok {
			continue
		}
		seen[meta.address.Hex()] = struct{}{}
		bal, err := u.tokenBalance(meta.address)
		if err != nil {
			continue
		}
		human := fromBaseUnits(bal, meta.decimals)
		out = append(out, connector.AssetBalance{
			Asset:     portfolio.NewAsset(strings.ToUpper(sym)),
			Free:      human,
			Locked:    numerical.Zero(),
			Total:     human,
			UpdatedAt: u.timeProvider.Now(),
		})
	}
	return out, nil
}

func (u *uniswapV3) GetBalance(asset portfolio.Asset) (*connector.AssetBalance, error) {
	addr, dec, ok := u.ResolveToken(asset.Symbol())
	if !ok {
		return nil, fmt.Errorf("unknown asset %s", asset.Symbol())
	}
	bal, err := u.tokenBalance(common.HexToAddress(addr))
	if err != nil {
		return nil, err
	}
	human := fromBaseUnits(bal, dec)
	return &connector.AssetBalance{
		Asset:     asset,
		Free:      human,
		Locked:    numerical.Zero(),
		Total:     human,
		UpdatedAt: u.timeProvider.Now(),
	}, nil
}

func (u *uniswapV3) GetTradingHistory(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	return nil, nil
}

func (u *uniswapV3) tokenBalance(token common.Address) (*big.Int, error) {
	if u.cfg.AccountAddress == "" {
		return nil, fmt.Errorf("account_address not set")
	}
	parsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, err
	}
	owner := common.HexToAddress(u.cfg.AccountAddress)
	data, err := parsed.Pack("balanceOf", owner)
	if err != nil {
		return nil, err
	}
	msg := ethereum.CallMsg{To: &token, Data: data}
	out, err := u.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, err
	}
	vals, err := parsed.Unpack("balanceOf", out)
	if err != nil {
		return nil, err
	}
	return vals[0].(*big.Int), nil
}

func fromBaseUnits(amount *big.Int, decimals uint8) numerical.Decimal {
	d := decimal.NewFromBigInt(amount, -int32(decimals))
	n, err := numerical.NewFromString(d.String())
	if err != nil {
		return numerical.Zero()
	}
	return n
}
