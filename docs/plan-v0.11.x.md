# Plan — v0.11.x

**Status:** **Active** — target **v0.11.0** (API1–API5 + SEC3). SEC1/SEC2 shipped in **v0.10.2**.

**Band goal:** ship an **API-only mode** so operators can run gghstats as JSON backend for their own UI (React, Svelte, …), exposing **what the official UI already needs** — not a generic CRUD platform. Name stays **gghstats**.

Parent: [ROADMAP.md](../ROADMAP.md) · Prior: [plan-v0.10.x.md](plan-v0.10.x.md) · Contracts: [SPEC.md](../SPEC.md) · Design: [superpowers/specs/2026-07-23-api-only-dogfood-design.md](superpowers/specs/2026-07-23-api-only-dogfood-design.md)

## Product framing

| Do | Do not |
|----|--------|
| Keep the **gghstats** name and default HTML dashboard | Rename the project or drop the in-tree UI |
| `GGHSTATS_API_ONLY`: serve JSON (+ health/metrics/badges as configured), skip HTML + SEO | Replace the UI with an in-repo React/Svelte SPA |
| Expand JSON only to **dogfood** official UI reads (index, repo detail, H2H, trends) via **new `/api/v1/...` routes** | Large public REST / OpenAPI of every table; fat `?include=` bags |
| Document CORS + `x-api-token` for browser clients | Put the API token in public frontend bundles without a BFF/proxy warning |

## In scope (must for v0.11.0)

| ID | Item | Notes |
|----|------|--------|
| API1 | **API-only mode** | Env disables HTML + `/robots.txt` / `/sitemap.xml`; sync + store + JSON + optional `/metrics` / badges / healthz still work. |
| API2 | **Dogfood parity** | Additive routes: extend `/api/repos` (sort/dir/q/page); `GET /api/v1/repos/{o}/{r}`; stars; popular; h2h; `GET /api/v1/charts/index-clones`. Prefer thin handlers over HTML refactor. |
| API3 | **CORS / auth docs** | `GGHSTATS_CORS_ORIGINS`; empty = `*` (compat). Startup **warn** if API-only + `*`. |
| API4 | **SPEC update** | List new routes/fields; API-only SEO policy. Feeds 1.0 freeze. |
| API5 | **Dogfood contract test** | External client rebuilds **index**, **repo**, **H2H** from documented endpoints alone. |
| SEC3 | **CSP (phased)** | Default Report-Only; `GGHSTATS_CSP=enforce` only when HeadHTML empty. |

## Deferred (not this band)

| ID | Item | Notes |
|----|------|--------|
| B1–B4 | **Webhooks + delta sync** | → **1.1+** |
| C? | **Thin leaderboard** | → ROADMAP Line C / later |
| SEC4 | **HSTS** | Prefer edge (selfhosted Traefik/Caddy) |
| SEC5 | **Reverse-proxy SSRF guardrails** | Document risk; optional later |
| SEC1/SEC2 | Trusted proxies / HTTP timeouts | **Done in v0.10.2** |

## Out of scope

- In-tree SPA rewrite.
- GitHub App / OAuth.
- Postgres / multi-writer.
- Blocking **1.0.0** on webhooks.
- “Any imaginable frontend feature” beyond official UI data.

## Exit criteria

1. Documented `GGHSTATS_API_ONLY`: HTML/SEO off, JSON on; smoke-tested.
2. Dogfood contract test green (index + repo + H2H via API only).
3. CORS configurable; open-CORS warn with API-only; auth documented.
4. CHANGELOG + SPEC updated (incl. sitemap/robots under API-only).
5. CSP Report-Only shipped; enforce opt-in documented.
6. Webhooks / leaderboard / HSTS / SSRF **not** required.

## Checklist

- [x] API-only flag + server wiring + tests
- [x] Dogfood endpoints (repos query, repo detail, stars, popular, h2h, index-clones chart)
- [x] Dogfood contract test (index + repo + H2H via API only)
- [x] CORS config + open-CORS warn
- [x] SEC3 CSP Report-Only + opt-in enforce
- [x] SPEC + README + env.example + man + **docs/api.md** consumer guide
- [x] CHANGELOG + VERSION 0.11.0
- [x] SEC1 Trusted proxies (**0.10.2**)
- [x] SEC2 http.Server timeouts (**0.10.2**)
- [x] Explicit defer note: B / C / SEC4 / SEC5 → 1.1+ / ROADMAP
