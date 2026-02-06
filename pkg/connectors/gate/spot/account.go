package spot

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/gate/gateapi-go/v7"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

func (g *gateSpot) GetBalance(asset portfolio.Asset) (*connector.AssetBalance, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()

	accountResponse, _, err := client.SpotApi.ListSpotAccounts(ctx, &gateapi.ListSpotAccountsOpts{
		Currency: optional.NewString(asset.Symbol()),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get spot account for asset %s: %w", asset, err)
	}

	if len(accountResponse) == 0 {
		return nil, fmt.Errorf("no account found for asset %s", asset)
	}

	account := accountResponse[0]

	available, _ := numerical.NewFromString(account.Available)
	locked, _ := numerical.NewFromString(account.Locked)
	total := available.Add(locked)

	return &connector.AssetBalance{
		Asset:     portfolio.NewAsset(account.Currency),
		Free:      available,
		Locked:    locked,
		Total:     total,
		UpdatedAt: g.timeProvider.Now(),
	}, nil
}

// GetBalances retrieves the account balance for spot trading
func (g *gateSpot) GetBalances() ([]connector.AssetBalance, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()

	accounts, _, err := client.SpotApi.ListSpotAccounts(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get spot accounts: %w", err)
	}

	var balances []connector.AssetBalance

	for _, account := range accounts {
		available, _ := numerical.NewFromString(account.Available)
		locked, _ := numerical.NewFromString(account.Locked)
		total := available.Add(locked)

		// Skip zero balances
		if total.IsZero() {
			continue
		}

		balance := connector.AssetBalance{
			Asset:     portfolio.NewAsset(account.Currency),
			Free:      available,
			Locked:    locked,
			Total:     total,
			UpdatedAt: g.timeProvider.Now(),
		}

		balances = append(balances, balance)
	}

	return balances, nil
}

// GetOpenOrders retrieves all open orders
func (g *gateSpot) GetOpenOrders(pair ...portfolio.Pair) ([]connector.Order, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()

	orders, _, err := client.SpotApi.ListAllOpenOrders(ctx, &gateapi.ListAllOpenOrdersOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	var connectorOrders []connector.Order
	for _, order := range orders {
		for _, order := range order.Orders {
			connectorOrder := g.convertGateOrderToConnector(&order)
			connectorOrders = append(connectorOrders, connectorOrder)

		}
	}

	return connectorOrders, nil
}

// GetTradingHistory retrieves trading history for a specific pair
func (g *gateSpot) GetTradingHistory(pair portfolio.Pair, limit int) ([]connector.Trade, error) {
	if !g.initialized {
		return nil, fmt.Errorf("connector not initialized")
	}

	client, err := g.spotClient.GetSpotApi()
	if err != nil {
		return nil, fmt.Errorf("failed to get spot API client: %w", err)
	}

	ctx := g.spotClient.GetAPIContext()

	currencyPair := g.GetSpotSymbol(pair)

	trades, _, err := client.SpotApi.ListMyTrades(ctx, &gateapi.ListMyTradesOpts{
		CurrencyPair: optional.NewString(currencyPair),
		Limit:        optional.NewInt32(int32(limit)),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get trading history: %w", err)
	}

	connectorTrades := make([]connector.Trade, 0, len(trades))
	for _, trade := range trades {
		connectorTrade := g.convertGateTradeToConnector(&trade)
		connectorTrades = append(connectorTrades, connectorTrade)
	}

	return connectorTrades, nil
}
