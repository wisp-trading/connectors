package perp

import (
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/perp"
)

// GetKlineChannels returns all kline channels
func (b *bybit) GetKlineChannels() map[string]<-chan connector.Kline {
	b.klineMu.RLock()
	defer b.klineMu.RUnlock()

	result := make(map[string]<-chan connector.Kline)
	for k, v := range b.klineChannels {
		result[k] = v
	}
	return result
}

// TradeUpdates returns the trade updates channel
func (b *bybit) TradeUpdates() <-chan connector.Trade {
	return b.tradeCh
}

// PositionUpdates returns the position updates channel
func (b *bybit) PositionUpdates() <-chan perp.Position {
	return b.positionCh
}

func (b *bybit) AccountBalanceUpdates() <-chan connector.AssetBalance {
	return b.balanceCh
}

func (b *bybit) FundingRateUpdates() <-chan perp.FundingRate {
	return b.fundingRateCh
}

// GetOrderBookChannels returns all active orderbook channels
func (b *bybit) GetOrderBookChannels() map[string]<-chan connector.OrderBook {
	b.orderBookMu.RLock()
	defer b.orderBookMu.RUnlock()

	result := make(map[string]<-chan connector.OrderBook)
	for k, v := range b.orderBookChannels {
		result[k] = v
	}
	return result
}

func (b *bybit) ErrorChannel() <-chan error {
	return b.GetErrorChannel()
}

func (b *bybit) ErrorUpdates() <-chan error {
	return b.errorCh
}

// IsWebSocketConnected returns whether the WebSocket is connected
func (b *bybit) IsWebSocketConnected() bool {
	return b.initialized
}
