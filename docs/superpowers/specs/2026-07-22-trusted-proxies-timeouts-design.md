# Design: Trusted proxies + HTTP server timeouts (v0.10.2)

**Status:** Approved for implementation planning  
**Band:** Security slice before **0.11.x** API-only  
**Release:** **v0.10.2** (SEC only) → then **v0.11.0** (API1–API5, separate design/plan)  
**Related:** [docs/plan-v0.11.x.md](../../plan-v0.11.x.md) SEC1–SEC2 · [ROADMAP.md](../../../ROADMAP.md)

## Context

Current release is **v0.10.1**. Band **0.10.x** product work is closed. Post-0.10.1 review flagged:

| ID | Issue |
|----|--------|
| SEC1 | `clientIP` always trusts `X-Forwarded-For` / `X-Real-IP` → forged headers bypass rate-limit / whitelist when the binary is exposed directly |
| SEC2 | `http.Server` has no timeouts → Slowloris-class header/body stalls |

**Out of this release:** API-only (API1–API5), thin leaderboard, SEC3–SEC5 (CSP/HSTS/SSRF), webhook stretch.

## Goals

1. Close XFF forge for rate-limit and whitelist without requiring a reverse proxy.
2. Set sensible `http.Server` timeouts with no new knobs in 0.10.2.
3. Document **problem → approach → how to apply** with clear typical situations.
4. Ship as **v0.10.2** so prod can patch before the larger 0.11 API band.

## Non-goals

- Configurable timeout env vars (revisit only if operators hit pain).
- Full Forwarded / RFC 7239 parsing.
- Changing badge / healthz / metrics exemption lists.
- selfhosted Compose image pin (follow-up after tag; optional short README note if examples mention client IP).

---

## SEC1 — Trusted proxies

### Problem

Rate limiting and IP whitelist key off **client IP**. Today `clientIP` prefers `X-Forwarded-For` (leftmost) then `X-Real-IP`, else `RemoteAddr`.

If gghstats listens on the public internet (or any path where clients can set those headers), an attacker sends:

```http
X-Forwarded-For: 203.0.113.1
```

and appears as a different IP → dilute rate-limit buckets or spoof a whitelisted address.

Those headers are only trustworthy when they come from a **known reverse proxy** that overwrites or sanitizes them.

### Approach

New env: **`GGHSTATS_TRUSTED_PROXIES`** — comma-separated IPs and/or CIDRs.

**Rules for `clientIP(r)`:**

1. Resolve **peer** from `r.RemoteAddr` (`host` via `net.SplitHostPort`, else whole string).
2. If **`GGHSTATS_TRUSTED_PROXIES` is empty** → always return **peer**. Ignore `X-Forwarded-For` and `X-Real-IP`.
3. If peer is **not** contained in any trusted CIDR/IP → return **peer**. Ignore forwarded headers.
4. If peer **is** trusted:
   - Prefer `X-Forwarded-For`: take the **leftmost** non-empty hop (same as today when trusted).
   - Else prefer `X-Real-IP` (trimmed).
   - If the chosen value is empty or `net.ParseIP` fails, fall back to **peer**.

**Default empty = secure.** This is a **breaking** behavior change for deployments behind a proxy that relied on unconditional XFF trust: they must set `GGHSTATS_TRUSTED_PROXIES` to the proxy’s address or Docker/bridge CIDR.

**XFF hop model (explicit):** With a single (or chain of) trusted proxies that **set** client IP correctly, leftmost matches “original client.” Operators must configure the proxy to not pass through unsanitized client-supplied XFF as the sole source of truth. Multi-proxy “walk from the right” is **out of scope** for 0.10.2; document the single-proxy / sanitizing-proxy assumption.

### Boot warning

At serve startup, if **rate limiting is enabled** *or* **whitelist is non-empty**, and `GGHSTATS_TRUSTED_PROXIES` is empty, emit **`slog.Warn`** once with a clear message, e.g.:

> Client IP headers (X-Forwarded-For / X-Real-IP) are ignored because GGHSTATS_TRUSTED_PROXIES is empty. Behind a reverse proxy, set GGHSTATS_TRUSTED_PROXIES to the proxy IP or CIDR so rate-limit and whitelist see the real client.

No warn when both rate-limit off and whitelist empty (headers unused for those features).

### Wiring

- Parse env near existing rate-limit / whitelist parse (serve startup).
- Pass trusted-proxy set into server package (field on shared config, or package-level setter used only from `serve` — prefer **explicit config** on the type that owns middleware, not a global if avoidable).
- `clientIP` must use that set; unit tests inject the set without env when practical.

### Tests (must)

