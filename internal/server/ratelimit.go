package server

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter is a per-IP token-bucket rate limiter.
type RateLimiter struct {
	limiters sync.Map // string → *rateLimiterEntry
	rate     rate.Limit
	burst    int
	maxIdle  time.Duration
	done     chan struct{}
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitConfig holds rate limiter parameters.
type RateLimitConfig struct {
	Enabled  bool
	Requests int           // request count per period (default 120)
	Period   time.Duration // time window (default 1 minute)
	Burst    int           // burst size (default 20)
}

// DefaultRateLimitConfig returns the built-in defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:  true,
		Requests: 120,
		Period:   1 * time.Minute,
		Burst:    20,
	}
}

// NewRateLimiter creates a per-IP token bucket rate limiter. It starts a
// background goroutine that cleans up idle entries; call Shutdown to stop it.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	r := rate.Limit(float64(cfg.Requests) / cfg.Period.Seconds())
	if r <= 0 {
		slog.Warn("rate limit requests/period is zero or negative, disabling",
			"requests", cfg.Requests, "period", cfg.Period)
		r = rate.Inf
	}
	rl := &RateLimiter{
		rate:    r,
		burst:   cfg.Burst,
		maxIdle: 5 * time.Minute,
		done:    make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// Shutdown stops the background cleanup goroutine.
func (rl *RateLimiter) Shutdown() {
	close(rl.done)
}

// Middleware wraps next with per-IP rate limiting. Paths matching skipPaths
// (exact match on the request URL path) are exempt.
func (rl *RateLimiter) Middleware(next http.Handler, skipPaths ...string) http.Handler {
	skip := make(map[string]struct{}, len(skipPaths))
	for _, p := range skipPaths {
		skip[p] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := skip[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}
		ip := clientIP(r)
		entry := rl.getOrCreate(ip)
		if !entry.limiter.Allow() {
			w.Header().Set("Retry-After", "60")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate_limit_exceeded"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getOrCreate(ip string) *rateLimiterEntry {
	if v, ok := rl.limiters.Load(ip); ok {
		e := v.(*rateLimiterEntry)
		e.lastSeen = time.Now()
		return e
	}
	entry := &rateLimiterEntry{
		limiter:  rate.NewLimiter(rl.rate, rl.burst),
		lastSeen: time.Now(),
	}
	v, _ := rl.limiters.LoadOrStore(ip, entry)
	return v.(*rateLimiterEntry)
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

func (rl *RateLimiter) cleanup() {
	now := time.Now()
	rl.limiters.Range(func(key, value any) bool {
		e := value.(*rateLimiterEntry)
		if now.Sub(e.lastSeen) > rl.maxIdle {
			rl.limiters.Delete(key)
		}
		return true
	})
}

// clientIP extracts the client IP from X-Forwarded-For, X-Real-IP, or RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the leftmost (original client) IP.
		ip := strings.TrimSpace(strings.Split(xff, ",")[0])
		if ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ParseRateLimitEnv reads env vars and returns a RateLimitConfig.
// Recognised keys: GGHSTATS_RATE_LIMIT_ENABLED, GGHSTATS_RATE_LIMIT_REQUESTS,
// GGHSTATS_RATE_LIMIT_PERIOD, GGHSTATS_RATE_LIMIT_BURST.
func ParseRateLimitEnv() RateLimitConfig {
	cfg := DefaultRateLimitConfig()

	if v := strings.TrimSpace(os.Getenv("GGHSTATS_RATE_LIMIT_ENABLED")); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "no", "off":
			cfg.Enabled = false
		}
	}

	if v := strings.TrimSpace(os.Getenv("GGHSTATS_RATE_LIMIT_REQUESTS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Requests = n
		}
	}

	if v := strings.TrimSpace(os.Getenv("GGHSTATS_RATE_LIMIT_PERIOD")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.Period = d
		}
	}

	if v := strings.TrimSpace(os.Getenv("GGHSTATS_RATE_LIMIT_BURST")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.Burst = n
		}
	}

	return cfg
}
