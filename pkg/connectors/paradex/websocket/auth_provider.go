package websockets

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/wisp-trading/connectors/pkg/connectors/paradex/adaptor"
)

type ParadexAuthProvider struct {
	client *adaptor.Client
}

func (pap *ParadexAuthProvider) GetAuthHeaders(ctx context.Context) (http.Header, error) {
	if err := pap.client.Authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", pap.client.GetJWTToken()))
	headers.Set("PARADEX-STARKNET-ACCOUNT", pap.client.GetDexAccountAddress())

	return headers, nil
}

func (pap *ParadexAuthProvider) IsAuthenticated() bool {
	return pap.client.GetJWTToken() != ""
}

func (pap *ParadexAuthProvider) Refresh(ctx context.Context) error {
	return pap.client.Authenticate(ctx)
}

func (pap *ParadexAuthProvider) GetTokenExpiry() time.Time {
	return pap.client.GetTokenExpiry()
}