| Case | Expect |
|------|--------|
| Empty trusted; XFF set; peer = public | Return peer (ignore forge) |
| Trusted peer CIDR; XFF `10.5.5.5, 172.16.0.1` | Return `10.5.5.5` |
| Untrusted peer; XFF set | Return peer |
| Trusted peer; only X-Real-IP | Return that IP |
| Trusted peer; garbage XFF | Fall back to peer |
| Existing whitelist/ratelimit XFF tests | Update to pass trusted peer / CIDR |

---

## SEC2 — HTTP server timeouts

Set on the `http.Server` constructed in `cmd/gghstats/serve.go` (no new env in 0.10.2):

| Field | Value | Role |
|-------|-------|------|
| `ReadHeaderTimeout` | `10s` | Bound Slowloris on headers (**required**) |
| `ReadTimeout` | `30s` | Full request read |
| `WriteTimeout` | `60s` | Response write (dashboard + API) |
| `IdleTimeout` | `120s` | Keep-alive idle |

Document values in CHANGELOG + short README note. No unit test required beyond compile/smoke; optional assert in a small serve-config test if one already constructs `http.Server`.

---

## Documentation

### Structure (operator-facing)

Rewrite the README reverse-proxy / client-IP guidance (today: unconditional XFF). Use this shape:

1. **Problem** — forged `X-Forwarded-For` / `X-Real-IP` when the app is reachable without a trusted hop.
2. **Approach** — trust forwarded headers **only** when TCP peer ∈ `GGHSTATS_TRUSTED_PROXIES`; empty list = ignore headers; boot warn when RL/whitelist need real client IPs.
3. **How to apply** — typical situations:

| Situation | What to set | What rate-limit / whitelist see |
|-----------|-------------|----------------------------------|
| **A. Direct exposure** (host port or Docker `-p`, no reverse proxy) | Leave `GGHSTATS_TRUSTED_PROXIES` unset/empty | Peer `RemoteAddr` only; forged XFF ignored |
| **B. Behind Traefik / Caddy / nginx** (proxy on Docker network) | Proxy IP or bridge/pod CIDR, e.g. `172.16.0.0/12` or the proxy container IP | Real client from XFF/XRI once peer is trusted |
| **C. Proxy on host, app on localhost** | `127.0.0.1/32` and if needed `::1/128` | Real client; local proxy is the only trusted peer |

Each situation: short env snippet + one sentence outcome. Prefer concrete, copy-pasteable examples over abstract CIDR theory.

### Also update

| Artifact | Change |
|----------|--------|
| `contrib/gghstats.env.example` | Commented `GGHSTATS_TRUSTED_PROXIES` + one-line pointer to README situations |
| `contrib/man/man1/gghstats.1` | New `.TP` under ENVIRONMENT |
| `CHANGELOG.md` | **0.10.2** — Breaking: unconditional XFF removed; SEC1+SEC2 bullets |
| `docs/plan-v0.11.x.md` | Mark SEC1–SEC2 as shipping in **0.10.2** (not deferred vague) |
| `docs/plan-v0.10.x.md` | Optional one-line: patch **0.10.2** for SEC1–SEC2 |
| `VERSION` / README badge | `0.10.2` at release time |
| SPEC | Brief note under rate-limit / whitelist: client IP only from forwarded headers when peer ∈ trusted proxies (normative pointer) |

### Docs language

English only (project rule). Keep tone practical: problem → fix → copy-paste config.

---

## Release sequence

1. Implement SEC1 + SEC2 + docs on `develop`.
2. `make release-check` (lint, test, security, docker-scan).
3. Bump `VERSION` → **0.10.2**, CHANGELOG, man `.TH`, badge.
4. Merge `develop` → `main`, annotated tag `v0.10.2`, push tag.
5. Deploy prod / selfhosted pin follow-up.
6. **Then** start **0.11.0** API-only design/implementation (separate plan).

## Success criteria

- [ ] Empty trusted proxies → forged XFF cannot change rate-limit/whitelist identity in tests.
- [ ] Trusted peer → XFF/XRI used as today for leftmost / Real-IP.
- [ ] Boot warn when RL or whitelist active and proxies empty.
- [ ] `http.Server` timeouts set to the table above.
- [ ] README documents problem, approach, situations A/B/C with examples.
- [ ] env.example + man + CHANGELOG + plan notes updated.
- [ ] `make test` (and release-check before tag) green.

## Risks

| Risk | Mitigation |
|------|------------|
| Ops behind proxy forget new env → all clients share proxy IP bucket | Boot warn + README situations B/C + CHANGELOG Breaking |
| Over-broad CIDR (e.g. `0.0.0.0/0`) reopens forge | Document: only proxy/network CIDRs; never `0.0.0.0/0` |
| WriteTimeout too low for rare long responses | 60s chosen for dashboard; raise in later patch if reported |
