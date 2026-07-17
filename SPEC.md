# Spec — HTTP API and sync

Normative operator contracts for **gghstats** as of **v0.9.0**.  
Narrative examples and env tables: **[README.md](README.md)**. Product direction: **[ROADMAP.md](ROADMAP.md)**.

This document describes **current** behavior. Changes that break clients must bump SemVer appropriately and update this file + CHANGELOG.

---

## 1. Deployment model

| Constraint | Rule |
|------------|------|
| Process | One `gghstats serve` (or CLI sync) per SQLite file |
| Storage | SQLite (`GGHSTATS_DB`); WAL; pragmatic `synchronous=NORMAL` |
| Writers | At most **one sync cycle** at a time (`sync.Coordinator`) |
| Auth to GitHub | Personal access token (`GGHSTATS_GITHUB_TOKEN`) only — no GitHub App / OAuth in-tree |
| Demo | `--demo` / `GGHSTATS_DEMO=true`: sample data, no token, sync/update-check off |
| Container | Distroless `static-debian13:nonroot`; default in-image DB `/data/gghstats.db` |

---

## 2. HTTP surface overview

| Route | Auth | Notes |
|-------|------|--------|
| `GET /api/v1/healthz` | Public | Liveness JSON |
| `GET /api/v1/badge/{owner}/{repo}` | Public by default | SVG; optional `GGHSTATS_BADGE_PUBLIC=false` |
| `GET /api/repos` | `x-api-token` | Disabled (**404**) if `GGHSTATS_API_TOKEN` unset |
| `GET /api/v1/repos/{owner}/{repo}/traffic` | `x-api-token` | Same gate as `/api/repos` |
| `GET` / `POST /api/v1/sync` | `x-api-token` | Same gate; requires sync coordinator |
| `GET /metrics` | Public by default | Off with `GGHSTATS_METRICS=false` |
| HTML UI (`/`, `/{owner}/{repo}`, `/h2h`, …) | Optional IP whitelist / rate limit | Not a JSON API |

**Always exempt** from IP rate limit and IP whitelist: `/metrics`, `/api/v1/healthz`, `/api/v1/badge/*`, and each `local` prefix from `GGHSTATS_REVERSE_PROXY_RULES`.

When `GGHSTATS_API_TOKEN` is set, a matching **`x-api-token`** header bypasses the IP whitelist on protected paths (token still validated by the API handler).

There is **no** generic REST CRUD layer.

