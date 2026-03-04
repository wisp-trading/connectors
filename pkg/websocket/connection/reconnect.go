package connection

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/logging"
)

type exponentialBackoffStrategy struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	maxAttempts  int
	Multiplier   float64
	Jitter       bool
	randSource   *rand.Rand
	mutex        sync.Mutex
}

func NewExponentialBackoffStrategy(initialDelay, maxDelay time.Duration, maxAttempts int) ReconnectionStrategy {
	return &exponentialBackoffStrategy{
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		maxAttempts:  maxAttempts,
		Multiplier:   2.0,
		Jitter:       true,
		randSource:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (ebs *exponentialBackoffStrategy) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return ebs.InitialDelay
	}

	delay := float64(ebs.InitialDelay) * math.Pow(ebs.Multiplier, float64(attempt-1))

	if delay > float64(ebs.MaxDelay) {
		delay = float64(ebs.MaxDelay)
	}

	if ebs.Jitter {
		ebs.mutex.Lock()
		jitterFactor := 2*ebs.randSource.Float64() - 1
		ebs.mutex.Unlock()

		jitter := delay * 0.1 * jitterFactor
		delay += jitter

		if delay < 0 {
			delay = float64(ebs.InitialDelay)
		}
	}

	return time.Duration(delay)
}

func (ebs *exponentialBackoffStrategy) ShouldReconnect(attempt int, _ error) bool {
	return attempt < ebs.maxAttempts
}

func (ebs *exponentialBackoffStrategy) MaxAttempts() int {
	return ebs.maxAttempts
}

func (ebs *exponentialBackoffStrategy) Reset() {
}

type reconnectManager struct {
	connectionManager ConnectionManager
	strategy          ReconnectionStrategy
	logger            logging.ApplicationLogger

	isReconnecting bool
	reconnectMutex sync.Mutex
	currentAttempt int

	onReconnectStart   func(attempt int)
	onReconnectFail    func(attempt int, err error)
	onReconnectSuccess func(attempt int)
}

func NewReconnectManager(
	connectionManager ConnectionManager,
	strategy ReconnectionStrategy,
	logger logging.ApplicationLogger,
) ReconnectManager {
	return &reconnectManager{
		connectionManager: connectionManager,
		strategy:          strategy,
		logger:            logger,
	}
}

func (rm *reconnectManager) SetCallbacks(
	onStart func(int),
	onFail func(int, error),
	onSuccess func(int),
) {
	rm.onReconnectStart = onStart
	rm.onReconnectFail = onFail
	rm.onReconnectSuccess = onSuccess
}

func (rm *reconnectManager) StopReconnection() {
	rm.reconnectMutex.Lock()
	defer rm.reconnectMutex.Unlock()
	rm.isReconnecting = false
}

func (rm *reconnectManager) StartReconnection(ctx context.Context) error {
	rm.reconnectMutex.Lock()
	defer rm.reconnectMutex.Unlock()

	if rm.isReconnecting {
		rm.logger.Debug("Reconnection already in progress")
		return nil
	}

	rm.isReconnecting = true
	rm.currentAttempt = 0

	go rm.reconnectLoop(ctx)
	return nil
}

