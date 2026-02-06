package paradex

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/portfolio"
)

func (p *paradex) SubscribePositions(pair portfolio.Pair) error {
	// TODO: Implement position subscription
	p.appLogger.Info("Position subscription requested for %s %s - not yet implemented", pair.Symbol())
	return nil
}

func (p *paradex) UnsubscribePositions(pair portfolio.Pair) error {
	// TODO: Implement position unsubscription
	p.appLogger.Info("Position unsubscription requested for %s %s - not yet implemented", pair.Symbol())
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

func (p *paradex) SubscribeOrderBook(pair portfolio.Pair) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	symbol := p.GetPerpSymbol(pair)

	if err := p.wsService.SubscribeOrderBook(symbol); err != nil {
		return fmt.Errorf("failed to subscribe to orderbook for %s: %w", pair.Symbol(), err)
	}

	p.tradingLogger.OrderLifecycle(fmt.Sprintf("Subscribed to orderbook for %s", symbol), pair.Symbol())
	return nil
}

func (p *paradex) UnsubscribeOrderBook(pair portfolio.Pair) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	symbol := p.GetPerpSymbol(pair)

	if err := p.wsService.UnsubscribeOrderbook(symbol); err != nil {
		return fmt.Errorf("failed to unsubscribe from orderbook for %s: %w", symbol, err)
	}

	p.tradingLogger.OrderLifecycle("Unsubscribed from orderbook for %s", symbol)
	return nil
}

func (p *paradex) SubscribeTrades(pair portfolio.Pair) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	symbol := p.GetPerpSymbol(pair)

	if err := p.wsService.SubscribeTrades(symbol); err != nil {
		return fmt.Errorf("failed to subscribe to trades for %s: %w", symbol, err)
	}

	p.tradingLogger.OrderLifecycle("Subscribed to trades for %s", symbol)
	return nil
}

func (p *paradex) UnsubscribeTrades(pair portfolio.Pair) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	symbol := p.GetPerpSymbol(pair)

	if err := p.wsService.UnsubscribeTrades(symbol); err != nil {
		return fmt.Errorf("failed to unsubscribe from trades for %s: %w", symbol, err)
	}

	p.tradingLogger.OrderLifecycle("Unsubscribed from trades for %s", symbol)
	return nil
}

// SubscribeKlines todo how we fetch klines now is not accurate, we need to implement proper kline subscription based on trades
// https://docs.paradex.trade/api/prod/markets/klines
func (p *paradex) SubscribeKlines(pair portfolio.Pair, interval string) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	symbol := p.GetPerpSymbol(pair)

	if err := p.wsService.SubscribeTrades(symbol); err != nil {
		return fmt.Errorf("failed to subscribe to trades for klines %s: %w", pair.Symbol(), err)
	}

	p.appLogger.Info("Subscribed to trades for klines %s %s", pair.Symbol(), interval)
	return nil
}

func (p *paradex) UnsubscribeKlines(pair portfolio.Pair, interval string) error {
	if !p.IsWebSocketConnected() {
		return fmt.Errorf("WebSocket not connected")
	}

	p.appLogger.Info("Klines unsubscription requested for %s %s - klines will stop when trade stream stops", pair.Symbol(), interval)
	return nil
}
