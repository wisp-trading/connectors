package order_manager

import (
	"context"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

// OrderManager wraps Polymarket CLOB API
type OrderManager interface {
	PlaceOrder(ctx context.Context, order prediction.LimitOrder) (clobtypes.OrderResponse, error)
}

// orderManager implementation
type orderManager struct {
	client clob.Client
	signer *auth.PrivateKeySigner
}

func NewOrderManager(
	client clob.Client,
	signer *auth.PrivateKeySigner,
) OrderManager {
	return &orderManager{
		client: client,
		signer: signer,
	}
}
