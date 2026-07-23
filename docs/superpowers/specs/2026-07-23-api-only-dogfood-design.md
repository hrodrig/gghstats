# Design: API-only mode + dogfood JSON (v0.11.0)

**Date:** 2026-07-23  
**Status:** Approved (band plan A / stretch SEC3 / endpoints X)  
**Parent:** [docs/plan-v0.11.x.md](../../plan-v0.11.x.md)

## Goal

Ship **gghstats v0.11.0** so operators can run the same binary as a JSON backend for external UIs (React, Svelte, …), covering **what the official HTML UI already shows** — not a generic CRUD API.

## Decisions

| ID | Choice |
|----|--------|
| Band shape | Full exit API1–API5 in **one** `0.11.0` release |
| Stretch | **SEC3 CSP** only; defer B webhooks, C leaderboard, SEC4 HSTS, SEC5 SSRF |
| JSON shape | Additive **`/api/v1/...`** routes (not fat `?include=` or `/api/v1/ui/{view}`) |
| Impl style | Spec-first + small shared loaders; no full HTML refactor |

## API1 — API-only

- Env: `GGHSTATS_API_ONLY` (truthy via existing `envBool`).
- When true: do not mount HTML routes; do not mount SEO (`/robots.txt`, `/sitemap.xml`).
- Keep: JSON API, healthz, badges (per `BadgePublic`), metrics (per config), reverse-proxy rules, static assets needed for badges/theme if still referenced (favicons/static may remain; HTML templates unused).
- Sync coordinator unchanged.

## API2 — Dogfood routes

Auth: existing `GGHSTATS_API_TOKEN` + `x-api-token` (404 if token unset).

| Method path | Role |
|-------------|------|
| `GET /api/repos` | Extend with `sort`, `dir`, `q`, `page` (defaults = current behavior) |
| `GET /api/v1/repos/{owner}/{repo}` | Repo summary + momentum 7d/30d |
| `GET /api/v1/repos/{owner}/{repo}/traffic` | Existing |
| `GET /api/v1/repos/{owner}/{repo}/stars` | Star history series |
| `GET /api/v1/repos/{owner}/{repo}/popular` | Referrers + paths (~14d) |
| `GET /api/v1/h2h?a=&b=&w=` | H2H scores + chart series |
| `GET /api/v1/charts/index-clones` | Aggregated index clones chart (lean list) |

## API3 — CORS

- `GGHSTATS_CORS_ORIGINS`: comma-separated allow list. Empty → `*` on authenticated API success (compat).
- Startup warn when `API_ONLY` and effective allow is `*`.

## API4 — Docs

SPEC, README, env.example, man: API-only, routes, CORS, CSP, SEO-off under API-only.

## API5 — Contract test

With API-only + token + seeded store: HTTP client rebuilds index, repo page, H2H from documented endpoints alone.

## SEC3 — CSP

- Default: `Content-Security-Policy-Report-Only` baseline (self + known CDN SRI scripts).
- `GGHSTATS_CSP=enforce`: only when `HeadHTML` empty; else warn and stay report-only.

## Out of scope

Webhooks/delta sync, thin leaderboard, in-app HSTS, reverse-proxy SSRF blocklist, in-tree SPA.
