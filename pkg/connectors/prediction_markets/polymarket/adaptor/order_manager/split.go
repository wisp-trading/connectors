package order_manager

import (
	"context"
	"math/big"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	"github.com/ethereum/go-ethereum/common"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

const usdcAddressHex = "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"

// SplitPosition deposits amountUSDC into the CTF contract and mints
// 1 YES token + 1 NO token per USDC unit.
// amountUSDC is in raw USDC units (6 decimals): $1.00 = 1_000_000.
func (c *orderManager) SplitPosition(ctx context.Context, market prediction.Market, amountUSDC *big.Int) (string, error) {
	resp, err := c.tokenManagement.SplitPosition(ctx, &ctf.SplitPositionRequest{
		CollateralToken:    common.HexToAddress(usdcAddressHex),
		ParentCollectionID: common.Hash{},
		ConditionID:        common.HexToHash(market.MarketID.String()),
		Partition:          ctf.BinaryPartition,
		Amount:             amountUSDC,
	})
	if err != nil {
		return "", err
	}
	return resp.TransactionHash.Hex(), nil
}

// MergePositions burns amountUSDC worth of YES + NO tokens and returns USDC.
// amountUSDC is in raw USDC units (6 decimals): $1.00 = 1_000_000.
func (c *orderManager) MergePositions(ctx context.Context, market prediction.Market, amountUSDC *big.Int) (string, error) {
	resp, err := c.tokenManagement.MergePositions(ctx, &ctf.MergePositionsRequest{
		CollateralToken:    common.HexToAddress(usdcAddressHex),
		ParentCollectionID: common.Hash{},
		ConditionID:        common.HexToHash(market.MarketID.String()),
		Partition:          ctf.BinaryPartition,
		Amount:             amountUSDC,
	})
	if err != nil {
		return "", err
	}
	return resp.TransactionHash.Hex(), nil
}
