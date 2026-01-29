package security

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/wisp-trading/sdk/pkg/types/logging"
)

// authManager handles authentication for WebSocket connections
type authManager struct {
	provider     AuthProvider
	refreshMutex sync.Mutex
	logger       logging.ApplicationLogger
}

func NewAuthManager(provider AuthProvider, logger logging.ApplicationLogger) AuthManager {
	return &authManager{
		provider: provider,
		logger:   logger,
	}
}

func (am *authManager) GetSecureHeaders(ctx context.Context) (http.Header, error) {
	// Ensure authentication is valid
	if !am.provider.IsAuthenticated() {
		if err := am.refreshAuth(ctx); err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Check if token is expiring soon (within 5 minutes)
	if time.Until(am.provider.GetTokenExpiry()) < 5*time.Minute {
		am.logger.Debug("Token expiring soon, refreshing authentication")
		if err := am.refreshAuth(ctx); err != nil {
			am.logger.Warn("Failed to refresh expiring token: %v", err)
		}
	}

	headers, err := am.provider.GetAuthHeaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth headers: %w", err)
	}

	// Add security headers
	if headers == nil {
		headers = make(http.Header)
	}

	headers.Set("User-Agent", "Trading-Bot-WebSocket/1.0")

	return headers, nil
}

func (am *authManager) refreshAuth(ctx context.Context) error {
	am.refreshMutex.Lock()
	defer am.refreshMutex.Unlock()

	// Double-check authentication status after acquiring lock
	if am.provider.IsAuthenticated() {
		return nil
	}

	am.logger.Debug("Refreshing authentication")
	if err := am.provider.Refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh authentication: %w", err)
	}

	am.logger.Debug("Authentication refreshed successfully")
	return nil
}

func (am *authManager) ValidateConnection(_ context.Context) error {
	if !am.provider.IsAuthenticated() {
		return fmt.Errorf("not authenticated")
	}

	// Check token expiry
	if time.Now().After(am.provider.GetTokenExpiry()) {
		return fmt.Errorf("authentication token expired")
	}

	return nil
}

// PeriodicRefresh starts a goroutine that periodically refreshes authentication
func (am *authManager) PeriodicRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := am.refreshAuth(ctx); err != nil {
				am.logger.Error("Periodic auth refresh failed: %v", err)
			}
		}
	}
}
