# Roadmap

Product direction for **gghstats** (application binary and image).  
Production Compose / Helm / observability manifests live in **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** — not here.

Current release: see **`VERSION`** and **[CHANGELOG.md](CHANGELOG.md)**.  
Contracts for HTTP API and sync: **[SPEC.md](SPEC.md)**.

Detailed band plans (scope, exit criteria, checklist):

| Band | Plan |
|------|------|
| **0.9.x** | [docs/plan-v0.9.x.md](docs/plan-v0.9.x.md) (closed) |
| **0.10.x** | [docs/plan-v0.10.x.md](docs/plan-v0.10.x.md) (closed) |
| **0.11.x** | [docs/plan-v0.11.x.md](docs/plan-v0.11.x.md) (closed) |
| **1.0.0** | [docs/plan-v1.0.0.md](docs/plan-v1.0.0.md) (closed) |
| **1.1.0** | [docs/plan-v1.1.0.md](docs/plan-v1.1.0.md) (closed) |
| **1.2.0 / 1.3.0** | No separate band plan file — scope in [CHANGELOG.md](CHANGELOG.md) |
| **1.4.0** | [docs/plan-v1.4.0.md](docs/plan-v1.4.0.md) (closed) |
| **1.5.0** | [docs/plan-v1.5.0.md](docs/plan-v1.5.0.md) (closed) |
| **1.6.x** | [docs/plan-v1.6.x.md](docs/plan-v1.6.x.md) (open) |

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
| **B** | **Webhooks + delta-oriented sync**; GraphQL where it cuts REST pagination | M–L | Less polling; large accounts hit REST rate limits. Prefer **2.0.0** (not 1.1.0). |
| **C** | **Multi-repo analytics** (leaderboards, org rollups) | M | Reuse H2H scoring; expose rankings / rollups. Deferred past **0.11**. |
| **D** | **API-only mode** + JSON dogfood for official UI reads | M | **0.11.x** primary. Same binary; HTML optional. External React/Svelte/etc. against documented `/api/v1`. Not an in-tree SPA. |
| **E** | **Repo pins CLI** + **Featured** showcase (editorial vitrine, not groups on `/`) | M | **v1.1.0**. FILTER stays; empty catalog = identical 1.0 UX. CLI is how you live on the VPS (add/rm/sync/backup) — dashboard shows, console stewards. Design: [2026-08-14-featured-and-repo-cli-design.md](docs/superpowers/specs/2026-08-14-featured-and-repo-cli-design.md). |

### Sync efficiency (feeds B)

| Item | Notes |
|------|--------|
| **Incremental star history** | Full stargazer re-fetch is **O(n)** pages per sync. Add a cursor / `last_seen_star_count` (or equivalent). **Shipped in 0.10 work:** skip when count unchanged; incremental pages on growth; full rebuild on drop (SPEC §4.7). |
| **UpdateDeltas / other sync cost** | **UpdateDeltas efficiency shipped in v0.11.0** — `sync.Run` uses date-scoped `UpdateDeltasSince(today)`; see [plan-v0.11.x.md](docs/plan-v0.11.x.md). **Default SQLite path (DEF) shipped in v1.0.0** — platform config dir via `os.UserConfigDir()`; see [plan-v1.0.0.md](docs/plan-v1.0.0.md). |

## Release bands (path to 1.x)

```
0.9.x  → insights + demo/backup + quick wins      → docs/plan-v0.9.x.md
0.10.x → stars incremental + alerts + XDG prep  → docs/plan-v0.10.x.md
0.11.x → API-only + dogfood JSON + CSP Report-Only → docs/plan-v0.11.x.md
1.0.0  → defaults + API freeze + packaging      → docs/plan-v1.0.0.md
1.0.x  → patches on 1.0.1 (1.0.2, 1.0.3, …)
1.1.0  → pins CLI ∪ FILTER + Featured page      → docs/plan-v1.1.0.md
1.1.x  → SemVer patches after 1.1.0 is tagged (1.1.1, …)
1.2.0  → Featured pagination/search/sort + compact numbers (CHANGELOG)
1.3.0  → index unique cloners + rank/share + JSONL export (CHANGELOG)
1.4.0  → uniques UX + Featured JSON + sitemap /featured → docs/plan-v1.4.0.md
1.5.0  → traffic freshness + report visibility (+ chart JSON / legend / report ls --json) → docs/plan-v1.5.0.md
1.6.0  → dashboard UX / Settings / locales (#46) → docs/plan-v1.6.x.md
1.6.x  → operator DX + docs debt + small UX (post-1.6.1) → docs/plan-v1.6.x.md
2.0.0  → Line B (webhooks / serious ROADMAP) — not Featured
```

