# Trusted proxies + HTTP timeouts (v0.10.2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship **v0.10.2** with SEC1 (trusted proxies for client IP) and SEC2 (`http.Server` timeouts), plus operator docs (problem → approach → situations A/B/C).

**Architecture:** New `TrustedProxies` type in `internal/server`; `clientIP` trusts `X-Forwarded-For` / `X-Real-IP` only when TCP peer ∈ that set. Inject the same pointer into `RateLimiter` and `Whitelist`. Serve startup parses `GGHSTATS_TRUSTED_PROXIES`, warns if RL/whitelist need real client IPs but the list is empty, and sets fixed server timeouts.

**Tech Stack:** Go 1.x stdlib (`net`, `net/http`, `log/slog`), existing `internal/server` middleware, English docs (README / man / env.example / SPEC / CHANGELOG).

**Spec:** [docs/superpowers/specs/2026-07-22-trusted-proxies-timeouts-design.md](../specs/2026-07-22-trusted-proxies-timeouts-design.md)

## Global Constraints

- Run all commands from the **repository root** (clone path is local; do not hard-code machine-specific absolute paths in docs or commands).
- English only for all project artifacts.
- Work on branch `develop`; do not commit to `main` until release merge.
- Do not delete files without explicit user approval.
- Before each `git commit`, show the full message and wait for user approval (project rule).
- Keep `golang.org/x/net v0.57.0 // indirect` pin; do not run `go mod tidy` in a way that drops it without re-pinning.
- Breaking change: empty `GGHSTATS_TRUSTED_PROXIES` ignores XFF/XRI (was unconditional trust).
- No new timeout env knobs in 0.10.2.
- Thin leaderboard / API-only / SEC3–SEC5 out of scope.

## File map

| File | Role |
|------|------|
| `internal/server/trusted.go` | `TrustedProxies`, parse, contains, boot-warn helper |
| `internal/server/trusted_test.go` | Parse + contains + warn condition tests |
| `internal/server/ratelimit.go` | `clientIP(r, trusted)`, `RateLimiter.trusted` |
| `internal/server/ratelimit_test.go` | Rewrite `TestClientIP` for trusted/empty cases |
| `internal/server/whitelist.go` | `Whitelist.trusted`; pass into `clientIP` |
| `internal/server/whitelist_test.go` | Fix XFF middleware test to set trusted peer |
| `internal/server/server.go` | `Config.TrustedProxies`; wire into RL/whitelist in `New` |
| `cmd/gghstats/serve.go` | Parse env, warn, timeouts on `http.Server`, pass into `Config` |
| README, env.example, man, SPEC, CHANGELOG, plan-v0.10/0.11, VERSION, badge | Docs + release bump |

---

### Task 1: TrustedProxies type + failing clientIP tests

**Files:**
- Create: `internal/server/trusted.go`
- Create: `internal/server/trusted_test.go`
- Modify: `internal/server/ratelimit.go` (`clientIP` signature — stub may not compile until Task 2 wires callers)
- Modify: `internal/server/ratelimit_test.go`
- Test: `go test ./internal/server/ -count=1 -run 'TestClientIP|TestParseTrustedProxies|TestTrustedProxiesContains'`

**Interfaces:**
- Consumes: none
- Produces:
  - `type TrustedProxies struct { nets []*net.IPNet }` (unexported field OK)
  - `func ParseTrustedProxies(s string) *TrustedProxies` — empty/`nil` nets → treat as empty (never trust headers)
  - `func (t *TrustedProxies) ContainsIP(ip net.IP) bool` — `nil` receiver or empty → false
  - `func clientIP(r *http.Request, trusted *TrustedProxies) string` — rules per design

- [ ] **Step 1: Write failing `TestClientIP` cases** (replace existing table in `ratelimit_test.go`)

