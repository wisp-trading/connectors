package prediction_markets_test

import (
	"fmt"
	"time"

	connector_test "github.com/wisp-trading/connectors/tests/integration/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// OrderPlacementParams contains parameters for placing a limit order
type OrderPlacementParams struct {
	Market     prediction.Market
	OutcomeIdx int
	Side       connector.OrderSide
	Price      numerical.Decimal
	Amount     numerical.Decimal
	Expiration time.Duration
}

// getMarketAndSubscribeOrderbook retrieves a market and subscribes to its orderbook
func getMarketAndSubscribeOrderbook(
	conn prediction.WebSocketConnector,
	marketSlug string,
) (prediction.Market, <-chan prediction.OrderBook, error) {
	market, err := conn.GetMarket(marketSlug)
	if err != nil {
		return prediction.Market{}, nil, fmt.Errorf("failed to get market: %w", err)
	}

	if len(market.Outcomes) == 0 {
		return prediction.Market{}, nil, fmt.Errorf("market has no outcomes")
	}

	// Subscribe to orderbook
	err = conn.SubscribeOrderBook(market)
	if err != nil {
		return prediction.Market{}, nil, fmt.Errorf("failed to subscribe to orderbook: %w", err)
	}

	orderbookChannels := conn.GetOrderbookChannels()
	if orderbookChannels == nil {
		return prediction.Market{}, nil, fmt.Errorf("orderbook channels is nil")
	}

	obChan, exists := orderbookChannels[market.MarketID]
	if !exists {
		return prediction.Market{}, nil, fmt.Errorf("no orderbook channel for market %s", market.Slug)
	}

	return market, obChan, nil
}

// placeLimitOrderAtPrice places a limit order with specified parameters
func placeLimitOrderAtPrice(
	conn prediction.WebSocketConnector,
	params OrderPlacementParams,
) (*connector.OrderResponse, error) {
	if params.OutcomeIdx < 0 || params.OutcomeIdx >= len(params.Market.Outcomes) {
		return nil, fmt.Errorf("invalid outcome index %d for market with %d outcomes",
			params.OutcomeIdx, len(params.Market.Outcomes))
	}

	fmt.Printf("Placing limit order: market=%s, outcome=%d, side=%s, price=%s, amount=%s\n",
		params.Market.Slug, params.OutcomeIdx, params.Side, params.Price.String(), params.Amount.String())

	order := prediction.LimitOrder{
		Outcome:    params.Market.Outcomes[params.OutcomeIdx],
		Side:       params.Side,
		Price:      params.Price,
		Amount:     params.Amount,
		Expiration: time.Now().Add(params.Expiration).Unix(),
	}

	orderResponse, err := conn.PlaceLimitOrder(order)
	if err != nil {
		return nil, fmt.Errorf("failed to place limit order: %w", err)
	}

	fmt.Printf("Order placed successfully: orderID=%s\n", orderResponse.OrderID)
	return orderResponse, nil
}

// cancelOrderAndVerify cancels an order and verifies the response
func cancelOrderAndVerify(
	conn prediction.WebSocketConnector,
	orderID string,
) (*connector.CancelResponse, error) {
	fmt.Printf("Canceling order: orderID=%s\n", orderID)

	resp, err := conn.CancelOrder(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	if resp.OrderID != orderID {
		return nil, fmt.Errorf("cancel response order ID mismatch: expected=%s, got=%s",
			orderID, resp.OrderID)
	}

	fmt.Printf("Order canceled successfully: orderID=%s\n", orderID)
	return resp, nil
}

// placeLimitOrderAtBestBid places a limit buy order at the current best bid price
// This ensures the order doesn't immediately fill
func placeLimitOrderAtBestBid(
	runner *connector_test.PredictionMarketTestRunner,
	conn prediction.WebSocketConnector,
	market prediction.Market,
	obChan <-chan prediction.OrderBook,
	amount numerical.Decimal,
	outcomeIdx int,
) (*connector.OrderResponse, error) {
	// Wait for orderbook data
	orderBook := runner.VerifyOrderBookData(obChan, 30*time.Second)
	if len(orderBook.Bids) == 0 {
		return nil, fmt.Errorf("no bids in orderbook")
	}

	bestBid := orderBook.Bids[0].Price

	params := OrderPlacementParams{
		Market:     market,
		OutcomeIdx: outcomeIdx,
		Side:       connector.OrderSideBuy,
		Price:      bestBid,
		Amount:     amount,
		Expiration: 1 * time.Hour,
	}

	return placeLimitOrderAtPrice(conn, params)
}

// placeLimitOrderAtBestAsk places a limit sell order at the current best ask price
// This ensures the order doesn't immediately fill
func placeLimitOrderAtBestAsk(
	runner *connector_test.PredictionMarketTestRunner,
	conn prediction.WebSocketConnector,
	market prediction.Market,
	obChan <-chan prediction.OrderBook,
	amount numerical.Decimal,
	outcomeIdx int,
) (*connector.OrderResponse, error) {
	// Wait for orderbook data
	orderBook := runner.VerifyOrderBookData(obChan, 30*time.Second)
	if len(orderBook.Asks) == 0 {
		return nil, fmt.Errorf("no asks in orderbook")
	}

	bestAsk := orderBook.Asks[0].Price

	params := OrderPlacementParams{
		Market:     market,
		OutcomeIdx: outcomeIdx,
		Side:       connector.OrderSideSell,
		Price:      bestAsk,
		Amount:     amount,
		Expiration: 1 * time.Hour,
	}

	return placeLimitOrderAtPrice(conn, params)
}
