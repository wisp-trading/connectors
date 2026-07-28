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
	onchainconnector "github.com/wisp-trading/sdk/pkg/types/connector/onchain"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// QuoterV2 quoteExactInputSingle(params) returns (amountOut, sqrtPriceX96After, initializedTicksCrossed, gasEstimate)
// params: (tokenIn, tokenOut, amountIn, fee, sqrtPriceLimitX96)
const quoterV2ABI = `[
  {
    "inputs": [
      {
        "components": [
          {"internalType":"address","name":"tokenIn","type":"address"},
          {"internalType":"address","name":"tokenOut","type":"address"},
          {"internalType":"uint256","name":"amountIn","type":"uint256"},
          {"internalType":"uint24","name":"fee","type":"uint24"},
          {"internalType":"uint160","name":"sqrtPriceLimitX96","type":"uint160"}
        ],
        "internalType":"struct IQuoterV2.QuoteExactInputSingleParams",
        "name":"params",
        "type":"tuple"
      }
    ],
    "name":"quoteExactInputSingle",
    "outputs": [
      {"internalType":"uint256","name":"amountOut","type":"uint256"},
      {"internalType":"uint160","name":"sqrtPriceX96After","type":"uint160"},
      {"internalType":"uint32","name":"initializedTicksCrossed","type":"uint32"},
      {"internalType":"uint256","name":"gasEstimate","type":"uint256"}
    ],
    "stateMutability":"nonpayable",
    "type":"function"
  }
]`

func (u *uniswapV3) quoteExactIn(tokenIn, tokenOut string, amountIn *big.Int, fee uint32) (*onchainconnector.Quote, error) {
	if u.cfg.Quoter == "" {
		return nil, fmt.Errorf("quoter address not configured")
	}

	parsed, err := abi.JSON(strings.NewReader(quoterV2ABI))
	if err != nil {
		return nil, err
	}

	type params struct {
		TokenIn           common.Address
		TokenOut          common.Address
		AmountIn          *big.Int
		Fee               *big.Int
		SqrtPriceLimitX96 *big.Int
	}
	p := params{
		TokenIn:           common.HexToAddress(tokenIn),
		TokenOut:          common.HexToAddress(tokenOut),
		AmountIn:          amountIn,
		Fee:               big.NewInt(int64(fee)),
		SqrtPriceLimitX96: big.NewInt(0),
	}

	data, err := parsed.Pack("quoteExactInputSingle", p)
	if err != nil {
		return nil, fmt.Errorf("pack quote: %w", err)
	}

	to := common.HexToAddress(u.cfg.Quoter)
	msg := ethereum.CallMsg{To: &to, Data: data}
	out, err := u.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		// try alternate fee tiers
		for _, alt := range []uint32{10000, 3000, 500, 100} {
			if alt == fee {
				continue
			}
			p.Fee = big.NewInt(int64(alt))
			data, err2 := parsed.Pack("quoteExactInputSingle", p)
			if err2 != nil {
				continue
			}
			msg.Data = data
			out, err = u.client.CallContract(context.Background(), msg, nil)
			if err == nil {
				fee = alt
				break
			}
		}
		if err != nil {
			return nil, fmt.Errorf("quoter call: %w", err)
		}
	}

	values, err := parsed.Unpack("quoteExactInputSingle", out)
	if err != nil {
		return nil, fmt.Errorf("unpack quote: %w", err)
	}
	amountOut, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected quote amountOut type %T", values[0])
	}

	// Store raw base units as decimal integers (strategy converts via token decimals).
	inDec := decimal.NewFromBigInt(amountIn, 0)
	outDec := decimal.NewFromBigInt(amountOut, 0)
	amountInNum, err := numerical.NewFromString(inDec.String())
	if err != nil {
		return nil, err
	}
	amountOutNum, err := numerical.NewFromString(outDec.String())
	if err != nil {
		return nil, err
	}

	return &onchainconnector.Quote{
		AmountIn:  amountInNum,
		AmountOut: amountOutNum,
		FeeTier:   fee,
	}, nil
}