```go
func TestClientIP(t *testing.T) {
	trustedLAN := ParseTrustedProxies("10.0.0.0/8")
	tests := []struct {
		name          string
		trusted       *TrustedProxies
		xForwardedFor string
		xRealIP       string
		remoteAddr    string
		want          string
	}{
		{
			name:          "empty trusted ignores xff forge",
			trusted:       nil,
			xForwardedFor: "203.0.113.1",
			remoteAddr:    "198.51.100.9:12345",
			want:          "198.51.100.9",
		},
		{
			name:          "empty trusted ignores x-real-ip",
			trusted:       ParseTrustedProxies(""),
			xRealIP:       "203.0.113.1",
			remoteAddr:    "198.51.100.9:12345",
			want:          "198.51.100.9",
		},
		{
			name:          "untrusted peer ignores xff",
			trusted:       trustedLAN,
			xForwardedFor: "10.5.5.5",
			remoteAddr:    "198.51.100.9:12345",
			want:          "198.51.100.9",
		},
		{
			name:          "trusted peer uses leftmost xff",
			trusted:       trustedLAN,
			xForwardedFor: "10.5.5.5, 172.16.0.1",
			remoteAddr:    "10.0.0.1:12345",
			want:          "10.5.5.5",
		},
		{
			name:       "trusted peer uses x-real-ip",
			trusted:    trustedLAN,
			xRealIP:    "10.9.9.9",
			remoteAddr: "10.0.0.1:12345",
			want:       "10.9.9.9",
		},
		{
			name:          "xff beats x-real-ip when trusted",
			trusted:       trustedLAN,
			xForwardedFor: "10.4.4.4",
			xRealIP:       "10.8.8.8",
			remoteAddr:    "10.0.0.1:9999",
			want:          "10.4.4.4",
		},
		{
			name:          "trusted peer garbage xff falls back to peer",
			trusted:       trustedLAN,
			xForwardedFor: "not-an-ip",
			remoteAddr:    "10.0.0.1:12345",
			want:          "10.0.0.1",
		},
		{
			name:       "remote-addr no port",
			trusted:    nil,
			remoteAddr: "192.168.1.1",
			want:       "192.168.1.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			if tt.xForwardedFor != "" {
				h.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				h.Set("X-Real-IP", tt.xRealIP)
			}
			r := &http.Request{Header: h, RemoteAddr: tt.remoteAddr}
			if got := clientIP(r, tt.trusted); got != tt.want {
				t.Errorf("clientIP = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Write `TestParseTrustedProxies` / `TestTrustedProxiesContains`** in `trusted_test.go`

```go
func TestParseTrustedProxies(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		tp := ParseTrustedProxies("  ")
		if tp == nil || tp.ContainsIP(net.ParseIP("10.0.0.1")) {
			t.Fatalf("empty must not contain any IP; tp=%v", tp)
		}
	})
	t.Run("single ip becomes /32", func(t *testing.T) {
		tp := ParseTrustedProxies("127.0.0.1")
		if !tp.ContainsIP(net.ParseIP("127.0.0.1")) {
			t.Fatal("expected 127.0.0.1 contained")
		}
		if tp.ContainsIP(net.ParseIP("127.0.0.2")) {
			t.Fatal("did not expect 127.0.0.2")
		}
	})
	t.Run("ipv6 single becomes /128", func(t *testing.T) {
		tp := ParseTrustedProxies("::1")
		if !tp.ContainsIP(net.ParseIP("::1")) {
			t.Fatal("expected ::1 contained")
		}
	})
	t.Run("skips invalid", func(t *testing.T) {
		tp := ParseTrustedProxies("not-an-ip, 10.0.0.0/8")
		if !tp.ContainsIP(net.ParseIP("10.1.2.3")) {
			t.Fatal("expected 10.1.2.3 in 10.0.0.0/8")
		}
	})
}
```

- [ ] **Step 3: Run tests — expect FAIL** (missing types / old `clientIP` signature)

```bash
go test ./internal/server/ -count=1 -run 'TestClientIP|TestParseTrustedProxies|TestTrustedProxiesContains'
```

Expected: compile error or FAIL on forge cases still returning XFF.

- [ ] **Step 4: Implement `trusted.go` + new `clientIP`**

```go
// trusted.go
package server

import (
	"net"
	"strings"
)

// TrustedProxies is the set of TCP peers allowed to supply X-Forwarded-For / X-Real-IP.
type TrustedProxies struct {
	nets []*net.IPNet
}

// ParseTrustedProxies parses comma-separated IPs/CIDRs.
// Bare IPv4 → /32; bare IPv6 → /128. Invalid entries are skipped.
// Empty input returns a non-nil empty set (ContainsIP always false).
func ParseTrustedProxies(s string) *TrustedProxies {
	tp := &TrustedProxies{}
	for _, raw := range strings.Split(s, ",") {
		cidr := strings.TrimSpace(raw)
		if cidr == "" {
			continue
		}
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		tp.nets = append(tp.nets, n)
	}
	return tp
}

