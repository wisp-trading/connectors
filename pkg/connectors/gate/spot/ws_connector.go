package spot

import (
	"fmt"

	"github.com/wisp-trading/connectors/pkg/connectors/gate/websocket"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/portfolio"
	"github.com/wisp-trading/sdk/pkg/types/wisp/numerical"
)

// ConnectWebSocket establishes WebSocket connection
func (g *gateSpot) ConnectWebSocket() error {
	if !g.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if g.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	// Pass the WebSocket URL from config
	return g.wsService.Connect(g.config.WebSocketURL)
}

// DisconnectWebSocket closes WebSocket connection
func (g *gateSpot) DisconnectWebSocket() error {
	if g.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	return g.wsService.Disconnect()
}

// SubscribeOrderBook subscribes to order book updates
func (g *gateSpot) SubscribeOrderBook(pair portfolio.Pair) error {
	if !g.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if g.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	// Create channel for this subscription
	ch := make(chan connector.OrderBook, 100)

	gateSymbol := g.GetSpotSymbol(pair)

	// Subscribe to WebSocket
	_, err := g.wsService.SubscribeToOrderBook(gateSymbol, func(msg *websocket.OrderBookMessage) {
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
			Bids:      bids,
			Asks:      asks,
			Timestamp: g.timeProvider.Now(),
		}

		select {
		case ch <- orderBook:
		default:
			g.appLogger.Warn("Order book channel full, dropping message")
		}
	})

	if err != nil {
		close(ch)
		return err
	}

	// Store channel
	g.orderBookMu.Lock()
	g.orderBookChannels[pair.Symbol()] = ch
	g.orderBookMu.Unlock()

	return nil
}

// UnsubscribeOrderBook unsubscribes from order book updates
func (g *gateSpot) UnsubscribeOrderBook(pair portfolio.Pair) error {
	// Remove from WebSocket subscriptions
	// Note: We need to track subscription IDs to properly unsubscribe
	// For now, just close the channel
	g.orderBookMu.Lock()
	if ch, exists := g.orderBookChannels[pair.Symbol()]; exists {
		close(ch)
		delete(g.orderBookChannels, pair.Symbol())
	}
	g.orderBookMu.Unlock()

	return nil
}

// SubscribeKlines subscribes to kline/candlestick updates
func (g *gateSpot) SubscribeKlines(pair portfolio.Pair, interval string) error {
	if !g.initialized {
		return fmt.Errorf("connector not initialized")
	}

	if g.wsService == nil {
		return fmt.Errorf("websocket service not initialized")
	}

	ch := make(chan connector.Kline, 100)
	gateSymbol := g.GetSpotSymbol(pair)

	_, err := g.wsService.SubscribeToKlines(gateSymbol, interval, func(msg *websocket.KlineMessage) {
		// Convert to connector.Kline
		open, _ := numerical.NewFromString(msg.Open)
		high, _ := numerical.NewFromString(msg.High)
		low, _ := numerical.NewFromString(msg.Low)
		closePrice, _ := numerical.NewFromString(msg.Close)
		volume, _ := numerical.NewFromString(msg.Volume)

		kline := connector.Kline{
			Pair:      pair,
			Interval:  interval,
			OpenTime:  g.timeProvider.Now(),
			Open:      open.InexactFloat64(),
			High:      high.InexactFloat64(),
			Low:       low.InexactFloat64(),
			Close:     closePrice.InexactFloat64(),
			Volume:    volume.InexactFloat64(),
			CloseTime: g.timeProvider.Now(),
		}

		select {
		case ch <- kline:
		default:
			g.appLogger.Warn("Kline channel full, dropping message")
		}
	})

	if err != nil {
		close(ch)
		return err
	}

	// Store channel
	key := pair.Symbol() + ":" + interval
	g.klineMu.Lock()
	g.klineChannels[key] = ch
	g.klineMu.Unlock()

	return nil
}

// UnsubscribeKlines unsubscribes from kline updates
func (g *gateSpot) UnsubscribeKlines(pair portfolio.Pair, interval string) error {
	key := pair.Symbol() + ":" + interval

	g.klineMu.Lock()
	if ch, exists := g.klineChannels[key]; exists {
		close(ch)
		delete(g.klineChannels, key)
	}
	g.klineMu.Unlock()

	return nil
}

// TradeUpdates subscribes to trade updates
func (g *gateSpot) TradeUpdates() <-chan connector.Trade {
	// Return the existing trade channel
	return g.tradeCh
}

// GetOrderBookChannels returns all order book channels
func (g *gateSpot) GetOrderBookChannels() map[string]<-chan connector.OrderBook {
	g.orderBookMu.RLock()
	defer g.orderBookMu.RUnlock()

	result := make(map[string]<-chan connector.OrderBook)
	for k, v := range g.orderBookChannels {
		result[k] = v
	}
	return result
}

// GetKlineChannels returns all kline channels
func (g *gateSpot) GetKlineChannels() map[string]<-chan connector.Kline {
	g.klineMu.RLock()
	defer g.klineMu.RUnlock()

	result := make(map[string]<-chan connector.Kline)
	for k, v := range g.klineChannels {
		result[k] = v
	}
	return result
}

// AssetBalanceUpdates subscribes to balance updates
func (g *gateSpot) AssetBalanceUpdates() <-chan connector.AssetBalance {
	if !g.initialized {
		g.appLogger.Error("Connector not initialized for AssetBalanceUpdates")
		return g.balanceCh
	}

	if g.wsService == nil {
		g.appLogger.Error("WebSocket service not initialized")
		return g.balanceCh
	}

	// Subscribe to account balance updates via WebSocket
	_, err := g.wsService.SubscribeToAccountBalance(func(msg *websocket.AccountBalanceMessage) {
		for _, bal := range msg.Balances {
			available, _ := numerical.NewFromString(bal.Available)
			locked, _ := numerical.NewFromString(bal.Locked)
			total := available.Add(locked)

			// Skip assets with zero balance
			if total.IsZero() {
				continue
			}

			balance := connector.AssetBalance{
				Asset:     portfolio.NewAsset(bal.Currency),
				Free:      available,
				Locked:    locked,
				Total:     total,
				UpdatedAt: g.timeProvider.Now(),
			}

			select {
			case g.balanceCh <- balance:
			default:
				g.appLogger.Warn("Balance channel full, dropping message", "asset", bal.Currency)
			}
		}
	})

	if err != nil {
		g.appLogger.Error("Failed to subscribe to account balance", "error", err)
	}

	return g.balanceCh
}

// ErrorChannel returns the error channel for WebSocket errors
func (g *gateSpot) ErrorChannel() <-chan error {
	if g.wsService != nil {
		return g.wsService.GetErrorChannel()
	}
	return g.errorCh
}
