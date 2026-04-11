package order_manager

import (
	"context"
	"fmt"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/clob"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	"github.com/ethereum/go-ethereum/common"
)

// SetupApprovals grants all on-chain approvals required for CLOB order settlement.
//
// Every CLOB order type needs a specific exchange to pull funds from the maker:
//   - BUY on NegRisk market:     NegRisk Exchange pulls USDC from EOA
//   - BUY on standard market:    CTF Exchange pulls USDC from EOA
//   - SELL on NegRisk market:    NegRisk Exchange pulls YES tokens (ERC-1155) from EOA
//   - SELL on standard market:   CTF Exchange pulls YES tokens (ERC-1155) from EOA
//
// Note: USDC → ConditionalTokens is handled per-split inside SplitPosition (not here).
// All four approvals here are idempotent — each checks the current on-chain state and
// skips the transaction if the approval is already sufficient.
func (c *orderManager) SetupApprovals(ctx context.Context) error {
	usdcAddr := common.HexToAddress(usdcAddressHex)
	negRiskAddr := common.HexToAddress(ctf.NegRiskExchangeAddress)
	ctfExchangeAddr := common.HexToAddress(clob.CTFExchangeAddress)

	// 1. USDC → NegRisk Exchange (BUY settlement on NegRisk markets)
	if err := c.tokenManagement.EnsureERC20Approved(ctx, usdcAddr, negRiskAddr); err != nil {
		return fmt.Errorf("approve USDC for NegRisk exchange: %w", err)
	}

	// 2. USDC → CTF Exchange (BUY settlement on standard binary markets)
	if err := c.tokenManagement.EnsureERC20Approved(ctx, usdcAddr, ctfExchangeAddr); err != nil {
		return fmt.Errorf("approve USDC for CTF exchange: %w", err)
	}

	// 3. Conditional tokens → NegRisk Exchange (SELL settlement on NegRisk markets)
	if err := c.tokenManagement.EnsureConditionalApproved(ctx, negRiskAddr); err != nil {
		return fmt.Errorf("approve conditional tokens for NegRisk exchange: %w", err)
	}

	// 4. Conditional tokens → CTF Exchange (SELL settlement on standard binary markets)
	if err := c.tokenManagement.EnsureConditionalApproved(ctx, ctfExchangeAddr); err != nil {
		return fmt.Errorf("approve conditional tokens for CTF exchange: %w", err)
	}

	return nil
}
