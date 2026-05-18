package metrics

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestClassifyGitHubOutcome(t *testing.T) {
	tests := []struct {
		name string
		resp *http.Response
		err  error
		want string
	}{
		{"error", nil, errors.New("network"), "error"},
		{"nil response", nil, nil, "error"},
		{"ok", &http.Response{StatusCode: http.StatusOK}, nil, "success"},
		{"429", &http.Response{StatusCode: http.StatusTooManyRequests}, nil, "rate_limited"},
		{"403 rate limit", &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"X-RateLimit-Remaining": []string{"0"}},
		}, nil, "rate_limited"},
		{"403 other", &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"X-RateLimit-Remaining": []string{"100"}},
		}, nil, "rate_limited"},
		{"500", &http.Response{StatusCode: http.StatusInternalServerError}, nil, "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyGitHubOutcome(tt.resp, tt.err); got != tt.want {
				t.Errorf("ClassifyGitHubOutcome() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGitHubRateLimitRemaining(t *testing.T) {
	if _, ok := GitHubRateLimitRemaining(nil); ok {
		t.Fatal("nil response should not report remaining")
	}
	resp := &http.Response{Header: http.Header{"X-RateLimit-Remaining": []string{"not-a-number"}}}
	if _, ok := GitHubRateLimitRemaining(resp); ok {
		t.Fatal("invalid header should not report remaining")
	}
	resp = &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-RateLimit-Remaining", "7")
	if n, ok := GitHubRateLimitRemaining(resp); !ok || n != 7 {
		t.Fatalf("remaining = %d, ok = %v, want 7 true", n, ok)
	}
}

func TestIsTimeout(t *testing.T) {
	if IsTimeout(errors.New("plain")) {
		t.Fatal("plain error is not a timeout")
	}
	if !IsTimeout(timeoutErr{}) {
		t.Fatal("timeoutErr should be a timeout")
	}
	if !IsTimeout(context.DeadlineExceeded) {
		t.Fatal("DeadlineExceeded should be a timeout")
	}
}