// ContainsIP reports whether ip is inside any trusted network.
func (t *TrustedProxies) ContainsIP(ip net.IP) bool {
	if t == nil || ip == nil {
		return false
	}
	for _, n := range t.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func (t *TrustedProxies) empty() bool {
	return t == nil || len(t.nets) == 0
}
```

Replace `clientIP` in `ratelimit.go`:

```go
func clientIP(r *http.Request, trusted *TrustedProxies) string {
	peer := peerIP(r.RemoteAddr)
	peerParsed := net.ParseIP(peer)
	if trusted.empty() || peerParsed == nil || !trusted.ContainsIP(peerParsed) {
		return peer
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip := strings.TrimSpace(strings.Split(xff, ",")[0])
		if ip != "" && net.ParseIP(ip) != nil {
			return ip
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}
	return peer
}

func peerIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
```

- [ ] **Step 5: Fix compile — temporary callers**

Until Task 2 finishes wiring, update every `clientIP(r)` call site to `clientIP(r, nil)` so the package builds, then run tests again.

```bash
go test ./internal/server/ -count=1 -run 'TestClientIP|TestParseTrustedProxies|TestTrustedProxiesContains'
```

Expected: PASS for those tests. Other middleware tests that relied on XFF with `nil` trusted may FAIL — fix in Task 2.

- [ ] **Step 6: Commit** (after user approves message)

```text
Add TrustedProxies and secure clientIP (SEC1 core)
```

---

### Task 2: Wire trusted proxies into RateLimiter, Whitelist, Config, serve

**Files:**
- Modify: `internal/server/ratelimit.go` — field + setter or set in `New`
- Modify: `internal/server/whitelist.go`
- Modify: `internal/server/server.go` — `Config.TrustedProxies`; in `New`, assign to RL/whitelist
- Modify: `cmd/gghstats/serve.go` — parse env, warn, pass `TrustedProxies`
- Modify: `internal/server/whitelist_test.go`, any ratelimit middleware tests using XFF
- Create/extend: `internal/server/trusted_test.go` for `ShouldWarnTrustedProxies`
- Test: `go test ./internal/server/ ./cmd/gghstats/ -count=1`

**Interfaces:**
- Consumes: `ParseTrustedProxies`, `clientIP(r, trusted)`
- Produces:
  - `Config.TrustedProxies *TrustedProxies`
  - `func ShouldWarnTrustedProxies(trusted *TrustedProxies, rateLimitEnabled bool, whitelistActive bool) bool`
  - `func WarnTrustedProxiesIfNeeded(...)` using `slog.Warn` with the design message
  - RateLimiter / Whitelist call `clientIP(r, their.trusted)`

- [ ] **Step 1: Add fields and wire `New`**

```go
// Config in server.go
TrustedProxies *TrustedProxies

// In New(cfg Config), after normalizeLocaleConfig, before building handler chain:
if cfg.RateLimiter != nil {
	cfg.RateLimiter.trusted = cfg.TrustedProxies
}
if cfg.Whitelist != nil {
	cfg.Whitelist.trusted = cfg.TrustedProxies
}
```

Add unexported `trusted *TrustedProxies` on both structs; update Middleware to `clientIP(r, rl.trusted)` / `clientIP(r, w.trusted)`.

- [ ] **Step 2: Fix `TestWhitelistMiddlewareXForwardedFor`**

Set `RemoteAddr` to a peer inside the whitelist path’s trusted set, and attach trusted proxies matching that peer:

```go
func TestWhitelistMiddlewareXForwardedFor(t *testing.T) {
	w := NewWhitelist(WhitelistConfig{CIDRs: "10.0.0.0/8"}, "")
	w.trusted = ParseTrustedProxies("192.168.1.0/24")
	handler := w.Middleware(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}), MiddlewareSkip{})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "10.5.5.5, 172.16.0.1")
	req.RemoteAddr = "192.168.1.1:12345" // trusted peer
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want 200 (trusted peer + XFF)", rec.Code)
	}
}
```

Add a sibling test: untrusted peer + forged XFF claiming `10.5.5.5` → **403** when whitelist is `10.0.0.0/8`.

- [ ] **Step 3: Boot warn helpers**

```go
func ShouldWarnTrustedProxies(trusted *TrustedProxies, rateLimitEnabled, whitelistActive bool) bool {
	if !rateLimitEnabled && !whitelistActive {
		return false
	}
	return trusted.empty()
}

