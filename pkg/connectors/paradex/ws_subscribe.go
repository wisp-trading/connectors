package paradex

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

func (p *paradex) SubscribePositions(asset portfolio.Asset, instrumentType connector.Instrument) error {
	// TODO: Implement position subscription
	p.appLogger.Info("Position subscription requested for %s %s - not yet implemented", asset.Symbol(), instrumentType)
	return nil
}

func (p *paradex) UnsubscribePositions(asset portfolio.Asset, instrumentType connector.Instrument) error {
	// TODO: Implement position unsubscription
	p.appLogger.Info("Position unsubscription requested for %s %s - not yet implemented", asset.Symbol(), instrumentType)
	return nil
}

func (p *paradex) SubscribeAccountBalance() error {
	// TODO: Implement account balance subscription
	p.appLogger.Info("Account balance subscription requested - not yet implemented")
	return nil
}

func (p *paradex) UnsubscribeAccountBalance() error {
	// TODO: Implement account balance unsubscription
	p.appLogger.Info("Account balance unsubscription requested - not yet implemented")
	return nil
}

func (p *paradex) SubscribeOrderBook(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	if instrumentType != connector.TypePerpetual {
		return fmt.Errorf("orderbook subscription only supported for perpetual contracts")
	}

	symbol := p.GetPerpSymbol(asset)

	if err := p.wsService.SubscribeOrderBook(symbol); err != nil {
		return fmt.Errorf("failed to subscribe to orderbook for %s: %w", asset.Symbol(), err)
	}

	p.tradingLogger.OrderLifecycle(fmt.Sprintf("Subscribed to orderbook for %s", symbol), asset.Symbol())
	return nil
}

func (p *paradex) UnsubscribeOrderBook(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	if instrumentType != connector.TypePerpetual {
		return fmt.Errorf("orderbook unsubscription only supported for perpetual contracts")
	}

	symbol := p.GetPerpSymbol(asset)

	if err := p.wsService.UnsubscribeOrderbook(symbol); err != nil {
		return fmt.Errorf("failed to unsubscribe from orderbook for %s: %w", symbol, err)
	}

	p.tradingLogger.OrderLifecycle("Unsubscribed from orderbook for %s", symbol)
	return nil
}

func (p *paradex) SubscribeTrades(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	if instrumentType != connector.TypePerpetual {
		return fmt.Errorf("trades subscription only supported for perpetual contracts")
	}

	symbol := p.GetPerpSymbol(asset)

	if err := p.wsService.SubscribeTrades(symbol); err != nil {
		return fmt.Errorf("failed to subscribe to trades for %s: %w", symbol, err)
	}

	p.tradingLogger.OrderLifecycle("Subscribed to trades for %s", symbol)
	return nil
}

func (p *paradex) UnsubscribeTrades(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	if instrumentType != connector.TypePerpetual {
		return fmt.Errorf("trades unsubscription only supported for perpetual contracts")
	}

	symbol := p.GetPerpSymbol(asset)

	if err := p.wsService.UnsubscribeTrades(symbol); err != nil {
		return fmt.Errorf("failed to unsubscribe from trades for %s: %w", symbol, err)
	}

	p.tradingLogger.OrderLifecycle("Unsubscribed from trades for %s", symbol)
	return nil
}

func (p *paradex) SubscribeKlines(asset portfolio.Asset, interval string) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	symbol := p.GetPerpSymbol(asset)

	if err := p.wsService.SubscribeTrades(symbol); err != nil {
		return fmt.Errorf("failed to subscribe to trades for klines %s: %w", asset.Symbol(), err)
	}

	p.appLogger.Info("Subscribed to trades for klines %s %s", asset.Symbol(), interval)
	return nil
}

func (p *paradex) UnsubscribeKlines(asset portfolio.Asset, interval string) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	// Since klines are built from trades, we could unsubscribe from trades
	// BUT: this is tricky because trades might be used for other purposes too
	// For now, just log that klines will stop when trades stop

	p.appLogger.Info("Klines unsubscription requested for %s %s - klines will stop when trade stream stops", asset.Symbol(), interval)
	return nil
}
