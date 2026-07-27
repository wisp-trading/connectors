package websocket

import "encoding/json"

// WSMessage is the Hyperliquid WebSocket wire envelope we parse ourselves.
//
// go-hyperliquid v0.35 unexported its internal wsMessage type and exported
// WsMsg with map[string]any data. Our stack still works best with raw JSON
// for channel-specific unmarshalling, so we keep a local envelope.
type WSMessage struct {
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
}
