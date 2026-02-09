package adaptor

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// Order represents a Polymarket CLOB order for signing
type Order struct {
	Salt          int64  `json:"salt"`
	Maker         string `json:"maker"`
	Signer        string `json:"signer"`
	Taker         string `json:"taker"`
	TokenID       string `json:"tokenId"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Side          string `json:"side"`
	FeeRateBps    string `json:"feeRateBps"`
	Nonce         string `json:"nonce"`
	SignatureType int    `json:"signatureType"`
	Expiration    int64  `json:"expiration"`
}

// OrderSigner handles EIP-712 signing for Polymarket orders
type OrderSigner struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
	chainID    int
	domain     apitypes.TypedDataDomain
}

// NewOrderSigner creates a new order signer with the given private key and chain ID
func NewOrderSigner(privateKeyHex string, chainID int) (*OrderSigner, error) {
	if chainID <= 0 {
		return nil, fmt.Errorf("invalid chain ID: must be positive")
	}

	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	if len(privateKeyHex) != 64 {
		return nil, fmt.Errorf("invalid private key length: expected 64 hex characters, got %d", len(privateKeyHex))
	}

	// Decode hex string to bytes
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	// Convert to ECDSA private key
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Derive the Ethereum address from the private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to get public key")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Create EIP-712 domain for Polymarket CLOB
	domain := apitypes.TypedDataDomain{
		Name:              "ClobAuthDomain",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(int64(chainID)),
		VerifyingContract: "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC", // Polymarket CLOB contract
	}

	return &OrderSigner{
		privateKey: privateKey,
		address:    address,
		chainID:    chainID,
		domain:     domain,
	}, nil
}

// SignOrder signs a Polymarket order using EIP-712
func (s *OrderSigner) SignOrder(order Order) (string, error) {
	// Validate required fields
	if order.Maker == "" {
		return "", fmt.Errorf("order maker is required")
	}
	if order.TokenID == "" {
		return "", fmt.Errorf("order tokenId is required")
	}
	if order.MakerAmount == "" {
		return "", fmt.Errorf("order makerAmount is required")
	}
	if order.TakerAmount == "" {
		return "", fmt.Errorf("order takerAmount is required")
	}

	// Define the Order type for EIP-712
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Order": []apitypes.Type{
				{Name: "salt", Type: "uint256"},
				{Name: "maker", Type: "address"},
				{Name: "signer", Type: "address"},
				{Name: "taker", Type: "address"},
				{Name: "tokenId", Type: "uint256"},
				{Name: "makerAmount", Type: "uint256"},
				{Name: "takerAmount", Type: "uint256"},
				{Name: "side", Type: "string"},
				{Name: "feeRateBps", Type: "uint256"},
				{Name: "nonce", Type: "uint256"},
				{Name: "signatureType", Type: "uint8"},
				{Name: "expiration", Type: "uint256"},
			},
		},
		PrimaryType: "Order",
		Domain:      s.domain,
		Message: apitypes.TypedDataMessage{
			"salt":          fmt.Sprintf("%d", order.Salt),
			"maker":         order.Maker,
			"signer":        order.Signer,
			"taker":         order.Taker,
			"tokenId":       order.TokenID,
			"makerAmount":   order.MakerAmount,
			"takerAmount":   order.TakerAmount,
			"side":          order.Side,
			"feeRateBps":    order.FeeRateBps,
			"nonce":         order.Nonce,
			"signatureType": fmt.Sprintf("%d", order.SignatureType),
			"expiration":    fmt.Sprintf("%d", order.Expiration),
		},
	}

	// Hash the typed data
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return "", fmt.Errorf("failed to hash domain: %w", err)
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return "", fmt.Errorf("failed to hash message: %w", err)
	}

	// Construct the final hash: keccak256("\x19\x01" + domainSeparator + typedDataHash)
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash := crypto.Keccak256Hash(rawData)

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign order: %w", err)
	}

	// Adjust V value (go-ethereum returns 0/1, Ethereum expects 27/28)
	if signature[64] < 27 {
		signature[64] += 27
	}

	// Return hex-encoded signature with 0x prefix
	return "0x" + hex.EncodeToString(signature), nil
}

// GetAddress returns the Ethereum address associated with this signer
func (s *OrderSigner) GetAddress() string {
	return s.address.Hex()
}

// GenerateSalt generates a random salt value for orders
func GenerateSalt() int64 {
	// Use current timestamp + secure random value for uniqueness
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	randomValue := new(big.Int).SetBytes(randomBytes).Int64()
	if randomValue < 0 {
		randomValue = -randomValue
	}
	return time.Now().UnixNano() + (randomValue % 1000000)
}

// SignMessage signs an arbitrary message for authentication
func (s *OrderSigner) SignMessage(message string) (string, error) {
	// Hash the message
	hash := crypto.Keccak256Hash([]byte(message))

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	// Adjust V value
	if signature[64] < 27 {
		signature[64] += 27
	}

	return "0x" + hex.EncodeToString(signature), nil
}
