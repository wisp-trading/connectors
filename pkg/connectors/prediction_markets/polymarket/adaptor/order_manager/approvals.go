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

	fmt.Printf("[polymarket:approvals] starting approval setup\n")
	fmt.Printf("[polymarket:approvals]   NegRisk Exchange : %s\n", negRiskAddr.Hex())
	fmt.Printf("[polymarket:approvals]   CTF Exchange     : %s\n", ctfExchangeAddr.Hex())

	// 1. USDC → NegRisk Exchange (BUY settlement on NegRisk markets)
	fmt.Printf("[polymarket:approvals] checking USDC → NegRisk Exchange...\n")
	if err := c.tokenManagement.EnsureERC20Approved(ctx, usdcAddr, negRiskAddr); err != nil {
		return fmt.Errorf("approve USDC for NegRisk exchange: %w", err)
	}
	fmt.Printf("[polymarket:approvals] ✓ USDC → NegRisk Exchange\n")

	// 2. USDC → CTF Exchange (BUY settlement on standard binary markets)
	fmt.Printf("[polymarket:approvals] checking USDC → CTF Exchange...\n")
	if err := c.tokenManagement.EnsureERC20Approved(ctx, usdcAddr, ctfExchangeAddr); err != nil {
		return fmt.Errorf("approve USDC for CTF exchange: %w", err)
	}
	fmt.Printf("[polymarket:approvals] ✓ USDC → CTF Exchange\n")

	// 3. Conditional tokens → NegRisk Exchange (SELL settlement on NegRisk markets)
	fmt.Printf("[polymarket:approvals] checking ERC-1155 setApprovalForAll → NegRisk Exchange...\n")
	if err := c.tokenManagement.EnsureConditionalApproved(ctx, negRiskAddr); err != nil {
		return fmt.Errorf("approve conditional tokens for NegRisk exchange: %w", err)
	}
	fmt.Printf("[polymarket:approvals] ✓ ERC-1155 → NegRisk Exchange\n")

	// 4. Conditional tokens → CTF Exchange (SELL settlement on standard binary markets)
	fmt.Printf("[polymarket:approvals] checking ERC-1155 setApprovalForAll → CTF Exchange...\n")
	if err := c.tokenManagement.EnsureConditionalApproved(ctx, ctfExchangeAddr); err != nil {
		return fmt.Errorf("approve conditional tokens for CTF exchange: %w", err)
	}
	fmt.Printf("[polymarket:approvals] ✓ ERC-1155 → CTF Exchange\n")

	fmt.Printf("[polymarket:approvals] all approvals confirmed\n")
	return nil
}
