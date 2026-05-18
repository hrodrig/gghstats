package metrics

import (
	"errors"
	"net/http"
	"strconv"
)

// ClassifyGitHubOutcome maps an HTTP response and error to a metric status label.
func ClassifyGitHubOutcome(resp *http.Response, err error) string {
	if err != nil {
		return "error"
	}
	if resp == nil {
		return "error"
	}
	if resp.StatusCode == http.StatusOK {
		return "success"
	}
	if resp.StatusCode == http.StatusForbidden {
		if rem, parseErr := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining")); parseErr == nil && rem == 0 {
			return "rate_limited"
		}
		return "rate_limited"
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return "rate_limited"
	}
	return "error"
}

// GitHubRateLimitRemaining parses X-RateLimit-Remaining when present.
func GitHubRateLimitRemaining(resp *http.Response) (int, bool) {
	if resp == nil {
		return 0, false
	}
	v := resp.Header.Get("X-RateLimit-Remaining")
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

// IsTimeout reports whether err is a client timeout (for future use).
func IsTimeout(err error) bool {
	var ne interface{ Timeout() bool }
	return errors.As(err, &ne) && ne.Timeout()
}
