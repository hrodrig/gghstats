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
| **0.11.x** (optional) | [docs/plan-v0.11.x.md](docs/plan-v0.11.x.md) |
| **1.0.0** | [docs/plan-v1.0.0.md](docs/plan-v1.0.0.md) |

## Principles

- Single binary, single SQLite file, one writer process — **do not abandon this**.
- Keep the JSON API **small** (no generic CRUD).
- Prefer **high-leverage insights** that reuse data already in SQLite / `internal/h2h` over new infrastructure.
- Packaging and supply-chain quality stay first-class; product features must not weaken `make release-check`.
- Breaking changes only with a clear SemVer bump and CHANGELOG note.

## Priority lines (impact order)

| Line | What | Effort | Why |
|------|------|--------|-----|
| **A** | **Trending / velocity on repo page** + optional **alerts** (clone/view drop) | M | Momentum already in H2H (`Momentum7d` / `Momentum30d`); not on the repo page. Alerts can follow **pgwd**-style sinks without bloating the core. |
| **B** | **Webhooks + delta-oriented sync**; GraphQL where it cuts REST pagination | M–L | Less polling; large accounts hit REST rate limits. |
| **C** | **Multi-repo analytics** (leaderboards, org rollups) | M | Reuse H2H scoring; expose rankings / rollups. |

### Sync efficiency (feeds B)

| Item | Notes |
|------|--------|
| **Incremental star history** | Full stargazer re-fetch is **O(n)** pages per sync. Add a cursor / `last_seen_star_count` (or equivalent). |

## Release bands (path to 1.x)

```
0.9.x  → insights + demo/backup + quick wins     → docs/plan-v0.9.x.md
0.10.x → stars incremental + alerts + XDG prep → docs/plan-v0.10.x.md
0.11.x → webhooks/delta (optional; else 1.1)   → docs/plan-v0.11.x.md
1.0.0  → defaults + API freeze + packaging     → docs/plan-v1.0.0.md
1.x+   → B/C leftovers; non-goals stay intact
```

| Band | Goal | Must land | Defer |
|------|------|-----------|--------|
| **0.9.x** | Raw data → insights; zero-friction try-out | Trends on repo page; backup **or** demo; README comparison; selected quick wins | Webhooks (B); heavy alerts |
| **0.10.x** | Cheaper sync; usable ops signals | Incremental stars; opt-in alerts (A2); XDG prep (docs/flag) | Full GraphQL rewrite |
| **0.11.x** | Delta / webhooks without abandoning PAT | Webhook → delta sync; less aggressive polling | GitHub App/OAuth — skip band if blocked → post-1.0 |
| **1.0.0** | Safe to depend | Sensible default DB path; SPEC freeze; packaging parity; `release-check`; Line A done | Large new features |

**Risk rule:** do **not** block 1.0 on Line B. Prefer 1.0 = A + incremental stars + defaults; finish B in **0.11** or **1.1**.

## This week (if only one track)

1. **Trends on repo page + backup subcommand** — quality perception.  
2. **Demo mode** — zero barrier for evaluators.  
3. **README comparison table** — niche positioning.

Pick **one**; ship on `develop` before starting Line B. Detail: [plan-v0.9.x.md](docs/plan-v0.9.x.md).

## Explicit non-goals (this repo)

- Multi-instance writers on one SQLite file.
- Replacing SQLite with PostgreSQL/MySQL as the default store.
- Converting the UI to a React/SPA (or similar) stack.
- GitHub App / OAuth flows (PAT-only), unless a later major rethink.
- Shipping production Traefik / Helm / full observability stacks (use **gghstats-selfhosted**).
- A large public REST surface beyond the documented API.

## How to propose work

Open an issue or PR against **`develop`**. Large ideas: describe the problem and fit to principles / band plans before coding. Prefer extending Line A–C over new product lines.
