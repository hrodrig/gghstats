package server

import (
	"regexp"
	"strings"
)

// GitHub-flavored emoji shortcodes like :whale: or :rocket:.
var emojiShortcodeRE = regexp.MustCompile(`:[a-zA-Z0-9_+-]+:`)

// stripEmojiShortcodes removes :name: tokens and collapses leftover whitespace.
// Featured descriptions often arrive with raw shortcodes from GitHub (#129).
func stripEmojiShortcodes(s string) string {
	if s == "" {
		return s
	}
	cleaned := emojiShortcodeRE.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(cleaned), " ")
}
