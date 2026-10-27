package server

import "crypto/subtle"

// tokenMatches reports whether presented equals configured, comparing both in
// constant time so that a rejected token does not leak how many leading bytes
// were correct through response timing.
//
// crypto/subtle returns early when the two inputs have different lengths, so
// the length of the token still leaks. That is accepted: the length of the
// configured token is not a secret, and hiding it would require comparing
// against a padded copy of the configured value on every request.
func tokenMatches(presented, configured string) bool {
	if configured == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented), []byte(configured)) == 1
}
