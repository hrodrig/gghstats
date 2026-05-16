package server

import (
	"fmt"
	"strings"
	"unicode"
)

// parseSyncRepoQuery validates optional ?repo=owner/name for POST /api/v1/sync.
func parseSyncRepoQuery(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	owner, name, ok := strings.Cut(raw, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return "", fmt.Errorf("repo must be owner/name")
	}
	if !isGitHubNameSegment(owner) || !isGitHubNameSegment(name) {
		return "", fmt.Errorf("invalid owner or repo name")
	}
	return owner + "/" + name, nil
}

func isGitHubNameSegment(s string) bool {
	if s == "" || len(s) > 100 {
		return false
	}
	for _, r := range s {
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
