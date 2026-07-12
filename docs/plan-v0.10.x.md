# Plan — v0.10.x

**Band goal:** cheaper sync for star-heavy repos, and **opt-in** operator signals when traffic drops.

Parent: [ROADMAP.md](../ROADMAP.md) · Prior band: [plan-v0.9.x.md](plan-v0.9.x.md)

## In scope

| ID | Item | Notes |
|----|------|--------|
| SYNC | **Incremental star history** | Cursor / `last_seen_star_count` (or equivalent); avoid full O(n) stargazer pagination every sync. |
| A2 | **Opt-in alerts** | Threshold or WoW/MoM drop on clones/views; webhook and/or Slack/Loki-style sink; **off by default**. Pattern inspiration: **pgwd**. |
| PATH | **XDG / default path prep** | Document and soft-land toward `~/.config/gghstats/` (or platform equivalent); keep `GGHSTATS_DB` override. No hard break yet. Cover **`gghstats.env.example`**, **`contrib/launchd/`**, and **BSD port** paths — not only the binary default. Optional: `gghstats --print-defaults` (or equivalent) for inspection. |
| QW | Remaining quick wins | Prefer concrete leftovers (post-0.9 audit filter): `getPaginatedCtx` cleanup (dead `slicePtr` / double marshal); access-log **Warn/Error** by status (status field already logged); docs that **demo = collector/telemetry off**; optional `:develop` GHCR tag / collector `describe` only if cheap. **Do not** bump `SetMaxOpenConns` without evidence; **do not** add redundant `(repo, date)` INDEX (already PRIMARY KEY). |
| SYNC+ | **UpdateDeltas efficiency** | Full-table LAG on referrers/paths each sync — consider incremental / less frequent with star-sync work (not a blind pool bump). |
| C? | **Optional thin leaderboard** | Only if A2/SYNC done early; reuse H2H scoring — not a full org BI product. |

## Out of scope (this band)

- Full webhook-driven architecture (→ 0.11.x or 1.1).
- GraphQL total rewrite.
- SPEC API freeze (→ 1.0).
- Multi-writer SQLite / Postgres.

## Exit criteria

1. Star history sync is incremental when star sync is enabled (measurable: no full re-page of all stargazers on every cycle for large repos).
2. Alerts documented, opt-in, covered by tests for threshold logic.
3. Default-path prep documented (README / launchd / env example); migration notes if behavior changes.

## Checklist

- [ ] Incremental star sync + tests
- [ ] Alert sink(s) + env flags + docs
- [ ] XDG / default path prep docs (env.example, launchd, BSD notes; soft behavior if any)
- [ ] QW leftovers: `getPaginatedCtx` cleanup and/or access-log level by status (as capacity allows)
- [ ] Demo/docs note: collector/telemetry off in demo (if not already obvious)
- [ ] CHANGELOG + SPEC updates if new routes/metrics
- [ ] `make test` / lint green

## Parked (do not promote without pain)

- Raising SQLite `MaxOpenConns` / `MaxIdleConns` without measured contention
- Extra SQL INDEX on `(repo, date)` where PRIMARY KEY already exists
- Spec-nits only (explicit HEAD handlers, Accept-negotiation doc spam)
