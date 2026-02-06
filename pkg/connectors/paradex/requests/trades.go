package requests

import (
	"context"
	"fmt"
	"time"

	"github.com/trishtzy/go-paradex/auth"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"

	"github.com/trishtzy/go-paradex/client/orders"
	"github.com/trishtzy/go-paradex/models"
)

type PlaceOrderParams struct {
	Market       string
	Side         string // "BUY" or "SELL"
	Size         string
	Price        string
	OrderType    string // "LIMIT" or "MARKET"
	ClientID     string
	ReduceOnly   bool
	Instruction  string // Optional, default "GTC"
	RecvWindow   int64  // Optional
	Stp          string // Optional
	TriggerPrice string // Optional
}

func (s *Service) PlaceOrder(ctx context.Context, params PlaceOrderParams) (*models.ResponsesOrderResp, error) {
	now := time.Now().UnixMilli()

	// Build signature params as required by Paradex
	signParams := map[string]interface{}{
		"timestamp": now,
		"market":    params.Market,
		"side":      params.Side,
		"orderType": params.OrderType,
		"size":      params.Size,
		"price":     params.Price,
	}
	if params.ClientID != "" {
		signParams["client_id"] = params.ClientID
	}
	if params.Instruction != "" {
		signParams["instruction"] = params.Instruction
	}
	if params.Stp != "" {
		signParams["stp"] = params.Stp
	}
	if params.TriggerPrice != "" {
		signParams["trigger_price"] = params.TriggerPrice
	}
	if params.RecvWindow != 0 {
		signParams["recv_window"] = params.RecvWindow
	}
	if params.ReduceOnly {
		signParams["flags"] = []string{"REDUCE_ONLY"}
	}

	orderSignature := auth.SignSNTypedData(auth.SignerParams{
		MessageType:       "order",
		DexAccountAddress: s.client.GetDexAccountAddress(),
		DexPrivateKey:     s.client.GetDexPrivateKey(),
		SysConfig:         *s.client.GetSystemConfig(),
		Params:            signParams,
	})

	orderReq := &models.RequestsOrderRequest{
		ClientID:           params.ClientID,
		Type:               struct{ models.ResponsesOrderType }{models.ResponsesOrderType(params.OrderType)},
		Side:               struct{ models.ResponsesOrderSide }{models.ResponsesOrderSide(params.Side)},
		Market:             &params.Market,
		Size:               &params.Size,
		Signature:          &orderSignature,
		SignatureTimestamp: &now,
	}
	// Only include price for LIMIT orders (omit for MARKET orders)
	if params.OrderType != "MARKET" && params.Price != "" {
		orderReq.Price = &params.Price
	}
	if params.Instruction != "" {
		orderReq.Instruction = &params.Instruction
	}
	if params.Stp != "" {
		orderReq.Stp = params.Stp
	}
	if params.TriggerPrice != "" {
		orderReq.TriggerPrice = params.TriggerPrice
	}
	if params.RecvWindow != 0 {
		orderReq.RecvWindow = params.RecvWindow
	}
	if params.ReduceOnly {
		orderReq.Flags = []models.ResponsesOrderFlag{"REDUCE_ONLY"}
	}

	orderParams := orders.NewOrdersNewParams().WithContext(ctx)
	orderParams.SetParams(orderReq)

	resp, err := s.client.API().Orders.OrdersNew(orderParams, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}
	return resp.Payload, nil
}

func (s *Service) CancelOrder(ctx context.Context, orderID string) error {
	cancelParams := orders.NewOrdersCancelParams().WithContext(ctx)
	cancelParams.SetOrderID(orderID)

	_, err := s.client.API().Orders.OrdersCancel(cancelParams, s.client.AuthWriter(ctx))
	if err != nil {
		return fmt.Errorf("failed to cancel order %s: %w", orderID, err)
	}

	return nil
}

func (s *Service) GetOrder(ctx context.Context, orderID string) (*models.ResponsesOrderResp, error) {
	orderParams := orders.NewOrdersGetParams().WithContext(ctx)
	orderParams.SetOrderID(orderID)

	resp, err := s.client.API().Orders.OrdersGet(orderParams, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get order %s: %w", orderID, err)
	}

	return resp.Payload, nil
}

func (s *Service) FetchOpenOrders(pair ...portfolio.Pair) ([]*models.ResponsesOrderResp, error) {
	ctx := context.Background()
	orderParams := orders.NewGetOpenOrdersParams().WithContext(ctx)

	if len(pair) > 0 {
		market := pair[0].Symbol()
		orderParams.SetMarket(&market)
	}

	resp, err := s.client.API().Orders.GetOpenOrders(orderParams, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	return resp.Payload.Results, nil
}
