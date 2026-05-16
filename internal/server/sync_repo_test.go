package server

import "testing"

func TestParseSyncRepoQuery(t *testing.T) {
	tests := []struct {
		in    string
		want  string
		errOK bool
	}{
		{"", "", false},
		{"hrodrig/pgwd", "hrodrig/pgwd", false},
		{"bad/ space", "", true},
		{"owner", "", true},
		{"a/b/c", "", true},
		{"hrodrig/my-app", "hrodrig/my-app", false},
	}
	for _, tc := range tests {
		got, err := parseSyncRepoQuery(tc.in)
		if tc.errOK {
			if err == nil {
				t.Errorf("parseSyncRepoQuery(%q) err = nil, want error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSyncRepoQuery(%q) unexpected err: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseSyncRepoQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
