package alert

import (
	"strings"
	"testing"
)

func TestParseRulesJSON_ScopeWindowCombination(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name: "all_repos with lifetime is valid",
			raw:  `[{"scope":"all_repos","metric":"clones","window":"lifetime","op":"gte","value":30000}]`,
		},
		{
			name:    "all_repos with 7d is rejected",
			raw:     `[{"scope":"all_repos","metric":"clones","window":"7d","op":"gte","value":10}]`,
			wantErr: true,
		},
		{
			name:    "all_repos with 30d is rejected",
			raw:     `[{"scope":"all_repos","metric":"clones","window":"30d","op":"gte","value":10}]`,
			wantErr: true,
		},
		{
			name:    "all_repos with 1d is rejected",
			raw:     `[{"scope":"all_repos","metric":"clones","window":"1d","op":"gte","value":10}]`,
			wantErr: true,
		},
		{
			name:    "all_repos with wow is rejected",
			raw:     `[{"scope":"all_repos","metric":"clones","window":"wow","op":"drop_pct","value":30}]`,
			wantErr: true,
		},
		{
			name: "each_repo with 7d is valid",
			raw:  `[{"scope":"each_repo","metric":"views","window":"7d","op":"lt","value":50}]`,
		},
		{
			name: "repo with 1d is valid",
			raw:  `[{"repo":"hrodrig/pgwd","metric":"views","window":"1d","op":"lt","value":10}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRulesJSON(tt.raw)
			if tt.wantErr && err == nil {
				t.Fatalf("ParseRulesJSON(%s) succeeded, want an error", tt.raw)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ParseRulesJSON(%s) error = %v, want nil", tt.raw, err)
			}
		})
	}
}

// TestParseRulesJSON_ScopeWindowErrorNamesScopeAndWindow keeps the failure
// actionable: the operator has to see which scope and which window are
// supported without reading the source.
func TestParseRulesJSON_ScopeWindowErrorNamesScopeAndWindow(t *testing.T) {
	_, err := ParseRulesJSON(`[{"scope":"all_repos","metric":"clones","window":"7d","op":"gte","value":10}]`)
	if err == nil {
		t.Fatal("ParseRulesJSON succeeded, want an error")
	}

	for _, want := range []string{"all_repos", "lifetime", "7d"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}
