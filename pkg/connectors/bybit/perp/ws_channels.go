package perp

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
)

func (b *bybit) AccountBalanceUpdates() <-chan connector.AssetBalance {
	return b.balanceCh
}

func (b *bybit) PositionUpdates() <-chan perp.Position {
	return b.positionCh
}

func (b *bybit) TradeUpdates() <-chan connector.Trade {
	return b.tradeCh
}

// GetOrderBookChannels returns all active orderbook channels
func (b *bybit) GetOrderBookChannels() map[string]<-chan connector.OrderBook {
	b.orderBookMu.RLock()
	defer b.orderBookMu.RUnlock()

	result := make(map[string]<-chan connector.OrderBook, len(b.orderBookChannels))
	for key, ch := range b.orderBookChannels {
		result[key] = ch
	}

	b.appLogger.Info("📊 Returning %d orderbook channels", len(result))
	return result
}

// GetKlineChannels returns all active kline channels
func (b *bybit) GetKlineChannels() map[string]<-chan connector.Kline {
	b.klineMu.RLock()
	defer b.klineMu.RUnlock()

	result := make(map[string]<-chan connector.Kline, len(b.klineChannels))
	for key, ch := range b.klineChannels {
		result[key] = ch
	}

	b.appLogger.Info("📊 Returning %d kline channels", len(result))
	return result
}

func (b *bybit) ErrorChannel() <-chan error {
	return b.errorCh
}

func (b *bybit) ErrorUpdates() <-chan error {
	return b.errorCh
}

// IsWebSocketConnected returns whether the WebSocket is connected
func (b *bybit) IsWebSocketConnected() bool {
	return b.initialized && b.realTime != nil
}
