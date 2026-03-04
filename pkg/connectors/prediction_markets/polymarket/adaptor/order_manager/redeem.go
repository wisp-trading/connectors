package order_manager

import (
	"context"
	"math/big"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	"github.com/ethereum/go-ethereum/common"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

func (c *orderManager) RedeemPosition(ctx context.Context, market prediction.Market) (string, error) {
	usdcAddress := common.HexToAddress("0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174")
	conditionId := common.HexToHash(market.MarketID.String())
	parentCollectionId := common.Hash{} // [32]byte all zeros

	indexSets := make([]*big.Int, len(market.Outcomes))
	for i := range market.Outcomes {
		indexSets[i] = new(big.Int).Lsh(big.NewInt(1), uint(i))
	}

	request := &ctf.RedeemPositionsRequest{
		CollateralToken:    usdcAddress,
		ParentCollectionID: parentCollectionId,
		ConditionID:        conditionId,
		IndexSets:          indexSets,
	}
	response, err := c.tokenManagement.RedeemPositions(ctx, request)
	if err != nil {
		return "", err
	}

	return response.TransactionHash.Hex(), nil
}
