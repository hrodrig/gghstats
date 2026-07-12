# Plan — v0.10.x

**Band goal:** cheaper sync for star-heavy repos, and **opt-in** operator signals when traffic drops.

Parent: [ROADMAP.md](../ROADMAP.md) · Prior band: [plan-v0.9.x.md](plan-v0.9.x.md)

## In scope

| ID | Item | Notes |
|----|------|--------|
| SYNC | **Incremental star history** | Cursor / `last_seen_star_count` (or equivalent); avoid full O(n) stargazer pagination every sync. |
| A2 | **Opt-in alerts** | Threshold or WoW/MoM drop on clones/views; webhook and/or Slack/Loki-style sink; **off by default**. Pattern inspiration: **pgwd**. |
| PATH | **XDG / default path prep** | Document and soft-land toward `~/.config/gghstats/` (or platform equivalent); keep `GGHSTATS_DB` override. No hard break yet. |
| QW | Remaining quick wins | e.g. document `internal/collector`, `:develop` image, service worker only if low risk. |
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
- [ ] XDG / default path prep docs (and soft behavior if any)
- [ ] CHANGELOG + SPEC updates if new routes/metrics
- [ ] `make test` / lint green