| Band | Goal | Must land | Defer |
|------|------|-----------|--------|
| **0.9.x** | Raw data → insights; zero-friction try-out | Trends on repo page; backup **or** demo; README comparison; selected quick wins | Webhooks (B); heavy alerts; API-only |
| **0.10.x** | Cheaper sync; usable ops signals | Incremental stars; opt-in alerts (A2); XDG prep (docs/flag); leftover QW in plan; **SEC1–SEC2** in **v0.10.2** | Full GraphQL rewrite |
| **0.11.x** | Bring-your-own frontend (still named gghstats) | API-only mode; JSON dogfood (official UI reads); CORS/auth + contract test; **SEC3** CSP phased | In-tree SPA; GitHub App; webhooks (**2.0.0**); leaderboard; HSTS/SSRF |
| **1.0.0** | Safe to depend | Sensible default DB path; SPEC freeze (incl. API-only); packaging parity; `release-check`; Line A done | Large new features |
| **1.1.0** | Catalog without breaking 1.0 | `repo` pins ∪ FILTER; Featured HTML + CLI; nav hidden if empty | Groups on `/`; Featured JSON; Line B |

**Risk rule:** do **not** block 1.0 on Line B. Prefer 1.0 = A + incremental stars + defaults + **API-only if 0.11 landed**; finish B in **2.0.0** (serious ROADMAP). Line E is **1.1.0** (additive, opt-in).

## Versioning (SemVer)

Current release: see **`VERSION`**. Patch = third digit of the **current** minor.

| Form | Examples | Meaning |
|------|----------|---------|
| Patch | `1.0.2`, `1.0.3`, `1.0.4`, … (`1.0.x`) | Small corrections on 1.0.1. No features. |
| Minor | `1.1.0`, then `1.2.0`, … | Add or remove **without** breaking the 1.x contract |
| Major | `2.0.0`, … | Serious product moment and/or ROADMAP-expected (e.g. Line B). Not required to break HTTP. |

Once **1.1.0** is tagged, patches of that line are `1.1.1`, `1.1.2` (`1.1.x`). Do not use `1.1.x` for fixes while HEAD is still 1.0.1.

## Next

Band plan **[docs/plan-v1.6.x.md](docs/plan-v1.6.x.md)** is **open** (post-**1.6.1**). Current tagged release: **`VERSION`** / [CHANGELOG.md](CHANGELOG.md).

**Shipped after 1.1.0:** **1.2.0** (compact numbers, Featured pagination/search/sort), **1.3.0** (index unique-cloners visibility, rank/share column, JSONL export), **1.4.0** (uniques UX, Featured JSON, sitemap `/featured`, Carlok #22–#25), **1.5.0** (traffic freshness, report visibility fail-closed, chart JSON / legend / `repo report ls --json`), **1.5.1** (Featured catalog not report-scoped, #47), **1.6.0** (responsive dashboard UX, gated Settings, bg/ru locales, soft theme starter, #46/#56), **1.6.1** (Featured Midnight/Dark chip contrast + `docs/themes.md`, #60/#61).

**1.6.x next:** operator DX (`config check` / `doctor` → often **1.7.0**), docs auth note, dogfood `(1d)` recheck, small UX (H2H prefill, shortcuts), pre-release drift checker — see the band plan. Line B (webhooks / delta sync) waits for **2.0.0**. Keep **plan ↔ ROADMAP ↔ SPEC ↔ CHANGELOG** in lockstep.

## Explicit non-goals (this repo)

- Multi-instance writers on one SQLite file.
- Replacing SQLite with PostgreSQL/MySQL as the default store.
- Converting the UI to a React/SPA (or similar) **in this repo** (external frontends via API-only are encouraged).
- GitHub App / OAuth flows (PAT-only), unless a later major rethink.
- Shipping production Traefik / Helm / full observability stacks (use **gghstats-selfhosted**).
- A large public REST surface beyond **dogfood** of the official UI / documented SPEC.

## How to propose work

Open an issue or PR against **`develop`**. Large ideas: describe the problem and fit to principles / band plans before coding. Line E shipped in **1.1.0**; further new lines need a band plan (or CHANGELOG + ROADMAP **Next** for small minors).
