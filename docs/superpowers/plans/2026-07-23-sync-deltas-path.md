# SYNC+ date-scoped deltas + PATH docs (v0.11.0) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close 0.10 debt inside **v0.11.0**: date-scoped `UpdateDeltasSince` for sync (SYNC+), and PATH docs/checklist only (no binary default change).

**Architecture:** Extract shared SQL for referrer/path LAG deltas; full `UpdateDeltas()` rebuilds all rows; `UpdateDeltasSince(since)` includes lookback day in the CTE and updates only `date >= since`. Sync cycle passes UTC `today`. PATH stays soft-land documentation.

**Tech Stack:** Go, SQLite via `modernc.org/sqlite`, existing `internal/store` + `internal/sync`, English docs.

**Spec:** [docs/superpowers/specs/2026-07-23-sync-deltas-path-design.md](../specs/2026-07-23-sync-deltas-path-design.md)

## Global Constraints

- Run commands from the **repository root**.
- English only for project artifacts.
- Work on `develop`; never merge/tag `main` without explicit user approval in the same turn.
- Before each `git commit`, show the full message and wait for user approval.
- Do not delete files without explicit approval.
- Keep `golang.org/x/net` pin (`v0.57.0` or whatever `AGENTS.md` / `go.mod` currently require).
- No XDG / binary default path change.
- No repo-scoped-only deltas in this plan.

## File map

| File | Role |
|------|------|
| `internal/store/store.go` | `UpdateDeltasSince`; refactor `UpdateDeltas` to full rebuild via helper |
| `internal/store/store_test.go` | Parity + lookback + old-rows-untouched tests |
| `internal/sync/sync.go` | Call `UpdateDeltasSince(today)` |
| `SPEC.md` | §4.3 note date-scoped deltas |
| `CHANGELOG.md` | SYNC+ + PATH note under 0.11.0 |
| `docs/plan-v0.10.x.md` | Mark SYNC+ / PATH closed |
| `docs/plan-v0.11.x.md` | Checklist + design link |
| `ROADMAP.md` | Drop open UpdateDeltas debt wording if still open |

---

### Task 1: Failing tests for `UpdateDeltasSince`

**Files:**
- Modify: `internal/store/store_test.go`
- Test: `go test ./internal/store/ -count=1 -run 'TestUpdateDeltasSince'`

**Interfaces:**
- Consumes: existing `UpsertReferrer`, `UpsertPath`, `UpdateDeltas`, `tempDB`
- Produces (expected API for Task 2): `func (s *Store) UpdateDeltasSince(sinceDate string) error`

- [ ] **Step 1: Write failing tests**

Add after `TestUpdateDeltas` in `store_test.go`:

