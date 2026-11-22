package server

import "testing"

func TestStripEmojiShortcodes(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"plain text", "plain text"},
		{":whale: A curated list of Docker resources", "A curated list of Docker resources"},
		{"Hello :rocket: world :sparkles:", "Hello world"},
		{"  :x:  spaced  :ok_hand:  ", "spaced"},
		{"no_colons_here", "no_colons_here"},
		{":custom-name:", ""},
		{":123:", ""},
	}
	for _, tc := range tests {
		if got := stripEmojiShortcodes(tc.in); got != tc.want {
			t.Errorf("stripEmojiShortcodes(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