func WarnTrustedProxiesIfNeeded(trusted *TrustedProxies, rateLimitEnabled, whitelistActive bool) {
	if !ShouldWarnTrustedProxies(trusted, rateLimitEnabled, whitelistActive) {
		return
	}
	slog.Warn("Client IP headers (X-Forwarded-For / X-Real-IP) are ignored because GGHSTATS_TRUSTED_PROXIES is empty. Behind a reverse proxy, set GGHSTATS_TRUSTED_PROXIES to the proxy IP or CIDR so rate-limit and whitelist see the real client")
}
```

Unit-test `ShouldWarnTrustedProxies` truth table (4–5 cases).

- [ ] **Step 4: Wire `serve.go`**

Where handler `Config` is built (same place as RateLimiter / Whitelist):

```go
trusted := server.ParseTrustedProxies(os.Getenv("GGHSTATS_TRUSTED_PROXIES"))
rl := setupRateLimiter()
whitelist := server.NewWhitelist(server.ParseWhitelistEnv(), cfg.APIToken)
server.WarnTrustedProxiesIfNeeded(trusted, rl != nil, whitelist != nil)
// ...
handler := server.New(server.Config{
	// existing fields...
	RateLimiter:    rl,
	Whitelist:      whitelist,
	TrustedProxies: trusted,
})
```

- [ ] **Step 5: Run package tests**

```bash
go test ./internal/server/ ./cmd/gghstats/ -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit** (after user approves)

```text
Wire GGHSTATS_TRUSTED_PROXIES into serve and middleware
```

---

### Task 3: SEC2 http.Server timeouts

**Files:**
- Modify: `cmd/gghstats/serve.go` (~line 334 `http.Server{...}`)
- Test: `go test ./cmd/gghstats/ -count=1` and `go build -o /dev/null ./cmd/gghstats/`

**Interfaces:**
- Consumes: none
- Produces: timeouts on the live server

- [ ] **Step 1: Set timeouts**

```go
srv := &http.Server{
	Addr:              addr,
	Handler:           handler,
	ReadHeaderTimeout: 10 * time.Second,
	ReadTimeout:       30 * time.Second,
	WriteTimeout:      60 * time.Second,
	IdleTimeout:       120 * time.Second,
}
```

Ensure `time` is already imported in `serve.go`.

- [ ] **Step 2: Build + test**

```bash
go build -o /dev/null ./cmd/gghstats/
go test ./cmd/gghstats/ ./internal/server/ -count=1
```

Expected: PASS / build OK.

- [ ] **Step 3: Commit** (after user approves)

```text
Set http.Server read/write/idle timeouts (SEC2)
```

---

### Task 4: Operator documentation

**Files:**
- Modify: `README.md` — replace unconditional XFF paragraph under Rate limiting (~line 524)
- Modify: `contrib/gghstats.env.example`
- Modify: `contrib/man/man1/gghstats.1` — new `.TP` before or after rate-limit vars
- Modify: `SPEC.md` — short normative note near rate-limit/whitelist (after line ~35)
- Modify: `docs/plan-v0.11.x.md` — SEC1–SEC2 shipping in **0.10.2**
- Modify: `docs/plan-v0.10.x.md` — one-line patch note for 0.10.2
- Modify: `CHANGELOG.md` — under `[Unreleased]` (Security + Breaking)

**Doc content requirements (README):**

1. **Problem** — forged XFF/XRI when app is directly reachable.
2. **Approach** — trust headers only if peer ∈ `GGHSTATS_TRUSTED_PROXIES`; empty = ignore; boot warn when RL/whitelist on.
3. **Situations A/B/C** with env snippets (direct; Docker proxy CIDR; localhost proxy).
4. Warning: never `0.0.0.0/0`.

Example README block shape:

