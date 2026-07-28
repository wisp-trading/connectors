package uniswap_v3

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// SwapRouter02 exactInputSingle
const swapRouterABI = `[
  {
    "inputs": [
      {
        "components": [
          {"internalType":"address","name":"tokenIn","type":"address"},
          {"internalType":"address","name":"tokenOut","type":"address"},
          {"internalType":"uint24","name":"fee","type":"uint24"},
          {"internalType":"address","name":"recipient","type":"address"},
          {"internalType":"uint256","name":"amountIn","type":"uint256"},
          {"internalType":"uint256","name":"amountOutMinimum","type":"uint256"},
          {"internalType":"uint160","name":"sqrtPriceLimitX96","type":"uint160"}
        ],
        "internalType":"struct IV3SwapRouter.ExactInputSingleParams",
        "name":"params",
        "type":"tuple"
      }
    ],
    "name":"exactInputSingle",
    "outputs": [{"internalType":"uint256","name":"amountOut","type":"uint256"}],
    "stateMutability":"payable",
    "type":"function"
  }
]`

const erc20ABI = `[
  {"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"},
  {"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"},
  {"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"}
]`

func (u *uniswapV3) executeSwap(tokenIn, tokenOut string, amountIn, minOut *big.Int, fee uint32) (string, error) {
	pk, err := parsePrivateKey(u.cfg.PrivateKey)
	if err != nil {
		return "", err
	}
	from := crypto.PubkeyToAddress(pk.PublicKey)
	if u.cfg.AccountAddress != "" && !strings.EqualFold(from.Hex(), u.cfg.AccountAddress) {
		return "", fmt.Errorf("private_key address %s != account_address %s", from.Hex(), u.cfg.AccountAddress)
	}

	router := common.HexToAddress(u.cfg.SwapRouter)
	if err := u.ensureAllowance(from, pk, common.HexToAddress(tokenIn), router, amountIn); err != nil {
		return "", fmt.Errorf("approve: %w", err)
	}

	parsed, err := abi.JSON(strings.NewReader(swapRouterABI))
	if err != nil {
		return "", err
	}

	type params struct {
		TokenIn           common.Address
		TokenOut          common.Address
		Fee               *big.Int
		Recipient         common.Address
		AmountIn          *big.Int
		AmountOutMinimum  *big.Int
		SqrtPriceLimitX96 *big.Int
	}
	p := params{
		TokenIn:           common.HexToAddress(tokenIn),
		TokenOut:          common.HexToAddress(tokenOut),
		Fee:               big.NewInt(int64(fee)),
		Recipient:         from,
		AmountIn:          amountIn,
		AmountOutMinimum:  minOut,
		SqrtPriceLimitX96: big.NewInt(0),
	}
	data, err := parsed.Pack("exactInputSingle", p)
	if err != nil {
		return "", fmt.Errorf("pack swap: %w", err)
	}

	hash, err := u.sendTx(from, pk, &router, big.NewInt(0), data)
	if err != nil {
		return "", err
	}
	u.tradingLogger.Info("uniswap_v3 swap sent tx=%s", hash)
	return hash, nil
}

func (u *uniswapV3) ensureAllowance(from common.Address, pk *ecdsa.PrivateKey, token, spender common.Address, need *big.Int) error {
	parsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return err
	}
	data, err := parsed.Pack("allowance", from, spender)
	if err != nil {
		return err
	}
	msg := ethereum.CallMsg{To: &token, Data: data}
	out, err := u.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return err
	}
	vals, err := parsed.Unpack("allowance", out)
	if err != nil {
		return err
	}
	allowance := vals[0].(*big.Int)
	if allowance.Cmp(need) >= 0 {
		return nil
	}
	// approve max
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	approveData, err := parsed.Pack("approve", spender, max)
	if err != nil {
		return err
	}
	_, err = u.sendTx(from, pk, &token, big.NewInt(0), approveData)
	return err
}

func (u *uniswapV3) sendTx(from common.Address, pk *ecdsa.PrivateKey, to *common.Address, value *big.Int, data []byte) (string, error) {
	ctx := context.Background()
	nonce, err := u.client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", err
	}
	gasPrice, err := u.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}
	msg := ethereum.CallMsg{From: from, To: to, Value: value, Data: data}
	gasLimit, err := u.client.EstimateGas(ctx, msg)
	if err != nil {
		// fallback
		gasLimit = 400000
		u.appLogger.Warn("gas estimate failed (%v); using %d", err, gasLimit)
	} else {
		gasLimit = gasLimit * 12 / 10 // +20%
	}

	tx := types.NewTransaction(nonce, *to, value, gasLimit, gasPrice, data)
	signer := types.LatestSignerForChainID(big.NewInt(int64(u.cfg.ChainID)))
	signed, err := types.SignTx(tx, signer, pk)
	if err != nil {
		return "", err
	}
	if err := u.client.SendTransaction(ctx, signed); err != nil {
		return "", err
	}
	return signed.Hash().Hex(), nil
}

func parsePrivateKey(hexKey string) (*ecdsa.PrivateKey, error) {
	hexKey = strings.TrimPrefix(strings.TrimSpace(hexKey), "0x")
	return crypto.HexToECDSA(hexKey)
}
