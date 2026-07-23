# Design: SYNC+ date-scoped deltas + PATH docs close (v0.11.0)

**Date:** 2026-07-23  
**Status:** Approved for implementation planning  
**Release:** fold into **v0.11.0** (before tag; API-only band already on `develop`)  
**Parent:** [docs/plan-v0.11.x.md](../../plan-v0.11.x.md) · Debt from [docs/plan-v0.10.x.md](../../plan-v0.10.x.md)  
**Related:** [SPEC.md](../../../SPEC.md) §4 sync · `internal/store.UpdateDeltas` · `internal/sync.Run`

## Goal

Close two open **0.10.x** items inside **0.11.0**:

1. **SYNC+** — make post-sync referrer/path delta updates cheaper for the common daily sync pattern.
2. **PATH** — document that soft-land is done; binary default stays `./data/gghstats.db` until **v1.0.0** (no code change).

## Decisions

| ID | Choice |
|----|--------|
| Band | Ship inside **0.11.0** (not a separate 0.10.3) |
| PATH | **Docs/checklist only** — no XDG/default-path code; DEF stays [plan-v1.0.0.md](../../plan-v1.0.0.md) |
| SYNC+ | **Date-scoped** deltas with **≥1 day lookback** for LAG (not repo-scoped alone) |

## PATH (docs only)

### Current state

- Soft-land already shipped in 0.10: README Data directory table, env.example, systemd/launchd/BSD notes, man `GGHSTATS_DB`.
- Binary default remains `./data/gghstats.db` via `defaultDBPath()` / serve env fallback.
- Stable platform default (XDG / Application Support / etc.) is **v1.0.0 DEF**.

### Work

- Mark PATH implemented/closed in `docs/plan-v0.10.x.md` checklist if not already clear; note SYNC+ closing in 0.11.
- Align ROADMAP “UpdateDeltas / PATH” wording so PATH is not open debt for 0.10.
- One-line in CHANGELOG **0.11.0** under Changed/docs if needed: PATH soft-land confirmed; default unchanged until 1.0.
- **No** Go changes for paths.

## SYNC+ — date-scoped `UpdateDeltas`

### Problem

After every sync cycle, `sync.Run` calls `db.UpdateDeltas()`, which recomputes `count_delta` / `uniques_delta` for **all** rows in `referrers` and `paths` via full-table `LAG` windows. Cost grows with history even though a typical cycle only inserts/updates **today’s** popular traffic rows.

### Approach

Add a date-bounded API used by the sync cycle:

- `UpdateDeltasSince(sinceDate string)` — `sinceDate` is `YYYY-MM-DD` (UTC), normally the sync cycle’s `today`.
- SQL (both tables):
  - CTE / window includes rows with `date >= sinceDate - 1 day` (lookback) so `LAG` sees the previous day when present.
  - `UPDATE` applies only to rows with `date >= sinceDate`.
- Keep existing `UpdateDeltas()` as full rebuild (demo seed, tests, optional repair). Implementation may share one helper: full = since epoch / unbounded.

### Correctness

- For every row with `date >= sinceDate`, deltas must match what full `UpdateDeltas()` would produce.
- Boundary day `sinceDate`: lookback supplies prior day inside the window; if no prior row exists, `LAG` null → delta = cumulative (same as today).
- Do **not** truncate the CTE to `date >= sinceDate` without lookback (wrong border LAG).

### Call site

- `internal/sync.Run`: after workers, call `UpdateDeltasSince(today)` instead of full `UpdateDeltas()`.
- `internal/demo` seed: keep full `UpdateDeltas()` (historical rows written in one shot).

### Tests

- Multi-day fixture on `referrers` and `paths`.
- Assert `UpdateDeltasSince(today)` equals full `UpdateDeltas()` for rows `date >= today`.
- Assert older rows unchanged when only `Since` runs (optional: pre-set wrong deltas on old rows, confirm untouched).
- Lookback: two consecutive days; after writing day2 and `Since(day2)`, day2 deltas match full rebuild.

### Docs

- SPEC §4: note deltas update is **since sync day** (+ lookback), not full-table every cycle; full rebuild still available in store for seed/repair.
- CHANGELOG 0.11.0: SYNC+ performance note (behavior-compatible for current-day metrics).
- `plan-v0.10.x`: mark SYNC+ done (shipped in 0.11.0).
- `plan-v0.11.x`: add checklist items for SYNC+ + PATH docs close.

## Out of scope

- Changing binary default DB path / XDG resolution.
- Repo-scoped-only `UpdateDeltas(repos)` (optional later combine with date).
- SQLite pool `MaxOpenConns` bumps or extra indexes without measurement.
- Skipping `UpdateDeltas` when no referrer/path writes (nice-to-have; not required).

## Exit criteria

1. Sync cycle uses date-scoped deltas; tests prove parity with full rebuild for the since-window.
2. PATH remains soft-land only; plans/ROADMAP no longer list PATH as open 0.10 code debt.
3. SPEC + CHANGELOG updated; `make test` / lint green.
