package uniswap_v3

import (
	"testing"
)

func TestConfigValidate_RequiresRPCAndChain(t *testing.T) {
	c := &Config{}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty config")
	}
	c.RPCURL = "https://example.invalid"
	if err := c.Validate(); err == nil {
		t.Fatal("expected error without chain_id")
	}
	c.ChainID = 1
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !c.DryRun {
		t.Fatal("empty private_key should force DryRun")
	}
	if c.DefaultFeeTier != 10000 {
		t.Fatalf("default fee tier want 10000 got %d", c.DefaultFeeTier)
	}
}

func TestConfigValidate_LiveNeedsAccount(t *testing.T) {
	c := &Config{
		RPCURL:     "https://example.invalid",
		ChainID:    1,
		PrivateKey: "ab",
		DryRun:     false,
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected account_address required when not dry_run")
	}
	c.AccountAddress = "0x0000000000000000000000000000000000000001"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