**Security headers** on all HTTP responses: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy` (camera/mic/geolocation disabled).

---

## 3. API contracts

### 3.1 `GET /api/v1/healthz`

- **200** `application/json` → `{"status":"ok"}`

### 3.2 `GET /api/v1/badge/{owner}/{repo}`

- Response: **200** `image/svg+xml` with `Cache-Control: public, max-age=…` (default 300s via `GGHSTATS_BADGE_CACHE_SECONDS`).
- Query: `metric` ∈ `{clones, clones_30d, views, stars}` (default `clones`); `style` ∈ `{flat, flat-square}`; optional `label`.
- Semantics: lifetime sums in SQLite for `clones` / `views`; rolling 30d UTC for `clones_30d`; latest synced stars for `stars`.

### 3.3 `GET /api/repos`

- Requires `GGHSTATS_API_TOKEN` + header `x-api-token`. Wrong/missing → **401** `{"error":"unauthorized"}`. Token unset → **404**.
- CORS on success: `Access-Control-Allow-Origin: *`.
- Sort: always `total_views` descending (independent of HTML index `sort=`).
- Body: `total_count`, `total_stars`, `total_forks`, `total_views`, `total_clones`, `items[]` (`RepoSummary` JSON tags — see README).

### 3.4 `GET /api/v1/repos/{owner}/{repo}/traffic`

- Same auth gate as `/api/repos`.
- Query `days`: UTC rolling window (default **30**); **0** = all stored days; max **3660**.
- **200** JSON: `name`, `days`, `from`, `to`, `clones[]`, `views[]` (`date`, `count`, `uniques`). Missing days omitted (not zero-filled).

### 3.5 `POST /api/v1/sync` and `GET /api/v1/sync`

- Same auth gate as `/api/repos`.
- **POST** (no query): start full sync → **202** `{"status":"started","scope":"all"}`.
- **POST** `?repo=owner/name`: single-repo sync → **202** with `scope`/`repo`.
- Already running → **409** with `sync_in_progress` (maps from `sync.ErrInProgress`).
- **GET**: snapshot of `sync.Status` — `running`, `scope`, `repo`, `last_started_at`, `last_finished_at`, `last_error` (RFC3339 UTC when set).

---

## 4. Sync contracts

### 4.1 Serialization

- `Coordinator` allows **one** run (startup, scheduler tick, or manual API).
- Scheduler (`GGHSTATS_SYNC_INTERVAL`, default `1h`) **skips** the tick if a run is in progress.
- `GGHSTATS_SYNC_ON_STARTUP=false` skips the blocking startup sync.

### 4.2 Discovery and filter

- Empty explicit repo list → `ListRepos` then `GGHSTATS_FILTER` (e.g. `owner/*,!fork`).
- Explicit list / `?repo=` → sync only those names (metadata fetched per repo).

### 4.3 Concurrency inside a run

- Worker pool size: `--sync-workers` / `GGHSTATS_SYNC_WORKERS` (default **4**). Values `< 1` collapse to **1** (serial).
- Per-repo failures are logged and counted; they **do not** abort the whole cycle.
- After workers finish, deltas are updated (`UpdateDeltas`).

### 4.4 Per-repo steps (kinds for metrics)

Typical GitHub GETs per repo: metadata, open PRs, views, clones, referrers, paths; optional stargazer history when star sync is enabled.

Prometheus classifier `gghstats_sync_errors_total{kind}` uses kinds such as:  
`worker`, `repo_meta`, `open_prs`, `views`, `clones`, `referrers`, `paths`, `stargazers`.

Also: `gghstats_sync_repos_processed_total{status}` with `success` | `error`.

### 4.5 GitHub HTTP retries

- Default: **4** attempts, exponential backoff with full jitter (base 1s, cap 60s).
- Retries on: **429**, **403** when rate-limited, **5xx**, network errors.
- Honors `X-RateLimit-Reset` when a near-future reset is advertised.
- Non-retryable **4xx** (other than rate-limit 403) fail that call without further attempts.

### 4.6 Consistency during sync

- Each repo upserts independently; the UI/API may briefly show mixed old/new rows until the run completes.
- No snapshot transaction across the full repo list.
- SQLite still has **one writer** at a time; the pool parallelizes GitHub I/O and serializes DB writes through the connection pool.

---

## 5. CLI data ops (non-HTTP)

| Command | Contract |
|---------|----------|
| `gghstats fetch --repo OWNER/REPO` | Pull traffic (and related) from GitHub REST into SQLite for one repo; requires token |
| `gghstats report --repo OWNER/REPO` | Print a terminal traffic summary from SQLite (`--days`, default 14) |
| `gghstats export --repo OWNER/REPO` | Write traffic CSV to stdout or `--output` (`--days`, default 14) |
| `gghstats backup --output PATH` | Snapshot DB via SQLite `VACUUM INTO` |
| `gghstats restore --input PATH` | Replace target DB by file copy; stop `serve` if the DB is open |

Shared flags for fetch/report/export: `--repo` / `GGHSTATS_REPO`, `--token` / `GGHSTATS_GITHUB_TOKEN`, `--db` / `GGHSTATS_DB`.

---

## 6. Stability policy

Until **1.0.0**, the JSON API may gain additive fields without a major bump.  
Removing or renaming documented fields/routes is a **breaking** change (major after 1.0; clear CHANGELOG note before).

Prometheus metric names introduced in release notes are treated as operator-facing; renames require a CHANGELOG entry.

---

## 7. Out of scope

See **Non-goals** in [ROADMAP.md](ROADMAP.md). Multi-writer SQLite, GitHub Apps, and production cluster manifests are not part of this spec.
