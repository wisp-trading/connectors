package perp

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/bybit/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// StartWebSocket starts the WebSocket connection for real-time data
func (b *bybit) StartWebSocket() error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if b.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	// Pass the WebSocket URL from config
	return b.wsService.Connect(b.config.WebSocketURL)
}

// StopWebSocket stops the WebSocket connection
func (b *bybit) StopWebSocket() error {
	if b.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	return b.wsService.Disconnect()
}

func (b *bybit) Connect() error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	return b.StartWebSocket()
}

func (b *bybit) Disconnect() error {
	return b.StopWebSocket()
}

func (b *bybit) SubscribeFundingRates(pair portfolio.Pair) error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if b.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	symbol := b.GetPerpSymbol(pair)

	// TODO: Implement funding rate subscription when Bybit WebSocket supports it
	// For now, return nil to indicate no error but not implemented
	b.appLogger.Warn("Funding rate WebSocket subscription not yet implemented for Bybit", "symbol", symbol)
	return nil
}

func (b *bybit) UnsubscribeFundingRates(pair portfolio.Pair) error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if b.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	symbol := b.GetPerpSymbol(pair)

	// TODO: Implement funding rate unsubscription when Bybit WebSocket supports it
	b.appLogger.Warn("Funding rate WebSocket unsubscription not yet implemented for Bybit", "symbol", symbol)
	return nil
}

func (b *bybit) SubscribeOrderBook(pair portfolio.Pair) error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if b.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	// Create channel for this subscription
	ch := make(chan connector.OrderBook, 100)
	symbol := b.GetPerpSymbol(pair)

	// Subscribe to WebSocket
	_, err := b.wsService.SubscribeToOrderBook(symbol, func(msg *websocket.OrderBookMessage) {
		// Convert to connector.OrderBook
		bids := make([]connector.PriceLevel, 0, len(msg.Bids))
		for _, bid := range msg.Bids {
			if len(bid) >= 2 {
				price, _ := numerical.NewFromString(bid[0])
				qty, _ := numerical.NewFromString(bid[1])
				bids = append(bids, connector.PriceLevel{
					Price:    price,
					Quantity: qty,
				})
			}
		}

		asks := make([]connector.PriceLevel, 0, len(msg.Asks))
		for _, ask := range msg.Asks {
			if len(ask) >= 2 {
				price, _ := numerical.NewFromString(ask[0])
				qty, _ := numerical.NewFromString(ask[1])
				asks = append(asks, connector.PriceLevel{
					Price:    price,
					Quantity: qty,
				})
			}
		}

		orderBook := connector.OrderBook{
			Pair:      pair,
			Bids:      bids,
			Asks:      asks,
			Timestamp: b.timeProvider.Now(),
		}

		select {
		case ch <- orderBook:
		default:
			b.appLogger.Warn("Order book channel full, dropping message")
		}
	})

	if err != nil {
		close(ch)
		return err
	}

	// Store channel
	b.orderBookMu.Lock()
	b.orderBookChannels[pair.Symbol()] = ch
	b.orderBookMu.Unlock()

	return nil
}

func (b *bybit) UnsubscribeOrderBook(pair portfolio.Pair) error {
	// Remove from WebSocket subscriptions and close the channel
	b.orderBookMu.Lock()
	if ch, exists := b.orderBookChannels[pair.Symbol()]; exists {
		close(ch)
		delete(b.orderBookChannels, pair.Symbol())
	}
	b.orderBookMu.Unlock()

	return nil
}

func (b *bybit) SubscribeTrades(pair portfolio.Pair) error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if b.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	symbol := b.GetPerpSymbol(pair)

	_, err := b.wsService.SubscribeToTrades(symbol, func(trades []websocket.TradeMessage) {
		for _, trade := range trades {
			parsedTrade := b.parseTrade(map[string]interface{}{
				"side":  string(trade.Side),
				"price": trade.Price,
				"size":  trade.Quantity,
			}, pair)

			select {
			case b.tradeCh <- parsedTrade:
			default:
				b.appLogger.Warn("Trade channel full, dropping message")
			}
		}
	})

	return err
}

