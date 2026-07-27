package config

import (
	"encoding/json"
	"testing"
)

// Ensures zero-value JSON keys match Validate() required credentials
// (GetRequiredCredentialFields discovery path).
func TestPolymarketZeroJSONKeysMatchValidateRequired(t *testing.T) {
	b, err := json.Marshal(&Config{})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	// required by Validate
	for _, k := range []string{"private_key", "polygon_rpc_url"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing required json key %q in zero Config: %v", k, m)
		}
	}
	// optional — omitempty
	if _, ok := m["polymarket_address"]; ok {
		t.Errorf("polymarket_address should be omitempty on zero value: %v", m)
	}
}
