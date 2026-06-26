package github

import (
	"context"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// RetryConfig controls how the client retries failed HTTP requests.
//
// Backoff: exponential with full jitter. delay = rand[0, base * 2^attempt),
// capped at MaxDelay. MaxAttempts is the total number of attempts
// (1 = no retry). After exhausting attempts, the last response/error is
// returned.
//
// On HTTP 403 (rate-limited) or 429, the client honors X-RateLimit-Reset
// when present and the suggested wait is shorter than MaxDelay.
type RetryConfig struct {
	MaxAttempts int           // total attempts; <=1 disables retry
	BaseDelay   time.Duration // first backoff window
	MaxDelay    time.Duration // cap per-attempt wait
}

func (r RetryConfig) withDefaults() RetryConfig {
	if r.MaxAttempts < 1 {
		r.MaxAttempts = 1
	}
	if r.BaseDelay <= 0 {
		r.BaseDelay = 1 * time.Second
	}
	if r.MaxDelay <= 0 {
		r.MaxDelay = 60 * time.Second
	}
	return r
}

// DefaultRetry is the package default used when Client.RetryConfig is unset.
var DefaultRetry = RetryConfig{
	MaxAttempts: 4,
	BaseDelay:   1 * time.Second,
	MaxDelay:    60 * time.Second,
}

// doGetWithRetry wraps Client.doGet with bounded exponential-backoff retries.
// It is safe for the caller to pass ctx == nil (background call).
//
// Retried status codes: 429, 403 (rate-limited only), and 5xx. Network
// errors (timeout, EOF, connection reset) are also retried. Other
// non-2xx responses and decoding errors return immediately.
func (c *Client) doGetWithRetry(ctx context.Context, path, accept string) (*http.Response, error) {
	cfg := c.retryConfig.withDefaults()

	var lastResp *http.Response
	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastResp != nil {
				return lastResp, err
			}
			return nil, err
		}

		resp, err := c.doGet(path, accept)
		lastResp, lastErr = resp, err

		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		if !shouldRetry(resp, err) {
			return resp, err
		}

		// Last attempt: stop here, return whatever we have.
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		wait := backoffDelay(cfg, attempt, resp)
		if cerr := sleepWithContext(ctx, wait); cerr != nil {
			if resp != nil {
				return resp, cerr
			}
			return nil, cerr
		}
	}
	return lastResp, lastErr
}

// shouldRetry reports whether a response/error pair warrants another attempt.
func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		// Network/transport errors are retryable.
		return true
	}
	if resp == nil {
		return true
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return true
	case http.StatusForbidden:
		// GitHub returns 403 for rate-limit only when X-RateLimit-Remaining == 0.
		if rem := resp.Header.Get("X-RateLimit-Remaining"); rem == "" || rem == "0" {
			return true
		}
		return false
	}
	if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
		return true
	}
	return false
}

// backoffDelay returns the wait duration before the next attempt.
//
// When the server hints a reset time (X-RateLimit-Reset, Unix seconds) and
// the suggested wait is shorter than the exponential cap, the smaller value
// wins. Otherwise it falls back to exp-backoff with full jitter.
func backoffDelay(cfg RetryConfig, attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if reset := parseRateLimitReset(resp.Header.Get("X-RateLimit-Reset")); !reset.IsZero() {
			wait := time.Until(reset)
			if wait > 0 && wait < cfg.MaxDelay {
				return wait
			}
		}
	}
	exp := float64(cfg.BaseDelay) * math.Pow(2, float64(attempt))
	if exp > float64(cfg.MaxDelay) {
		exp = float64(cfg.MaxDelay)
	}
	// Full jitter: uniform in [0, exp).
	return time.Duration(rand.Int64N(int64(exp)))
}

// parseRateLimitReset parses an X-RateLimit-Reset header. GitHub sends a
// Unix timestamp (seconds). Returns zero when the header is missing or
// unparseable.
func parseRateLimitReset(v string) time.Time {
	if v == "" {
		return time.Time{}
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(n, 0)
}

// sleepWithContext waits for d or until ctx is done. Returns ctx.Err() on
// cancellation, nil on a clean sleep.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