func (b *bybit) UnsubscribeTrades(pair portfolio.Pair) error {
	// TODO: Implement proper unsubscription with subscription ID tracking
	return nil
}

func (b *bybit) SubscribePositions(pair portfolio.Pair) error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if b.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	_, err := b.wsService.SubscribeToPositions(func(msg *websocket.PositionMessage) {
		// Parse single position message
		posData := map[string]interface{}{
			"symbol":         msg.Symbol,
			"side":           msg.Side,
			"size":           msg.Size,
			"avgPrice":       msg.EntryPrice,
			"markPrice":      msg.MarkPrice,
			"unrealisedPnl":  msg.UnrealizedPnL,
			"cumRealisedPnl": msg.RealizedPnL,
			"leverage":       msg.Leverage,
			"liqPrice":       msg.LiquidationPrice,
		}

		pos := b.parsePosition(posData)
		select {
		case b.positionCh <- pos:
		default:
			b.appLogger.Warn("Position channel full, dropping message")
		}
	})

	return err
}

func (b *bybit) SubscribeAccountBalance() error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if b.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	_, err := b.wsService.SubscribeToAccountBalance(func(msg *websocket.AccountBalanceMessage) {
		// AccountBalanceMessage doesn't have detailed coin breakdown in this format
		// This is aggregate wallet data - would need to subscribe to wallet.update for coin details
		// For now, log that we received a balance update
		b.appLogger.Debug("Account balance update received",
			"totalEquity", msg.TotalEquity,
			"totalAvailableBalance", msg.TotalAvailableBalance)
	})

	return err
}

func (b *bybit) UnsubscribeAccountBalance() error {
	// TODO: Implement proper unsubscription with subscription ID tracking
	return nil
}

func (b *bybit) UnsubscribePositions(pair portfolio.Pair) error {
	// TODO: Implement proper unsubscription with subscription ID tracking
	return nil
}

func (b *bybit) SubscribeKlines(pair portfolio.Pair, interval string) error {
	if !b.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if b.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	ch := make(chan connector.Kline, 100)
	symbol := b.GetPerpSymbol(pair)

	_, err := b.wsService.SubscribeToKlines(symbol, interval, func(msg *websocket.KlineMessage) {
		// Convert string values to float64
		open, _ := numerical.NewFromString(msg.Open)
		high, _ := numerical.NewFromString(msg.High)
		low, _ := numerical.NewFromString(msg.Low)
		closePrice, _ := numerical.NewFromString(msg.Close)
		volume, _ := numerical.NewFromString(msg.Volume)

		kline := connector.Kline{
			Pair:      pair,
			Interval:  interval,
			OpenTime:  b.timeProvider.Now(),
			Open:      open.InexactFloat64(),
			High:      high.InexactFloat64(),
			Low:       low.InexactFloat64(),
			Close:     closePrice.InexactFloat64(),
			Volume:    volume.InexactFloat64(),
			CloseTime: b.timeProvider.Now(),
		}

		select {
		case ch <- kline:
		default:
			b.appLogger.Warn("Kline channel full, dropping message")
		}
	})

	if err != nil {
		close(ch)
		return err
	}

	// Store channel
	key := pair.Symbol() + ":" + interval
	b.klineMu.Lock()
	b.klineChannels[key] = ch
	b.klineMu.Unlock()

	return nil
}

func (b *bybit) UnsubscribeKlines(pair portfolio.Pair, interval string) error {
	key := pair.Symbol() + ":" + interval

	b.klineMu.Lock()
	if ch, exists := b.klineChannels[key]; exists {
		close(ch)
		delete(b.klineChannels, key)
	}
	b.klineMu.Unlock()

	return nil
}

func (b *bybit) GetErrorChannel() <-chan error {
	if b.wsService != nil {
		return b.wsService.GetErrorChannel()
	}
	return b.errorCh
}

// AssetBalanceUpdates returns the asset balance updates channel
func (b *bybit) AssetBalanceUpdates() <-chan connector.AssetBalance {
	return b.balanceCh
}
