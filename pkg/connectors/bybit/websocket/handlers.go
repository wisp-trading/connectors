package websocket

import (
	"encoding/json"
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// ============================================================================
// Message Handlers
// ============================================================================

func (ws *WebSocketService) handleOrderBookMessage(_ string, data interface{}, timestamp int64) error {
	// Parse orderbook data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal orderbook data: %w", err)
	}

	var obData OrderBookData
	if err := json.Unmarshal(dataBytes, &obData); err != nil {
		return fmt.Errorf("failed to parse orderbook data: %w", err)
	}

	// Convert to OrderBookMessage
	obMsg := &OrderBookMessage{
		Symbol:    obData.Symbol,
		Bids:      obData.Bids,
		Asks:      obData.Asks,
		Timestamp: timestamp,
		UpdateID:  obData.UpdateID,
	}

	// Call all registered callbacks for this symbol
	ws.orderBookMu.RLock()
	for _, callback := range ws.orderBookCallbacks {
		go callback(obMsg) // Call in goroutine to avoid blocking
	}
	ws.orderBookMu.RUnlock()

	return nil
}

func (ws *WebSocketService) handleTradesMessage(_ string, data interface{}, _ int64) error {
	// Parse trade data (Bybit sends array of trades)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal trade data: %w", err)
	}

	var tradeDataList []TradeData
	if err := json.Unmarshal(dataBytes, &tradeDataList); err != nil {
		return fmt.Errorf("failed to parse trade data: %w", err)
	}

	// Convert to TradeMessage array
	trades := make([]TradeMessage, 0, len(tradeDataList))
	for _, td := range tradeDataList {
		trades = append(trades, TradeMessage{
			ID:        td.TradeID,
			Symbol:    td.Symbol,
			Price:     td.Price,
			Quantity:  td.Size,
			Side:      parseSide(td.Side),
			Timestamp: td.Timestamp,
		})
	}

	// Call all registered callbacks
	ws.tradesMu.RLock()
	for _, callback := range ws.tradesCallbacks {
		go callback(trades) // Call in goroutine to avoid blocking
	}
	ws.tradesMu.RUnlock()

	return nil
}

func (ws *WebSocketService) handleKlineMessage(topic string, data interface{}, timestamp int64) error {
	// Parse kline data (Bybit sends array with single kline)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal kline data: %w", err)
	}

	var klineDataList []KlineData
	if err := json.Unmarshal(dataBytes, &klineDataList); err != nil {
		return fmt.Errorf("failed to parse kline data: %w", err)
	}

	if len(klineDataList) == 0 {
		return nil
	}

	kd := klineDataList[0]

	// Extract symbol and interval from topic: "kline.1.BTCUSDT"
	// Topic format: kline.{interval}.{symbol}
	symbol := ""
	interval := ""
	if len(topic) > 6 {
		parts := topic[6:] // Remove "kline."
		// Find first dot
		for i, ch := range parts {
			if ch == '.' {
				interval = parts[:i]
				if i+1 < len(parts) {
					symbol = parts[i+1:]
				}
				break
			}
		}
	}

	// Convert to KlineMessage
	klineMsg := &KlineMessage{
		Symbol:    symbol,
		Interval:  interval,
		StartTime: kd.Start,
		Open:      kd.Open,
		High:      kd.High,
		Low:       kd.Low,
		Close:     kd.Close,
		Volume:    kd.Volume,
		Timestamp: timestamp,
	}

	// Call all registered callbacks
	ws.klinesMu.RLock()
	for _, callback := range ws.klinesCallbacks {
		go callback(klineMsg) // Call in goroutine to avoid blocking
	}
	ws.klinesMu.RUnlock()

	return nil
}

func (ws *WebSocketService) handlePositionMessage(data interface{}, timestamp int64) error {
	// Parse position data (Bybit sends array of positions)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal position data: %w", err)
	}

	var posDataList []PositionData
	if err := json.Unmarshal(dataBytes, &posDataList); err != nil {
		return fmt.Errorf("failed to parse position data: %w", err)
	}

	// Convert and call callbacks for each position
	for _, pd := range posDataList {
		posMsg := &PositionMessage{
			Symbol:           pd.Symbol,
			Side:             pd.Side,
			Size:             pd.Size,
			EntryPrice:       pd.EntryPrice,
			MarkPrice:        pd.MarkPrice,
			LiquidationPrice: pd.LiqPrice,
			UnrealizedPnL:    pd.UnrealisedPnl,
			RealizedPnL:      pd.CumRealisedPnl,
			Leverage:         pd.Leverage,
			Timestamp:        timestamp,
		}

		// Call all registered callbacks
		ws.positionMu.RLock()
		for _, callback := range ws.positionCallbacks {
			go callback(posMsg) // Call in goroutine to avoid blocking
		}
		ws.positionMu.RUnlock()
	}

	return nil
}

func (ws *WebSocketService) handleBalanceMessage(data interface{}, timestamp int64) error {
	// Parse wallet data (Bybit sends array with single wallet)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet data: %w", err)
	}

	var walletDataList []WalletData
	if err := json.Unmarshal(dataBytes, &walletDataList); err != nil {
		return fmt.Errorf("failed to parse wallet data: %w", err)
	}

	if len(walletDataList) == 0 {
		return nil
	}

	wd := walletDataList[0]

	// Convert to AccountBalanceMessage
	balanceMsg := &AccountBalanceMessage{
		TotalEquity:           wd.TotalEquity,
		TotalAvailableBalance: wd.TotalAvailableBalance,
		TotalMarginBalance:    wd.TotalMarginBalance,
		TotalPerpUPL:          wd.TotalPerpUPL,
		Timestamp:             timestamp,
	}

	// Call all registered callbacks
	ws.balanceMu.RLock()
	for _, callback := range ws.balanceCallbacks {
		go callback(balanceMsg) // Call in goroutine to avoid blocking
	}
	ws.balanceMu.RUnlock()

	return nil
}

// ============================================================================
// Helper functions
// ============================================================================

func parseSide(side string) connector.OrderSide {
	// Bybit uses "Buy" and "Sell"
	// Convert to our connector types
	if side == "Buy" {
		return connector.OrderSideBuy
	}
	if side == "Sell" {
		return connector.OrderSideSell
	}
	return connector.OrderSide(side)
}