func (rm *reconnectManager) reconnectLoop(ctx context.Context) {
	defer func() {
		rm.reconnectMutex.Lock()
		rm.isReconnecting = false
		rm.reconnectMutex.Unlock()
	}()

	rm.logger.Info("🔄 Reconnection loop started - will continuously monitor connection and attempt reconnection on disconnection")

	// Track the previous connection state to detect transitions
	previousState := rm.connectionManager.GetState()

	// Main monitoring loop - runs forever (until ctx.Done())
	monitorTicker := time.NewTicker(500 * time.Millisecond)
	defer monitorTicker.Stop()

	reconnectInProgress := false

	for {
		select {
		case <-ctx.Done():
			rm.logger.Error("❌ Reconnection loop CANCELLED BY CONTEXT")
			return
		case <-monitorTicker.C:
			currentState := rm.connectionManager.GetState()

			// Only trigger reconnection on state transition from connected → disconnected
			// AND only if we're not already attempting reconnection
			if !reconnectInProgress && previousState == StateConnected && currentState != StateConnected {
				rm.logger.Error("🔴 Connection lost! Detected state transition from %s to %s - starting reconnection attempts", previousState.String(), currentState.String())
				reconnectInProgress = true

				// Start attempting reconnection
				if err := rm.attemptReconnection(ctx); err != nil {
					rm.logger.Error("❌ Reconnection sequence failed: %v", err)
				}
				reconnectInProgress = false

				// Wait a bit before checking state again to let connection stabilize
				time.Sleep(100 * time.Millisecond)
				previousState = rm.connectionManager.GetState()
			} else if !reconnectInProgress && currentState != previousState {
				// Update previousState only when not reconnecting
				previousState = currentState
				rm.logger.Debug("📊 Connection state changed to: %s", currentState.String())
			}
		}
	}
}

// attemptReconnection handles the actual reconnection logic with exponential backoff
func (rm *reconnectManager) attemptReconnection(ctx context.Context) error {
	rm.currentAttempt = 0

	for {
		select {
		case <-ctx.Done():
			rm.logger.Error("❌ Reconnection attempt CANCELLED BY CONTEXT after %d attempts", rm.currentAttempt)
			return fmt.Errorf("context cancelled")
		default:
			// Check if user commanded stop before attempting reconnect
			if rm.connectionManager.GetState() == StateStopped {
				rm.logger.Info("🛑 StateStopped detected during reconnection - user commanded disconnect")
				return fmt.Errorf("user commanded disconnect")
			}

			rm.currentAttempt++

			if rm.currentAttempt > rm.strategy.MaxAttempts() {
				rm.logger.Error("🛑 Max reconnection attempts reached: %d attempts failed", rm.currentAttempt-1)
				if rm.onReconnectFail != nil {
					rm.onReconnectFail(rm.currentAttempt-1, fmt.Errorf("max attempts reached"))
				}
				return fmt.Errorf("max reconnection attempts exceeded")
			}

			delay := rm.strategy.NextDelay(rm.currentAttempt)
			rm.logger.Info("🔄 Reconnection attempt #%d after %.1f second delay", rm.currentAttempt, delay.Seconds())

			if rm.onReconnectStart != nil {
				rm.onReconnectStart(rm.currentAttempt)
			}

			select {
			case <-ctx.Done():
				rm.logger.Error("❌ Reconnection CANCELLED during delay at attempt %d", rm.currentAttempt)
				return fmt.Errorf("context cancelled during delay")
			case <-time.After(delay):
			}

			err := rm.connectionManager.Connect(ctx, nil, nil)
			if err == nil {
				rm.logger.Info("✅ Reconnection successful after %d attempts", rm.currentAttempt)
				if rm.onReconnectSuccess != nil {
					rm.onReconnectSuccess(rm.currentAttempt)
				}
				return nil // Successfully reconnected, exit attemptReconnection
			}

			rm.logger.Error("❌ Reconnection attempt #%d failed: %v", rm.currentAttempt, err)
			if rm.onReconnectFail != nil {
				rm.onReconnectFail(rm.currentAttempt, err)
			}
		}
	}
}

func (rm *reconnectManager) IsReconnecting() bool {
	rm.reconnectMutex.Lock()
	defer rm.reconnectMutex.Unlock()
	return rm.isReconnecting
}

func (rm *reconnectManager) GetCurrentAttempt() int {
	rm.reconnectMutex.Lock()
	defer rm.reconnectMutex.Unlock()
	return rm.currentAttempt
}

func (rm *reconnectManager) Stop() {
	rm.reconnectMutex.Lock()
	defer rm.reconnectMutex.Unlock()
	rm.isReconnecting = false
}