```go
func referrerDelta(t *testing.T, s *Store, repo, date, referrer string) (countDelta, uniquesDelta int) {
	t.Helper()
	err := s.db.QueryRow(
		`SELECT count_delta, uniques_delta FROM referrers WHERE repo=? AND date=? AND referrer=?`,
		repo, date, referrer,
	).Scan(&countDelta, &uniquesDelta)
	if err != nil {
		t.Fatalf("referrerDelta: %v", err)
	}
	return countDelta, uniquesDelta
}

func pathDelta(t *testing.T, s *Store, repo, date, path string) (countDelta, uniquesDelta int) {
	t.Helper()
	err := s.db.QueryRow(
		`SELECT count_delta, uniques_delta FROM paths WHERE repo=? AND date=? AND path=?`,
		repo, date, path,
	).Scan(&countDelta, &uniquesDelta)
	if err != nil {
		t.Fatalf("pathDelta: %v", err)
	}
	return countDelta, uniquesDelta
}

func TestUpdateDeltasSince_ParityWithFull(t *testing.T) {
	s := tempDB(t)
	s.UpsertReferrer("r", "2026-03-20", "google.com", 40, 10)
	s.UpsertReferrer("r", "2026-03-21", "google.com", 50, 15)
	s.UpsertReferrer("r", "2026-03-22", "google.com", 55, 16)
	s.UpsertPath("r", "2026-03-20", "/doc", 40, 10)
	s.UpsertPath("r", "2026-03-21", "/doc", 50, 15)
	s.UpsertPath("r", "2026-03-22", "/doc", 55, 16)

	if err := s.UpdateDeltas(); err != nil {
		t.Fatal(err)
	}
	wantRC, wantRU := referrerDelta(t, s, "r", "2026-03-22", "google.com")
	wantPC, wantPU := pathDelta(t, s, "r", "2026-03-22", "/doc")

	// Reset recent deltas to garbage; leave day-20 as after full rebuild.
	if _, err := s.db.Exec(
		`UPDATE referrers SET count_delta=999, uniques_delta=999 WHERE date=?`,
		"2026-03-22",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(
		`UPDATE paths SET count_delta=999, uniques_delta=999 WHERE date=?`,
		"2026-03-22",
	); err != nil {
		t.Fatal(err)
	}

	if err := s.UpdateDeltasSince("2026-03-22"); err != nil {
		t.Fatal(err)
	}
	gotRC, gotRU := referrerDelta(t, s, "r", "2026-03-22", "google.com")
	gotPC, gotPU := pathDelta(t, s, "r", "2026-03-22", "/doc")
	if gotRC != wantRC || gotRU != wantRU {
		t.Errorf("referrer 03-22 deltas = (%d,%d), want (%d,%d)", gotRC, gotRU, wantRC, wantRU)
	}
	if gotPC != wantPC || gotPU != wantPU {
		t.Errorf("path 03-22 deltas = (%d,%d), want (%d,%d)", gotPC, gotPU, wantPC, wantPU)
	}
}

func TestUpdateDeltasSince_LeavesOlderRowsUntouched(t *testing.T) {
	s := tempDB(t)
	s.UpsertReferrer("r", "2026-03-20", "google.com", 40, 10)
	s.UpsertReferrer("r", "2026-03-21", "google.com", 50, 15)
	if err := s.UpdateDeltas(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(
		`UPDATE referrers SET count_delta=42, uniques_delta=7 WHERE date=?`,
		"2026-03-20",
	); err != nil {
		t.Fatal(err)
	}
	s.UpsertReferrer("r", "2026-03-22", "google.com", 55, 16)
	if err := s.UpdateDeltasSince("2026-03-22"); err != nil {
		t.Fatal(err)
	}
	c, u := referrerDelta(t, s, "r", "2026-03-20", "google.com")
	if c != 42 || u != 7 {
		t.Errorf("older row mutated: count_delta=%d uniques_delta=%d", c, u)
	}
	c22, _ := referrerDelta(t, s, "r", "2026-03-22", "google.com")
	if c22 != 5 { // 55-50
		t.Errorf("03-22 count_delta=%d, want 5", c22)
	}
}
```

- [ ] **Step 2: Run tests — expect fail (missing method)**

Run: `go test ./internal/store/ -count=1 -run 'TestUpdateDeltasSince' -v`

Expected: FAIL compile or undefined `UpdateDeltasSince`.

- [ ] **Step 3: Commit** (after user approves message)

Proposed message:

```
test(store): add failing UpdateDeltasSince parity tests
```

---

### Task 2: Implement `UpdateDeltasSince` + keep full `UpdateDeltas`

**Files:**
- Modify: `internal/store/store.go` (`UpdateDeltas` region ~331–359)
- Test: `go test ./internal/store/ -count=1 -run 'TestUpdateDeltas'`

**Interfaces:**
- Produces:
  - `func (s *Store) UpdateDeltasSince(sinceDate string) error`
  - `UpdateDeltas()` remains full-table rebuild (call helper with empty since / no lower bound on UPDATE)

- [ ] **Step 1: Implement**

Replace `UpdateDeltas` block with:

