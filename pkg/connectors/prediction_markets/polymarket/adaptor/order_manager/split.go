package order_manager

import (
	"context"
	"fmt"
	"math/big"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	"github.com/ethereum/go-ethereum/common"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func maxUint256() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

// usdcAddressHex is the native USDC ERC-20 on Polygon mainnet (chain 137).
// Polymarket migrated from the bridged USDC.e (0x2791…) to native USDC in 2024.
const usdcAddressHex = "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"

// SplitPosition ensures the CTF contract is approved to spend USDC, then
// deposits amountUSDC and mints 1 YES token + 1 NO token per USDC unit.
// amountUSDC is in raw USDC units (6 decimals): $1.00 = 1_000_000.
func (c *orderManager) SplitPosition(ctx context.Context, market prediction.Market, amountUSDC *big.Int) (string, error) {
	usdcAddr := common.HexToAddress(usdcAddressHex)

	// Ensure the CTF contract has sufficient ERC-20 allowance before submitting
	// splitPosition. Without this the on-chain call reverts with "transfer amount
	// exceeds balance" even when the EOA holds enough USDC.
	if err := c.tokenManagement.EnsureCollateralApproved(ctx, usdcAddr, amountUSDC); err != nil {
		return "", fmt.Errorf("approve USDC for CTF: %w", err)
	}

	resp, err := c.tokenManagement.SplitPosition(ctx, &ctf.SplitPositionRequest{
		CollateralToken:    usdcAddr,
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
