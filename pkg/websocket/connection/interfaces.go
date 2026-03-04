package connection

import (
	"context"
	"time"
)

type ReconnectStrategy interface {
	NextDelay(attempt int) time.Duration
	ShouldReconnect(attempt int, err error) bool
	Reset()
}

// ConnectionManager Interface defines WebSocket connection operations
type ConnectionManager interface {
	Connect(ctx context.Context, config *Config, websocketUrl *string) error
	Disconnect() error
	Send(data []byte) error
	SendMessage(data []byte) error
	SendJSON(v interface{}) error
	SendPing() error
	SetCallbacks(onConnect func() error, onDisconnect func() error, onMessage func([]byte) error, onError func(error))
	GetState() ConnectionState
	GetConnectionStats() map[string]interface{}
	IsHealthy() bool
}

// reconnectManager Interface defines reconnection strategy operations
type ReconnectManager interface {
	StartReconnection(ctx context.Context) error
	StopReconnection()
	SetCallbacks(onStart func(int), onFail func(int, error), onSuccess func(int))
}

// ReconnectionStrategy Interface defines strategies for reconnection backoff
type ReconnectionStrategy interface {
	NextDelay(attempt int) time.Duration
	MaxAttempts() int
}
