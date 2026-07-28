package connector

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/wisp-trading/connectors/pkg/connectors"
	onchainTypes "github.com/wisp-trading/sdk/pkg/markets/onchain/types"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	onchainconnector "github.com/wisp-trading/sdk/pkg/types/connector/onchain"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"github.com/wisp-trading/sdk/pkg/types/strategy"
	wispTypes "github.com/wisp-trading/sdk/pkg/types/wisp"
	"github.com/wisp-trading/sdk/wisp"
)

// OnchainTestRunner wires uniswap_v3 + full SDK for e2e onchain tests.
type OnchainTestRunner struct {
	*BaseRunnerImpl
	conn         onchainconnector.Connector
	exchangeName connector.ExchangeName
	wisp         wispTypes.Wisp
	store        onchainTypes.MarketStore
	executor     onchainTypes.SignalExecutor
}

// NewOnchainTestRunner starts fx, initializes the onchain connector, marks it ready.
func NewOnchainTestRunner(connectorName connector.ExchangeName, config connector.Config) (*OnchainTestRunner, error) {
	var reg registry.ConnectorRegistry
	var wispInstance wispTypes.Wisp
	var store onchainTypes.MarketStore
	var executor onchainTypes.SignalExecutor

	app := fx.New(
		wisp.Module,
		connectors.Module,
		fx.Populate(&reg, &wispInstance, &store, &executor),
		fx.NopLogger,
	)

	startCtx, startCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("failed to start fx app: %w", err)
	}

	conn, exists := reg.Onchain(connectorName)
	if !exists {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("onchain connector %s not found in registry", connectorName)
	}

	if err := conn.Initialize(config); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	if err := reg.MarkReady(connectorName); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to mark connector ready: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &OnchainTestRunner{
		BaseRunnerImpl: &BaseRunnerImpl{
			app:    app,
			ctx:    ctx,
			cancel: cancel,
			reg:    reg,
		},
		conn:         conn,
		exchangeName: connectorName,
		wisp:         wispInstance,
		store:        store,
		executor:     executor,
	}, nil
}

func (tr *OnchainTestRunner) GetOnchainConnector() onchainconnector.Connector { return tr.conn }

func (tr *OnchainTestRunner) GetBaseConnector() connector.Connector { return tr.conn }

func (tr *OnchainTestRunner) HasWebSocketSupport() bool { return false }

func (tr *OnchainTestRunner) GetWebSocketCapable() connector.WebSocketCapable { return nil }

func (tr *OnchainTestRunner) GetWisp() wispTypes.Wisp { return tr.wisp }

func (tr *OnchainTestRunner) ExchangeName() connector.ExchangeName { return tr.exchangeName }

func (tr *OnchainTestRunner) GetOnchainStore() onchainTypes.MarketStore { return tr.store }

func (tr *OnchainTestRunner) GetOnchainExecutor() onchainTypes.SignalExecutor { return tr.executor }

// StrategyName for signal builders in tests.
func (tr *OnchainTestRunner) StrategyName() strategy.StrategyName {
	return strategy.StrategyName("onchain-e2e")
}