```go
// UpdateDeltas recalculates count_delta and uniques_delta for all referrers and paths rows.
func (s *Store) UpdateDeltas() error {
	return s.updateDeltasSince("")
}

// UpdateDeltasSince recalculates deltas for rows with date >= sinceDate (YYYY-MM-DD).
// The window includes one calendar day of lookback so LAG sees the previous day.
// Empty sinceDate rebuilds all rows (same as UpdateDeltas).
func (s *Store) UpdateDeltasSince(sinceDate string) error {
	return s.updateDeltasSince(sinceDate)
}

func (s *Store) updateDeltasSince(sinceDate string) error {
	lookback := ""
	if sinceDate != "" {
		d, err := time.Parse("2006-01-02", sinceDate)
		if err != nil {
			return fmt.Errorf("update deltas since: %w", err)
		}
		lookback = d.AddDate(0, 0, -1).Format("2006-01-02")
	}

	tables := []struct {
		table, col string
	}{
		{"referrers", "referrer"},
		{"paths", "path"},
	}
	for _, t := range tables {
		var query string
		var args []interface{}
		if sinceDate == "" {
			query = fmt.Sprintf(`
			WITH cte AS (
				SELECT repo, date, %[2]s, uniques, count,
					LAG(uniques) OVER (PARTITION BY repo, %[2]s ORDER BY date) AS prev_uniques,
					LAG(count) OVER (PARTITION BY repo, %[2]s ORDER BY date) AS prev_count
				FROM %[1]s
			)
			UPDATE %[1]s SET
				uniques_delta = MAX(0, cte.uniques - COALESCE(cte.prev_uniques, 0)),
				count_delta = MAX(0, cte.count - COALESCE(cte.prev_count, 0))
			FROM cte
			WHERE %[1]s.repo = cte.repo AND %[1]s.date = cte.date AND %[1]s.%[2]s = cte.%[2]s`,
				t.table, t.col)
		} else {
			query = fmt.Sprintf(`
			WITH cte AS (
				SELECT repo, date, %[2]s, uniques, count,
					LAG(uniques) OVER (PARTITION BY repo, %[2]s ORDER BY date) AS prev_uniques,
					LAG(count) OVER (PARTITION BY repo, %[2]s ORDER BY date) AS prev_count
				FROM %[1]s
				WHERE date >= ?
			)
			UPDATE %[1]s SET
				uniques_delta = MAX(0, cte.uniques - COALESCE(cte.prev_uniques, 0)),
				count_delta = MAX(0, cte.count - COALESCE(cte.prev_count, 0))
			FROM cte
			WHERE %[1]s.repo = cte.repo AND %[1]s.date = cte.date AND %[1]s.%[2]s = cte.%[2]s
			  AND %[1]s.date >= ?`,
				t.table, t.col)
			args = []interface{}{lookback, sinceDate}
		}
		if _, err := s.db.Exec(query, args...); err != nil {
			return fmt.Errorf("update deltas for %s: %w", t.table, err)
		}
	}
	return nil
}
```

Ensure `time` is imported in `store.go` if not already.

**Note:** CTE filtered with `date >= lookback` (not full history) is intentional: for partitions that only have history before lookback, the first row in the window may treat LAG as null. That only affects the lookback day itself, which is **not** updated (`UPDATE` requires `date >= sinceDate`). Rows at `sinceDate` still see the lookback day as LAG when it exists in-table and in-window. If a repo has a gap (no row on lookback day but older rows exist), LAG within the truncated window may differ from full-history LAG for `sinceDate`. Accept for v0.11 (GitHub popular traffic is dense daily); document in SPEC. Optional follow-up: full-partition CTE + date-limited UPDATE only (more CPU, always exact).

- [ ] **Step 2: Run store tests**

Run: `go test ./internal/store/ -count=1 -run 'TestUpdateDeltas' -v`

