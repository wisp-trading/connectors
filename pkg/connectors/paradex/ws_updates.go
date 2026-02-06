package paradex

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
)

func (p *paradex) GetKlineChannels() map[string]<-chan connector.Kline {
	p.klineMu.RLock()
	defer p.klineMu.RUnlock()

	result := make(map[string]<-chan connector.Kline, len(p.klineChannels))
	for key, ch := range p.klineChannels {
		result[key] = ch
	}

	p.appLogger.Info("📊 Returning %d kline channels", len(result))
	return result
}

// GetOrderBookChannels returns all active orderbook channels
func (p *paradex) GetOrderBookChannels() map[string]<-chan connector.OrderBook {
	p.orderBookMu.RLock()
	defer p.orderBookMu.RUnlock()

	result := make(map[string]<-chan connector.OrderBook, len(p.orderBookChannels))
	for key, ch := range p.orderBookChannels {
		result[key] = ch
	}

	p.appLogger.Info("📊 Returning %d orderbook channels", len(result))
	return result
}

// TradeUpdates returns a channel for trade updates
func (p *paradex) TradeUpdates() <-chan connector.Trade {
	return p.tradeCh
}

func (p *paradex) PositionUpdates() <-chan perp.Position {
	if p.wsService == nil {
		ch := make(chan perp.Position)
		close(ch)
		return ch
	}

	// TODO: Implement actual position updates conversion
	convertedChan := make(chan perp.Position, 100)
	go p.convertPositionUpdates(convertedChan)
	return convertedChan
}

func (p *paradex) AccountBalanceUpdates() <-chan connector.AssetBalance {
	if p.wsService == nil {
		ch := make(chan connector.AssetBalance)
		close(ch)
		return ch
	}

	// TODO: Implement actual account balance updates conversion
	convertedChan := make(chan connector.AssetBalance, 100)
	go p.convertAccountBalanceUpdates(convertedChan)
	return convertedChan
}

func (p *paradex) ErrorChannel() <-chan error {
	if p.wsService == nil {
		ch := make(chan error)
		close(ch)
		return ch
	}

	return p.wsService.ErrorChannel()
}

// Stub converter methods - implement these when you need the functionality
func (p *paradex) convertPositionUpdates(out chan<- perp.Position) {
	defer close(out)
	// TODO: Convert from paradex position format to connector.Position
	// For now, just a placeholder that doesn't send anything
	<-p.wsContext.Done()
}

func (p *paradex) convertAccountBalanceUpdates(out chan<- connector.AssetBalance) {
	defer close(out)
	// TODO: Convert from paradex balance format to connector.AccountBalance
	// For now, just a placeholder that doesn't send anything
	<-p.wsContext.Done()
}

func (p *paradex) convertKlineUpdates(out chan<- connector.Kline) {
	defer close(out)

	for {
		select {
		case <-p.wsContext.Done():
			return
		case paradexKline, ok := <-p.wsService.KlineUpdates():
			if !ok {
				return
			}

			pair, err := p.PerpSymbolToPair(paradexKline.Symbol)
			if err != nil {
				p.appLogger.Error("Failed to convert symbol to pair: %v", err)
				continue
			}

			// Convert from paradex KlineUpdate to connector.Kline
			connectorKline := connector.Kline{
				Pair:        pair,
				Interval:    paradexKline.Interval,
				OpenTime:    paradexKline.OpenTime,
				Open:        paradexKline.Open,
				High:        paradexKline.High,
				Low:         paradexKline.Low,
				Close:       paradexKline.Close,
				Volume:      paradexKline.Volume,
				CloseTime:   paradexKline.CloseTime,
				QuoteVolume: 0,                            // Not available from paradex
				TradeCount:  int(paradexKline.TradeCount), // Convert int64 to int
				TakerVolume: 0,                            // Not available from paradex
			}

			select {
			case out <- connectorKline:
			case <-p.wsContext.Done():
				return
			default:
				// Channel full, drop update to prevent blocking
				p.appLogger.Debug("Dropped kline update for %s due to full channel", paradexKline.Symbol)
			}
		}
	}
}
