# Plan — v1.0.0

**Band goal:** **safe to depend** — stable defaults, frozen documented API, packaging parity. Not a feature dump.

Parent: [ROADMAP.md](../ROADMAP.md) · Spec: [SPEC.md](../SPEC.md)

## Prerequisites (from earlier bands)

| From | Requirement |
|------|-------------|
| **0.9.x** | Trends on repo page (Line A1) landed |
| **0.10.x** | Incremental star sync if star history remains enabled by default; alerts optional but documented if shipped |
| **0.11.x** | **Preferred:** API-only + dogfood JSON in SPEC. **Not required:** webhooks (Line B) |

## In scope

| ID | Item | Notes |
|----|------|--------|
| DEF | **Stable default SQLite path** | Sensible local/daemon default (e.g. under `~/.config/gghstats/` or platform equivalent); `GGHSTATS_DB` always wins. Migration / upgrade notes in README. |
| API | **SPEC freeze** | Documented routes and JSON fields (including API-only mode) stable for 1.x; **additive** fields OK; removals/renames = major (2.0). |
| PKG | **Packaging parity** | `.deb` / `.rpm` / Homebrew / GHCR tags; BSD ports + docs match `VERSION`. |
| REL | **Release bar** | `make release-check` green (lint, test, security, docker-scan). Release only from `main` per git-flow. |
| DOC | **Man / env / ports sync** | `contrib/man`, `gghstats.env.example`, FreeBSD/OpenBSD sync when `VERSION` bumps (`AGENTS.md`). |

## Out of scope (this release)

- Large new product lines (new Line beyond A–D).
- Blocking on webhooks/GraphQL (Line B).
- Postgres / in-tree SPA / multi-writer.
- Production Compose/Helm (stays in **gghstats-selfhosted**).

## Exit criteria

1. `VERSION` = `1.0.0`; CHANGELOG section complete; annotated tag `v1.0.0` on `main`.
2. Default DB behavior documented and tested (override + default).
3. SPEC marked as 1.0 stability baseline (HTML + API-only); README points to it.
4. `make release-check` passed; CI + Security workflows green.
5. Packages / ports version fields aligned.

## Checklist

- [ ] Prerequisites from 0.9 / 0.10 verified; 0.11 API-only preferred
- [ ] Default path implementation + tests + upgrade notes
- [ ] SPEC “1.0 stability” section or banner
- [ ] Man page + env example + port sync
- [ ] `make release-check`
- [ ] Merge `develop` → `main`, tag, GitHub release

## After 1.0

- **1.1+:** finish Line B if skipped; Line C rollups; selfhosted dashboards for new metrics.
- Non-goals remain unless a new major version rethinks them.
