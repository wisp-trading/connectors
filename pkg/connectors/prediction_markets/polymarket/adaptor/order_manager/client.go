package order_manager

import (
	"context"
	"math/big"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	"github.com/ethereum/go-ethereum/common"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"
)

// OrderManager wraps Polymarket CLOB API
type OrderManager interface {
	PlaceOrder(ctx context.Context, order prediction.LimitOrder) (clobtypes.OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (clobtypes.CancelResponse, error)
	GetOrderBook(ctx context.Context, outcome prediction.Outcome) (clobtypes.OrderBookResponse, error)
	GetOrderBooks(ctx context.Context, outcomes []prediction.Outcome) ([]clobtypes.OrderBook, error)
	GetBalance(ctx context.Context) (clobtypes.BalanceAllowanceResponse, error)
	GetMarketBalance(ctx context.Context, market prediction.Market) (map[prediction.OutcomeID]clobtypes.BalanceAllowanceResponse, error)
	RedeemPosition(ctx context.Context, market prediction.Market) (string, error)
	// SplitPosition deposits amountUSDC and mints YES+NO tokens (6 decimal units).
	// Returns the tx hash immediately and a ready channel that closes once the tx
	// is mined and the CLOB balance cache is refreshed. Callers MUST drain ready
	// before placing SELL orders.
	SplitPosition(ctx context.Context, market prediction.Market, amountUSDC *big.Int) (txHash string, ready <-chan error, err error)
	// MergePositions burns YES+NO tokens and returns amountUSDC (6 decimal units).
	MergePositions(ctx context.Context, market prediction.Market, amountUSDC *big.Int) (string, error)
	// GetLockedPositions returns all CTF ERC-1155 positions currently held on-chain
	// by the signing EOA. Uses Alchemy NFT + transfers APIs — requires an Alchemy
	// Polygon RPC URL; returns empty slice when none is configured.
	GetLockedPositions(ctx context.Context) ([]prediction.LockedPosition, error)
	// SetupApprovals grants the exchange contracts all required ERC-20 and ERC-1155
	// approvals so that CLOB order settlement can proceed without on-chain reverts.
	// Safe to call multiple times — each approval is a no-op if already granted.
	SetupApprovals(ctx context.Context) error
	// ConfirmConditionalBalance triggers a CLOB balance refresh for every outcome
	// in the market and polls until each reports a balance of at least minAmount.
	// Use after CLOB buy fills to wait for settlement before MergePositions.
	ConfirmConditionalBalance(ctx context.Context, market prediction.Market, minAmount *big.Int) error
	// GetNativeBalance returns the native MATIC balance (in wei) for the signing address.
	// Requires polygon_rpc_url to be configured; returns 0 when no RPC is available.
	GetNativeBalance(ctx context.Context) (*big.Int, error)
}

// orderManager implementation
type orderManager struct {
	client          clob.Client
	tokenManagement ctf.Client
	signer          auth.Signer
	rpcURL          string
	sigType         auth.SignatureType
	safeAddr        common.Address
}

func NewOrderManager(
	client clob.Client,
	manager ctf.Client,
	signer auth.Signer,
	rpcURL string,
	sigType auth.SignatureType,
	safeAddr common.Address,
) OrderManager {
	return &orderManager{
		client:          client,
		tokenManagement: manager,
		signer:          signer,
		rpcURL:          rpcURL,
		sigType:         sigType,
		safeAddr:        safeAddr,
	}
}
