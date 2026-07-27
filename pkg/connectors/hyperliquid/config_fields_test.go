package hyperliquid

import (
	"encoding/json"
	"testing"
)

func TestHyperliquidZeroJSONKeysMatchValidateRequired(t *testing.T) {
	b, err := json.Marshal(&Config{})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"private_key", "account_address"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing %q: %v", k, m)
		}
	}
	// optional omitempty
	for _, k := range []string{"base_url", "vault_address", "use_testnet"} {
		if _, ok := m[k]; ok {
			t.Errorf("%q should be omitted when zero: %v", k, m)
		}
	}
}
