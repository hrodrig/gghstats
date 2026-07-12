# Plan — v0.9.x

**Band goal:** turn raw traffic charts into **insights**, and let evaluators try the UI with **near-zero friction**.

Parent: [ROADMAP.md](../ROADMAP.md) · Contracts: [SPEC.md](../SPEC.md)

## In scope

| ID | Item | Notes |
|----|------|--------|
| A1 | **Trends / velocity on repo page** | Reuse `internal/h2h` (`Momentum7d` / `Momentum30d`); show on repository detail UI with short copy. |
| QW | **Backup CLI** | e.g. `gghstats backup` / restore via SQLite `VACUUM INTO` (or documented equivalent). |
| QW | **Demo mode** | Sample/fake dataset; no GitHub token required to browse UI. |
| QW | **README comparison table** | vs niche peers (self-hosted, history beyond 14d, single binary, Cosign/SBOM). |
| QW | **Selected quick wins** | Prefer: security headers baseline, SRI for Chart.js/Luxon, SQL index `(repo, date)` if missing, access-log status codes. Optional: `:develop` GHCR tag, document `internal/collector`. |

## Out of scope (this band)

- Line **B** (webhooks / GraphQL / delta sync).
- Line **D** (API-only / dogfood JSON) — **0.11.x**.
- Line **A2** heavy multi-channel alerts (belongs in 0.10.x).
- Default DB path freeze (1.0).
- Postgres, in-tree SPA, GitHub App.

## Exit criteria

Ship **at least**:

1. Trends visible on the repo page (A1), **and**
2. **Either** backup/restore CLI **or** demo mode (both preferred), **and**
3. README comparison table.

All changes: tests + CHANGELOG + `make lint` / `make test` green on `develop`.

## This-week pick (choose one track)

1. Trends + backup  
2. Demo mode  
3. README comparison only  

Do not start Line B until one track has landed.

## Checklist

- [x] A1 trends on repo page
- [x] Backup and/or restore subcommand
- [ ] Demo mode
- [ ] README comparison table
- [ ] Quick wins batch (headers / SRI / index / log status — as needed)
- [x] CHANGELOG `[Unreleased]` updated
- [ ] SPEC touched only if API/JSON fields change
