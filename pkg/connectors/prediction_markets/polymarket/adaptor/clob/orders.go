package clob

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

// PlaceOrder places an order on Polymarket
func (c *polymarketClient) PlaceOrder(ctx context.Context, limitOrder prediction.LimitOrder) error {
	if !c.IsConfigured() {
		return fmt.Errorf("client not configured")
	}

	side := "BUY"
	if limitOrder.Side == connector.OrderSideSell {
		side = "SELL"
	}

	// Calculate maker/taker amounts
	// For BUY: spend USDC (taker), receive tokens (maker)
	// For SELL: spend tokens (maker), receive USDC (taker)
	var makerAmount, takerAmount decimal.Decimal
	if side == "BUY" {
		// Receive tokens
		makerAmount = decimal.NewFromFloat(limitOrder.ReceiveAmount.InexactFloat64())
		// Spend USDC = tokens * price
		priceDecimal := decimal.NewFromFloat(limitOrder.Price.InexactFloat64())
		takerAmount = makerAmount.Mul(priceDecimal)
	} else {
		// Spend tokens
		makerAmount = decimal.NewFromFloat(limitOrder.SpendAmount.InexactFloat64())
		// Receive USDC = tokens * price
		priceDecimal := decimal.NewFromFloat(limitOrder.Price.InexactFloat64())
		takerAmount = makerAmount.Mul(priceDecimal)
	}

	// Generate salt (max 53 bits for JSON safety)
	salt, err := rand.Int(rand.Reader, big.NewInt(1<<53))
	if err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	sigType := 2

	outcomeIdInt64, err := strconv.ParseInt(limitOrder.Outcome.OutcomeId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid outcome ID: %w", err)
	}

	order := &clobtypes.Order{
		Salt:          types.U256{Int: salt},
		Maker:         c.polymarketAddress,
		Signer:        c.signerAddress,
		Taker:         common.HexToAddress("0x0000000000000000000000000000000000000000"),
		TokenID:       types.U256{Int: big.NewInt(outcomeIdInt64)},
		MakerAmount:   makerAmount,
		TakerAmount:   takerAmount,
		Side:          side,
		Expiration:    types.U256{Int: big.NewInt(limitOrder.Expiration)},
		Nonce:         types.U256{Int: big.NewInt(0)},
		FeeRateBps:    decimal.Zero,
		SignatureType: &sigType,
	}

	resp, err := c.client.CLOB.CreateOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	fmt.Printf("Order placed: %v\n", resp)
	return nil
}
