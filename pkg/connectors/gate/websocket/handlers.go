package websocket

import (
	"fmt"

	"github.com/wisp-trading/sdk/pkg/types/connector"
)

// handleOrderBookMessage processes order book updates
func (ws *WebSocketService) handleOrderBookMessage(gateMsg map[string]interface{}) error {
	result, ok := gateMsg["result"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid order book message format")
	}

	symbol, _ := result["s"].(string)
	timestamp, _ := result["t"].(float64)

	// Parse bids
	bidsRaw, _ := result["bids"].([]interface{})
	bids := make([][]string, 0, len(bidsRaw))
	for _, bid := range bidsRaw {
		bidArr, ok := bid.([]interface{})
		if ok && len(bidArr) >= 2 {
			price, _ := bidArr[0].(string)
			qty, _ := bidArr[1].(string)
			bids = append(bids, []string{price, qty})
		}
	}

	// Parse asks
	asksRaw, _ := result["asks"].([]interface{})
	asks := make([][]string, 0, len(asksRaw))
	for _, ask := range asksRaw {
		askArr, ok := ask.([]interface{})
		if ok && len(askArr) >= 2 {
			price, _ := askArr[0].(string)
			qty, _ := askArr[1].(string)
			asks = append(asks, []string{price, qty})
		}
	}

	msg := &OrderBookMessage{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: int64(timestamp),
	}

	// Call all registered handlers
	ws.orderBookMu.RLock()
	defer ws.orderBookMu.RUnlock()

	for _, handler := range ws.orderBookCallbacks {
		handler(msg)
	}

	return nil
}

// handleTradesMessage processes trade updates
func (ws *WebSocketService) handleTradesMessage(gateMsg map[string]interface{}) error {
	result, ok := gateMsg["result"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid trades message format")
	}

	trades := make([]TradeMessage, 0, len(result))
	for _, tradeRaw := range result {
		trade, ok := tradeRaw.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := trade["id"].(float64)
		symbol, _ := trade["currency_pair"].(string)
		price, _ := trade["price"].(string)
		amount, _ := trade["amount"].(string)
		sideStr, _ := trade["side"].(string)
		createTime, _ := trade["create_time"].(float64)
		createTimeMs, _ := trade["create_time_ms"].(float64)

		var side connector.OrderSide
		if sideStr == "buy" {
			side = connector.OrderSideBuy
		} else {
			side = connector.OrderSideSell
		}

		trades = append(trades, TradeMessage{
			ID:           int64(id),
			Symbol:       symbol,
			Price:        price,
			Amount:       amount,
			Side:         side,
			Timestamp:    int64(createTime),
			CreateTimeMs: int64(createTimeMs),
		})
	}

	// Call all registered handlers
	ws.tradesMu.RLock()
	defer ws.tradesMu.RUnlock()

	for _, handler := range ws.tradesCallbacks {
		handler(trades)
	}

	return nil
}

// handleKlineMessage processes kline/candlestick updates
func (ws *WebSocketService) handleKlineMessage(gateMsg map[string]interface{}) error {
	result, ok := gateMsg["result"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid kline message format")
	}

	symbol, _ := result["n"].(string)
	interval, _ := result["i"].(string)
	openTime, _ := result["t"].(float64)
	closeTime, _ := result["c"].(float64)
	open, _ := result["o"].(string)
	high, _ := result["h"].(string)
	low, _ := result["l"].(string)
	closePrice, _ := result["a"].(string)
	volume, _ := result["v"].(string)
	quoteVolume, _ := result["q"].(string)

	msg := &KlineMessage{
		Symbol:      symbol,
		Interval:    interval,
		OpenTime:    int64(openTime),
		CloseTime:   int64(closeTime),
		Open:        open,
		High:        high,
		Low:         low,
		Close:       closePrice,
		ClosePrice:  closePrice,
		Volume:      volume,
		QuoteVolume: quoteVolume,
	}

	// Call all registered handlers
	ws.klinesMu.RLock()
	defer ws.klinesMu.RUnlock()

	for _, handler := range ws.klinesCallbacks {
		handler(msg)
	}

	return nil
}

// handleBalanceMessage processes account balance updates
func (ws *WebSocketService) handleBalanceMessage(gateMsg map[string]interface{}) error {
	result, ok := gateMsg["result"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid balance message format")
	}

	balances := make(map[string]Balance)
	timestamp := int64(0)

	for _, balRaw := range result {
		bal, ok := balRaw.(map[string]interface{})
		if !ok {
			continue
		}

		currency, _ := bal["currency"].(string)
		available, _ := bal["available"].(string)
		locked, _ := bal["locked"].(string)

		balances[currency] = Balance{
			Currency:  currency,
			Available: available,
			Locked:    locked,
		}

		if ts, ok := bal["timestamp"].(float64); ok {
			timestamp = int64(ts)
		}
	}

	msg := &AccountBalanceMessage{
		Timestamp: timestamp,
		Balances:  balances,
	}

	// Call all registered handlers
	ws.balanceMu.RLock()
	defer ws.balanceMu.RUnlock()

	for _, handler := range ws.balanceCallbacks {
		handler(msg)
	}

	return nil
}

// handleOrderMessage processes order updates
func (ws *WebSocketService) handleOrderMessage(gateMsg map[string]interface{}) error {
	result, ok := gateMsg["result"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid order message format")
	}

	for _, orderRaw := range result {
		order, ok := orderRaw.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := order["id"].(string)
		symbol, _ := order["currency_pair"].(string)
		side, _ := order["side"].(string)
		orderType, _ := order["type"].(string)
		status, _ := order["status"].(string)
		price, _ := order["price"].(string)
		amount, _ := order["amount"].(string)
		filledAmount, _ := order["filled_amount"].(string)
		createTime, _ := order["create_time_ms"].(float64)
		updateTime, _ := order["update_time_ms"].(float64)

		msg := &OrderMessage{
			ID:           id,
			Symbol:       symbol,
			Side:         side,
			Type:         orderType,
			Status:       status,
			Price:        price,
			Amount:       amount,
			FilledAmount: filledAmount,
			CreateTime:   int64(createTime),
			UpdateTime:   int64(updateTime),
		}

		// Call all registered handlers
		ws.orderMu.RLock()
		for _, handler := range ws.orderCallbacks {
			handler(msg)
		}
		ws.orderMu.RUnlock()
	}

	return nil
}
