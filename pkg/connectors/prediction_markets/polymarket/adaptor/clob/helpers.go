package clob

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/wisp-trading/sdk/pkg/types/connector"
	"github.com/wisp-trading/sdk/pkg/types/connector/prediction"
)

func calculateAmounts(order prediction.LimitOrder) (maker, taker string) {
	price := order.Price.InexactFloat64()

	if order.Side == connector.OrderSideBuy {
		// BUY: maker gives USDC, taker gives tokens
		if order.SpendAmount != nil {
			// User said "I want to spend $X"
			usdcAmount := order.SpendAmount.InexactFloat64()
			tokensAmount := (usdcAmount * price) / (1 - price)

			maker = fmt.Sprintf("%.0f", usdcAmount*1_000_000)
			taker = fmt.Sprintf("%.0f", tokensAmount*1_000_000)
		} else {
			// User said "I want to receive Y tokens"
			tokensAmount := order.ReceiveAmount.InexactFloat64()
			usdcAmount := (tokensAmount * (1 - price)) / price

			maker = fmt.Sprintf("%.0f", usdcAmount*1_000_000)
			taker = fmt.Sprintf("%.0f", tokensAmount*1_000_000)
		}
	} else {
		// SELL: maker gives tokens, taker gives USDC
		if order.SpendAmount != nil {
			// User said "I want to sell X tokens"
			tokensAmount := order.SpendAmount.InexactFloat64()
			usdcAmount := tokensAmount * price

			maker = fmt.Sprintf("%.0f", tokensAmount*1_000_000)
			taker = fmt.Sprintf("%.0f", usdcAmount*1_000_000)
		} else {
			// User said "I want to receive $Y"
			usdcAmount := order.ReceiveAmount.InexactFloat64()
			tokensAmount := usdcAmount / price

			maker = fmt.Sprintf("%.0f", tokensAmount*1_000_000)
			taker = fmt.Sprintf("%.0f", usdcAmount*1_000_000)
		}
	}

	return maker, taker
}

func (c *polymarketClient) deriveCredentials() (*APICredentials, error) {
	timestamp := time.Now().Unix()
	nonce := int64(0) // Default nonce

	// Build the EIP-712 typed data
	domain := apitypes.TypedDataDomain{
		Name:    "ClobAuthDomain",
		Version: "1",
		ChainId: math.NewHexOrDecimal256(137), // Polygon mainnet
	}

	types := apitypes.Types{
		"EIP712Domain": []apitypes.Type{
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
		},
		"ClobAuth": []apitypes.Type{
			{Name: "address", Type: "address"},
			{Name: "timestamp", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "message", Type: "string"},
		},
	}

	message := apitypes.TypedDataMessage{
		"address":   strings.ToLower(c.signerAddress),
		"timestamp": fmt.Sprintf("%d", timestamp),
		"nonce":     fmt.Sprintf("%d", nonce),
		"message":   "This message attests that I control the given wallet",
	}

	typedData := apitypes.TypedData{
		Types:       types,
		PrimaryType: "ClobAuth",
		Domain:      domain,
		Message:     message,
	}

	// Hash and sign
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("failed to hash domain: %w", err)
	}

	typedDataHash, err := typedData.HashStruct("ClobAuth", typedData.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash message: %w", err)
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash := crypto.Keccak256Hash(rawData)

	signature, err := crypto.Sign(hash.Bytes(), c.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	if signature[64] < 27 {
		signature[64] += 27
	}

	// Make request with L1 headers
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/auth/derive-api-key", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("POLY_ADDRESS", c.signerAddress)
	req.Header.Set("POLY_SIGNATURE", "0x"+hex.EncodeToString(signature))
	req.Header.Set("POLY_TIMESTAMP", fmt.Sprintf("%d", timestamp))
	req.Header.Set("POLY_NONCE", fmt.Sprintf("%d", nonce))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var creds APICredentials
	if err := json.Unmarshal(respBody, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &creds, nil
}
