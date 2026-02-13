package clob

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/polymarket/go-order-utils/pkg/builder"
	"github.com/polymarket/go-order-utils/pkg/model"
)

// SignOrder signs an order using Polymarket's official library
func (c *polymarketClient) SignOrder(order OrderRequest) (*model.SignedOrder, error) {
	orderBuilder := builder.NewExchangeOrderBuilderImpl(c.chainID, generateSalt)

	side := model.BUY
	if strings.ToUpper(order.Side) == "SELL" {
		side = model.SELL
	}

	orderData := &model.OrderData{
		Maker:       c.polymarketAddress,
		Taker:       "0x0000000000000000000000000000000000000000",
		Signer:      c.signerAddress,
		TokenId:     order.TokenID,
		MakerAmount: order.MakerAmount,
		TakerAmount: order.TakerAmount,
		Side:        side,
		FeeRateBps:  fmt.Sprintf("%s", order.FeeRateBps),
		Nonce:       fmt.Sprintf("%s", order.Nonce),
		Expiration:  fmt.Sprintf("%d", order.Expiration),
	}

	// Build and sign
	signedOrder, err := orderBuilder.BuildSignedOrder(c.privateKey, orderData, model.CTFExchange)
	if err != nil {
		return nil, fmt.Errorf("failed to sign order: %w", err)
	}

	return signedOrder, nil
}

// GenerateSalt generates a random salt value for orders
func generateSalt() int64 {
	// Use current timestamp + secure random value for uniqueness
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	randomValue := new(big.Int).SetBytes(randomBytes).Int64()
	if randomValue < 0 {
		randomValue = -randomValue
	}
	return time.Now().UnixNano() + (randomValue % 1000000)
}
