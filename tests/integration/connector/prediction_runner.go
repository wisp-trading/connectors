package connector

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/wisp-trading/connectors/pkg/connectors"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
	"github.com/wisp-trading/sdk/pkg/types/registry"
	"github.com/wisp-trading/sdk/wisp"
)

// PredictionMarketTestRunner manages the lifecycle of prediction market connector tests
type PredictionMarketTestRunner struct {
	*BaseRunnerImpl
	conn prediction.Connector
}

// NewPredictionMarketTestRunner creates a new test runner for prediction market connectors
func NewPredictionMarketTestRunner(connectorName connector.ExchangeName, config connector.Config) (*PredictionMarketTestRunner, error) {
	var reg registry.ConnectorRegistry

	app := fx.New(
		wisp.Module,
		connectors.Module,
		fx.Populate(&reg),
		fx.NopLogger,
	)

	startCtx, startCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer startCancel()

	if err := app.Start(startCtx); err != nil {
		return nil, fmt.Errorf("failed to start fx app: %w", err)
	}

	// Get PREDICTION MARKET connector from registry
	conn, exists := reg.Prediction(connectorName)
	if !exists {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("prediction market connector %s not found in registry", connectorName)
	}

	// Initialize connector
	if err := conn.Initialize(config); err != nil {
		_ = app.Stop(context.Background())
		return nil, fmt.Errorf("failed to initialize connector: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &PredictionMarketTestRunner{
		BaseRunnerImpl: &BaseRunnerImpl{
			app:    app,
			ctx:    ctx,
			cancel: cancel,
			reg:    reg,
		},
		conn: conn,
	}, nil
}

// GetPredictionMarketConnector returns the prediction market connector instance
func (tr *PredictionMarketTestRunner) GetPredictionMarketConnector() prediction.Connector {
	return tr.conn
}

// GetBaseConnector returns the base connector for shared tests
func (tr *PredictionMarketTestRunner) GetBaseConnector() connector.Connector {
	return tr.conn // prediction.Connector embeds connector.Connector
}

// HasWebSocketSupport checks if connector supports WebSocket
func (tr *PredictionMarketTestRunner) HasWebSocketSupport() bool {
	// Check if the connector implements WebSocketCapable
	_, ok := tr.conn.(connector.WebSocketCapable)
	return ok
}

// GetWebSocketCapable returns the base WebSocket capability
func (tr *PredictionMarketTestRunner) GetWebSocketCapable() prediction.WebSocketConnector {
	wsCapable, ok := tr.conn.(prediction.WebSocketConnector)
	if !ok {
		return nil
	}
	return wsCapable
}

// VerifyOrderBookData waits for order book data from channel with timeout
func (tr *PredictionMarketTestRunner) VerifyOrderBookData(
	obChan <-chan connector.OrderBook,
	timeout time.Duration,
) connector.OrderBook {
	select {
	case ob, ok := <-obChan:
		if !ok {
			return connector.OrderBook{}
		}
		return ob
	case <-time.After(timeout):
		return connector.OrderBook{}
	}
}

// VerifyPriceChangeData waits for order book data from channel with timeout
func (tr *PredictionMarketTestRunner) VerifyPriceChangeData(
	obChan <-chan prediction.PriceChange,
	timeout time.Duration,
) (prediction.PriceChange, error) {
	select {
	case ob, ok := <-obChan:
		if !ok {
			return prediction.PriceChange{}, fmt.Errorf("price change channel closed")
		}
		return ob, nil
	case <-time.After(timeout):
		return prediction.PriceChange{}, fmt.Errorf("timed out waiting for price change data")
	}
}

func (tr *PredictionMarketTestRunner) VerifyTradeData(channel <-chan connector.Trade, duration time.Duration) (connector.Trade, error) {
	timeout := time.After(duration)

	for {
		select {
		case trade, ok := <-channel:
			if !ok {
				return connector.Trade{}, fmt.Errorf("trade channel closed")
			}
			return trade, nil
		case <-timeout:
			return connector.Trade{}, fmt.Errorf("timed out waiting for trade data")
		}
	}

}

func (tr *PredictionMarketTestRunner) VerifyOrderData(channel <-chan connector.Order, duration time.Duration) (connector.Order, error) {
	timeout := time.After(duration)

	for {
		select {
		case trade, ok := <-channel:
			if !ok {
				return connector.Order{}, fmt.Errorf("order channel closed")
			}
			return trade, nil
		case <-timeout:
			return connector.Order{}, fmt.Errorf("timed out waiting for order data")
		}
	}

}
