# Plan — v0.11.x

**Band goal:** ship an **API-only mode** so operators can run gghstats as JSON backend for their own UI (React, Svelte, …), exposing **what the official UI already needs** — not a generic CRUD platform. Name stays **gghstats**.

Parent: [ROADMAP.md](../ROADMAP.md) · Prior: [plan-v0.10.x.md](plan-v0.10.x.md) · Contracts: [SPEC.md](../SPEC.md)

## Product framing

| Do | Do not |
|----|--------|
| Keep the **gghstats** name and default HTML dashboard | Rename the project or drop the in-tree UI |
| `GGHSTATS_API_ONLY` / equivalent: serve JSON (+ health/metrics/badges as configured), skip HTML templates | Replace the UI with an in-repo React/Svelte SPA |
| Expand JSON only to **dogfood** official UI reads (index, repo detail, H2H, trends) | Large public REST / OpenAPI of every table |
| Document CORS + `x-api-token` for browser clients | Put the API token in public frontend bundles without a BFF/proxy warning |

## In scope (must)

| ID | Item | Notes |
|----|------|--------|
| API1 | **API-only mode** | Env/flag disables HTML routes; sync + store + JSON + optional `/metrics` / badges / healthz still work. |
| API2 | **Dogfood parity** | Endpoints (or fields) for data the official UI shows: repo list aggregates, traffic series, H2H inputs/scores, trends/momentum once A1 exists. Prefer additive `/api/v1/...`. |
| API3 | **CORS / auth docs** | Configurable CORS for API-only; clear SPEC + README: token via header; warn against embedding secrets in SPAs. Startup **warn** if API-only + overly open CORS (e.g. `Access-Control-Allow-Origin: *`) — token-leak footgun. |
| API4 | **SPEC update** | List new routes/fields; note API-only behavior (including whether `/sitemap.xml` / `/robots.txt` stay on or off). Feeds 1.0 freeze. |
| API5 | **Dogfood contract test** | Mandatory before exit: start with API-only and verify an external client can rebuild **index**, **repo page**, and **H2H** from documented endpoints alone (checklist in SPEC or test). |

## Stretch (same band if capacity)

| ID | Item | Notes |
|----|------|--------|
| B1–B4 | **Webhooks + delta sync** | Former primary of this band. Only if API1–API4 done early and scope stays tight. Else → **1.1+**. |
| C? | **Optional thin leaderboard** | Moved from 0.10.x. Rank tracked repos via H2H scoring — not full org BI (Line **C**). Only if API1–API5 done early. |

## Security hardening (from post-0.10.1 review)

Triage of external review (Kimi S1-*): **do not** reopen **v0.10.1**. The first two items ship as the **v0.10.2** security patch so **0.11.x** can stay focused on API-only work; CSP/HSTS remain later or at the edge.

| ID | Item | Priority | Notes |
|----|------|----------|--------|
| SEC1 | **Trusted proxies** | High | Shipped in **v0.10.2** via `GGHSTATS_TRUSTED_PROXIES` (CIDRs). Forwarded headers are now trusted only for configured proxy peers. |
| SEC2 | **`http.Server` timeouts** | High | Shipped in **v0.10.2** with fixed `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`. |
| SEC3 | **CSP (phased)** | Medium | Baseline headers already omit CSP on purpose (CDN + optional `HeadHTML`). Options: report-only first, or strict CSP when `HeadHTML` empty + document operator tradeoff. Chart.js / Bootstrap **JS** already use SRI; reverse-proxy strip of backend CSP is intentional for proxied UIs. |
| SEC4 | **HSTS** | Low (app) / edge | Prefer Traefik/Caddy in [gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted). App often HTTP behind TLS terminator — HSTS on the binary can be wrong. Optional `GGHSTATS_HSTS=…` only when TLS terminates in-process. |
| SEC5 | **Reverse-proxy SSRF guardrails** | Medium | Operator-configured URLs (trust boundary). Optional blocklist `localhost` / link-local / cloud metadata; document risk in SPEC/README. |
| SEC6 | **Follow-ups (other tracks)** | Low | Prepared statements audit (store); alert sink secrets hygiene; systemd hardening docs; keep govulncheck in CI — park unless pain. |

**Not inflated:** “Bootstrap CDN without SRI” is partly wrong — JS has `integrity=`; CSS is largely self-hosted under `/static/`.

## Out of scope

- In-tree SPA rewrite.
- GitHub App / OAuth.
- Postgres / multi-writer.
- Blocking **1.0.0** on webhooks.
- “Any imaginable frontend feature” beyond official UI data.
- Requiring full CSP + in-app HSTS for band exit (SEC3/SEC4 stretch or edge).

## Decision gate (webhooks only)

| If… | Then… |
|-----|--------|
| API-only + dogfood shipped; webhook design still small | Optional stretch in 0.11.x |
| Webhook scope balloons | Defer Line B to **1.1+**; still ship 0.11 on API-only |

## Exit criteria

1. Documented `GGHSTATS_API_ONLY` (or agreed name): HTML off, JSON on; smoke-tested.
2. External client can rebuild **core** dashboard views from documented endpoints alone (checklist in SPEC or README) — **dogfood contract test** green.
3. CORS + auth documented; tests for API-only routing (HTML → 404/disabled); startup warn when CORS is dangerously open with API-only.
4. CHANGELOG + SPEC updated (incl. sitemap/robots under API-only). Webhooks **not** required for band exit.
5. **Security (soft):** SEC1 and SEC2 shipped in **0.10.2**; do not block API-only on CSP/HSTS.

## Checklist

- [ ] API-only flag + server wiring + tests
- [ ] Dogfood gap list vs official UI → endpoints/fields added
- [ ] Dogfood contract test (index + repo + H2H via API only)
- [ ] CORS config (sensible defaults for API-only) + open-CORS warn
- [ ] SPEC + README (“Bring your own frontend”; sitemap/robots policy)
- [ ] CHANGELOG
- [ ] (Stretch) Webhooks / delta — or explicit defer note to 1.1
- [ ] (Stretch) Thin leaderboard (C?) — or keep parked under ROADMAP Line C
- [x] SEC1 Trusted proxies + tests (**0.10.2**)
- [x] SEC2 http.Server timeouts (**0.10.2**)
- [ ] (Stretch) SEC3 CSP phased / SEC4 HSTS edge-or-opt-in / SEC5 proxy SSRF guards
