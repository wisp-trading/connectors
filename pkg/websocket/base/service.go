package base

import (
	"context"
	"sync"
	"time"

	"github.com/wisp-trading/connectors/pkg/websocket/connection"
	"github.com/wisp-trading/connectors/pkg/websocket/performance"
	"github.com/wisp-trading/connectors/pkg/websocket/security"
	"github.com/wisp-trading/sdk/pkg/types/logging"
)

type Config struct {
	URL            string
	ReconnectDelay time.Duration
	MaxReconnects  int
	PingInterval   time.Duration
	PongTimeout    time.Duration
	MaxMessageSize int
}

type baseService struct {
	// Configuration
	config Config
	logger logging.ApplicationLogger

	validator      security.MessageValidator
	rateLimiter    security.RateLimiter
	metrics        performance.Metrics
	circuitBreaker performance.CircuitBreaker

	// Connection management
	ConnectionManager connection.ConnectionManager

	// Connection state
	isConnected bool
	connMutex   sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewBaseService(
	config Config,
	logger logging.ApplicationLogger,
	validator security.MessageValidator,
	rateLimiter security.RateLimiter,
	metrics performance.Metrics,
	circuitBreaker performance.CircuitBreaker,
) BaseService {
	return &baseService{
		config:         config,
		logger:         logger,
		validator:      validator,
		rateLimiter:    rateLimiter,
		metrics:        metrics,
		circuitBreaker: circuitBreaker,
	}
}

func (bs *baseService) ProcessMessage(message []byte, handler func([]byte) error) error {
	start := time.Now()
	bs.metrics.IncrementReceived()

	defer func() {
		latency := time.Since(start)
		bs.metrics.IncrementProcessed(latency)

		if latency > 10*time.Millisecond {
			bs.logger.Warn("Slow message processing: %v", latency)
		}
	}()

	// Rate limiting
	if !bs.rateLimiter.Allow() {
		bs.metrics.IncrementDropped()
		bs.logger.Warn("Message rate limit exceeded, dropping message")
		return nil
	}

	// Validation
	if err := bs.validator.ValidateMessage(message); err != nil {
		bs.metrics.IncrementDropped()
		bs.logger.Warn("Message validation failed: %v", err)
		return nil
	}

	// Process message with circuit breaker
	return bs.circuitBreaker.Execute(func() error {
		return handler(message)
	})
}

func (bs *baseService) GetMetrics() map[string]interface{} {
	return bs.metrics.GetStats()
}

func (bs *baseService) IsConnected() bool {
	bs.connMutex.RLock()
	defer bs.connMutex.RUnlock()
	return bs.isConnected
}

func (bs *baseService) SetConnected(connected bool) {
	bs.connMutex.Lock()
	defer bs.connMutex.Unlock()
	bs.isConnected = connected
}