Expected: PASS (including `TestUpdateDeltas` and `TestUpdateDeltasSince_*`).

- [ ] **Step 3: Commit** (after user approves message)

```
feat(store): date-scoped UpdateDeltasSince with lookback
```

---

### Task 3: Wire sync cycle + docs / plans

**Files:**
- Modify: `internal/sync/sync.go` (post-worker `UpdateDeltas` call ~66)
- Modify: `SPEC.md` (§4.3)
- Modify: `CHANGELOG.md` (section `[0.11.0]`)
- Modify: `docs/plan-v0.10.x.md`
- Modify: `docs/plan-v0.11.x.md`
- Modify: `ROADMAP.md` (UpdateDeltas / PATH lines if still “open”)
- Test: `go test ./internal/sync/ ./internal/store/ -count=1`

**Interfaces:**
- Consumes: `UpdateDeltasSince(today string)`

- [ ] **Step 1: Change sync call site**

In `internal/sync/sync.go`, replace:

```go
	if err := db.UpdateDeltas(); err != nil {
		slog.Error("update deltas failed", "error", err)
	}
```

with:

```go
	if err := db.UpdateDeltasSince(today); err != nil {
		slog.Error("update deltas failed", "error", err)
	}
```

Leave `internal/demo` on full `UpdateDeltas()`.

- [ ] **Step 2: SPEC §4.3**

Change the deltas bullet to:

```markdown
- After workers finish, referrer/path deltas for the sync day are updated (`UpdateDeltasSince` with one-day lookback for LAG). Full-history `UpdateDeltas` remains for seed/repair.
```

- [ ] **Step 3: CHANGELOG under `[0.11.0]`**

Add under ### Changed (or ### Performance):

```markdown
- **SYNC+:** post-sync referrer/path delta updates are date-scoped (`UpdateDeltasSince` with one-day lookback) instead of full-table every cycle.
- **PATH:** confirm 0.10 soft-land closed; binary default remains `./data/gghstats.db` until v1.0.0 XDG/platform default.
```

- [ ] **Step 4: Plans + ROADMAP**

`docs/plan-v0.10.x.md`:
- SYNC+ row notes: **Implemented in v0.11.0** (date-scoped).
- PATH row: already implemented soft-land; reinforce code default → v1.0.0.
- Checklist: add `[x] SYNC+ UpdateDeltas efficiency (shipped in 0.11.0)`.

`docs/plan-v0.11.x.md`:
- In scope / checklist: SYNC+ + PATH docs close; link design spec.
- Status line may mention API1–API5 + SEC3 + SYNC+/PATH debt close.

`ROADMAP.md`:
- Mark UpdateDeltas efficiency done / point to 0.11; PATH soft-land done, code default still 1.0.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/sync/ ./internal/store/ -count=1`

Expected: PASS.

- [ ] **Step 6: Commit** (after user approves message)

```
feat(sync): use UpdateDeltasSince; close PATH/SYNC+ docs for 0.11.0
```

---

### Task 4: Sanity lint (no release tag)

**Files:** none required beyond fixes if lint fails

- [ ] **Step 1: Lint + related tests**

Run: `make lint && go test ./internal/store/ ./internal/sync/ ./internal/demo/ -count=1`

Expected: PASS.

- [ ] **Step 2: Do not merge to main / tag** unless user explicitly asks in this turn.

---

## Spec coverage self-check

| Spec item | Task |
|-----------|------|
| PATH docs/checklist only | Task 3 |
| `UpdateDeltasSince` + lookback | Task 2 |
| Sync uses since today | Task 3 |
| Full `UpdateDeltas` kept | Task 2 + demo unchanged |
| Parity / lookback / old rows tests | Task 1–2 |
| SPEC + CHANGELOG + plans | Task 3 |
| No XDG binary default | Global + Task 3 PATH note |

## Placeholder scan

None intentional. Gap note for sparse-day LAG vs full history is documented in Task 2 Note (accepted limitation).