```markdown
### Client IP behind reverse proxies

**Problem.** Rate limiting and the optional IP whitelist key off the client IP.
If gghstats is reachable without a trusted hop, a client can send
`X-Forwarded-For: …` or `X-Real-IP: …` and spoof that identity.

**Approach.** Set `GGHSTATS_TRUSTED_PROXIES` to the reverse proxy’s IP or CIDR.
gghstats trusts forwarded headers **only** when the TCP peer is in that list.
If the list is empty (default), those headers are **ignored** and the peer
`RemoteAddr` is used. When rate limiting or a whitelist is active and the list
is empty, serve logs a warning at startup.

**How to apply**

**A — Direct exposure** (host port or Docker `-p`, no reverse proxy): leave unset.

```bash
# GGHSTATS_TRUSTED_PROXIES=
```

Forged `X-Forwarded-For` is ignored.

**B — Traefik / Caddy / nginx on a Docker network:** trust the proxy or bridge CIDR.

```bash
GGHSTATS_TRUSTED_PROXIES=172.16.0.0/12
```

Rate-limit buckets use the real client from XFF/XRI.

**C — Proxy on the host, app on localhost:**

```bash
GGHSTATS_TRUSTED_PROXIES=127.0.0.1/32,::1/128
```

Do **not** set `0.0.0.0/0` — that reopens header forgery.
```

- [ ] **Step 1: Edit README + env.example + man + SPEC + plans**
- [ ] **Step 2: Add CHANGELOG `[Unreleased]` entries**

```markdown
## [Unreleased]

### Security

- **Trusted proxies (SEC1):** `GGHSTATS_TRUSTED_PROXIES` — trust `X-Forwarded-For` / `X-Real-IP` only when the TCP peer is in the configured CIDR list; empty (default) ignores those headers. Startup warn when rate-limit or whitelist is active without trusted proxies.
- **HTTP server timeouts (SEC2):** `ReadHeaderTimeout` 10s, `ReadTimeout` 30s, `WriteTimeout` 60s, `IdleTimeout` 120s.

### Changed

- **Breaking:** unconditional trust of `X-Forwarded-For` / `X-Real-IP` removed. Deployments behind a reverse proxy must set `GGHSTATS_TRUSTED_PROXIES` (see README).
```

- [ ] **Step 3: Commit** (after user approves)

```text
Document trusted proxies problem, approach, and apply examples
```

---

### Task 5: Version bump prep + verification

**Files:**
- Modify: `VERSION` → `0.10.2`
- Modify: `README.md` version badge `0.10.1` → `0.10.2`
- Modify: `contrib/man/man1/gghstats.1` `.TH` line → July 2026 / `gghstats v0.10.2`
- Modify: `CHANGELOG.md` — move Unreleased SEC bullets into `## [0.10.2] - 2026-07-22` (use actual release date at tag time); fix compare links

```markdown
[Unreleased]: https://github.com/hrodrig/gghstats/compare/v0.10.2...HEAD
[0.10.2]: https://github.com/hrodrig/gghstats/compare/v0.10.1...v0.10.2
[0.10.1]: https://github.com/hrodrig/gghstats/compare/v0.10.0...v0.10.1
```

- [ ] **Step 1: Bump VERSION / badge / man `.TH` / CHANGELOG section**
- [ ] **Step 2: Run checks**

```bash
make lint
make test
# Before tag/release:
make release-check   # needs Docker for docker-scan
```

Expected: all green.

- [ ] **Step 3: Commit** (after user approves)

```text
Bump version to 0.10.2
```

- [ ] **Step 4: Stop — do not merge/tag/push until user explicitly requests release**

Release sequence (manual, user-driven): `develop` → `main`, annotated `v0.10.2`, push tag, then selfhosted pin follow-up.

---

## Spec coverage checklist

| Spec item | Task |
|-----------|------|
| Empty trusted → ignore XFF/XRI | 1 |
| Trusted peer → leftmost XFF / XRI / ParseIP fallback | 1 |
| `GGHSTATS_TRUSTED_PROXIES` parse CIDR/IP | 1–2 |
| Boot warn RL/whitelist + empty | 2 |
| Wire into serve Config | 2 |
| Update whitelist/ratelimit XFF tests | 2 |
| Server timeouts 10/30/60/120 | 3 |
| README problem/approach/A/B/C | 4 |
| env.example, man, SPEC, plans, CHANGELOG | 4 |
| VERSION 0.10.2 + badge + man TH | 5 |
| release-check before tag | 5 |

## Self-review notes

- No TBD placeholders.
- `clientIP` signature change is consistent across Tasks 1–2.
- Commits in tasks require **user message approval** before `git commit` (overrides “commit in step” automation).
