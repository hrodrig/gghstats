# Roadmap

Product direction for **gghstats** (application binary and image).  
Production Compose / Helm / observability manifests live in **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** — not here.

Current release: see **`VERSION`** and **[CHANGELOG.md](CHANGELOG.md)**.  
Contracts for HTTP API and sync: **[SPEC.md](SPEC.md)**.

Detailed band plans (scope, exit criteria, checklist):

| Band | Plan |
|------|------|
| **0.9.x** | [docs/plan-v0.9.x.md](docs/plan-v0.9.x.md) |
| **0.10.x** | [docs/plan-v0.10.x.md](docs/plan-v0.10.x.md) |
| **0.11.x** | [docs/plan-v0.11.x.md](docs/plan-v0.11.x.md) |
| **1.0.0** | [docs/plan-v1.0.0.md](docs/plan-v1.0.0.md) |

## Principles

- Single binary, single SQLite file, one writer process — **do not abandon this**.
- Keep the JSON API **small** (no generic CRUD) — expand only to **dogfood** the official UI; optional **API-only** mode for external frontends.
- Prefer **high-leverage insights** that reuse data already in SQLite / `internal/h2h` over new infrastructure.
- Packaging and supply-chain quality stay first-class; product features must not weaken `make release-check`.
- Breaking changes only with a clear SemVer bump and CHANGELOG note.
- Project name stays **gghstats**; API-only is a **mode**, not a fork or rename.

## Priority lines (impact order)

| Line | What | Effort | Why |
|------|------|--------|-----|
| **A** | **Trending / velocity on repo page** + optional **alerts** (clone/view drop + ops + star milestones + SMTP) | M | Momentum **0.9**; alerts **0.10** (Slack/webhook/Loki) + milestones/SMTP in **v0.10.1** ([SPEC §8](SPEC.md)). Thin leaderboard → later / Line C. |
| **B** | **Webhooks + delta-oriented sync**; GraphQL where it cuts REST pagination | M–L | Less polling; large accounts hit REST rate limits. Prefer **1.1+** (not 0.11). |
| **C** | **Multi-repo analytics** (leaderboards, org rollups) | M | Reuse H2H scoring; expose rankings / rollups. Deferred past **0.11**. |
| **D** | **API-only mode** + JSON dogfood for official UI reads | M | **0.11.x** primary. Same binary; HTML optional. External React/Svelte/etc. against documented `/api/v1`. Not an in-tree SPA. |

### Sync efficiency (feeds B)

| Item | Notes |
|------|--------|
| **Incremental star history** | Full stargazer re-fetch is **O(n)** pages per sync. Add a cursor / `last_seen_star_count` (or equivalent). **Shipped in 0.10 work:** skip when count unchanged; incremental pages on growth; full rebuild on drop (SPEC §4.7). |
| **UpdateDeltas / other sync cost** | **UpdateDeltas efficiency shipped in v0.11.0** — `sync.Run` uses date-scoped `UpdateDeltasSince(today)`; see [plan-v0.11.x.md](docs/plan-v0.11.x.md). PATH soft-land is docs-only; binary default still `./data/gghstats.db` until v1.0.0. |

## Release bands (path to 1.x)

```
0.9.x  → insights + demo/backup + quick wins      → docs/plan-v0.9.x.md
0.10.x → stars incremental + alerts + XDG prep  → docs/plan-v0.10.x.md
0.11.x → API-only + dogfood JSON + CSP Report-Only → docs/plan-v0.11.x.md
1.0.0  → defaults + API freeze + packaging      → docs/plan-v1.0.0.md
1.x+   → Line B/C leftovers; non-goals intact
```

| Band | Goal | Must land | Defer |
|------|------|-----------|--------|
| **0.9.x** | Raw data → insights; zero-friction try-out | Trends on repo page; backup **or** demo; README comparison; selected quick wins | Webhooks (B); heavy alerts; API-only |
| **0.10.x** | Cheaper sync; usable ops signals | Incremental stars; opt-in alerts (A2); XDG prep (docs/flag); leftover QW in plan; **SEC1–SEC2** in **v0.10.2** | Full GraphQL rewrite |
| **0.11.x** | Bring-your-own frontend (still named gghstats) | API-only mode; JSON dogfood (official UI reads); CORS/auth + contract test; **SEC3** CSP phased | In-tree SPA; GitHub App; webhooks (**1.1+**); leaderboard; HSTS/SSRF |
| **1.0.0** | Safe to depend | Sensible default DB path; SPEC freeze (incl. API-only); packaging parity; `release-check`; Line A done | Large new features |

**Risk rule:** do **not** block 1.0 on Line B. Prefer 1.0 = A + incremental stars + defaults + **API-only if 0.11 landed**; finish B in **1.1+**.

## Next (after 0.10)

**0.10.x closed** (core **v0.10.1**; security patch **v0.10.2** SEC1/SEC2). Active band: [plan-v0.11.x.md](docs/plan-v0.11.x.md) — API-only + JSON dogfood + SEC3 CSP, plus SYNC+/PATH debt close.

## Explicit non-goals (this repo)

- Multi-instance writers on one SQLite file.
- Replacing SQLite with PostgreSQL/MySQL as the default store.
- Converting the UI to a React/SPA (or similar) **in this repo** (external frontends via API-only are encouraged).
- GitHub App / OAuth flows (PAT-only), unless a later major rethink.
- Shipping production Traefik / Helm / full observability stacks (use **gghstats-selfhosted**).
- A large public REST surface beyond **dogfood** of the official UI / documented SPEC.

## How to propose work

Open an issue or PR against **`develop`**. Large ideas: describe the problem and fit to principles / band plans before coding. Prefer extending Line A–C over new product lines.
