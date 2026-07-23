# Spec — HTTP API and sync

Normative operator contracts for **gghstats** as of **v0.11.0**.
**Client how-to (examples, auth, dogfood map):** **[docs/api.md](docs/api.md)**.  
Narrative install/env: **[README.md](README.md)**. Product direction: **[ROADMAP.md](ROADMAP.md)**.

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
| API-only | `GGHSTATS_API_ONLY=true`: no HTML UI; no `/robots.txt` / `/sitemap.xml`; JSON + probes as configured |

---

## 2. HTTP surface overview

| Route | Auth | Notes |
|-------|------|--------|
| `GET /api/v1/healthz` | Public | Liveness JSON |
| `GET /api/v1/badge/{owner}/{repo}` | Public by default | SVG; optional `GGHSTATS_BADGE_PUBLIC=false` |
| `GET /api/repos` | `x-api-token` | List + KPIs; optional `sort`/`dir`/`q`/`page`/`per_page` |
| `GET /api/v1/repos/{owner}/{repo}` | `x-api-token` | Summary + momentum |
| `GET /api/v1/repos/{owner}/{repo}/traffic` | `x-api-token` | Clones/views series |
| `GET /api/v1/repos/{owner}/{repo}/stars` | `x-api-token` | Star history series |
| `GET /api/v1/repos/{owner}/{repo}/popular` | `x-api-token` | Referrers + paths (~14d) |
| `GET /api/v1/h2h` | `x-api-token` | Compare `a`/`b`/`w` + chart payload |
| `GET /api/v1/charts/index-clones` | `x-api-token` | Aggregated index clones chart |
| `GET` / `POST /api/v1/sync` | `x-api-token` | Sync coordinator |
| `GET /metrics` | Public by default | Off with `GGHSTATS_METRICS=false` |
| HTML UI (`/`, `/{owner}/{repo}`, `/h2h`, …) | Optional IP whitelist / rate limit | Omitted when `GGHSTATS_API_ONLY=true` |
| `/robots.txt`, `/sitemap.xml` | — | Omitted (404) when API-only |

**Always exempt** from IP rate limit and IP whitelist: `/metrics`, `/api/v1/healthz`, `/api/v1/badge/*`, and each `local` prefix from `GGHSTATS_REVERSE_PROXY_RULES`.

When `GGHSTATS_API_TOKEN` is set, a matching **`x-api-token`** header bypasses the IP whitelist on protected paths (token still validated by the API handler).

For rate limit and IP whitelist identity, `X-Forwarded-For` / `X-Real-IP`
are trusted only when the TCP peer is in `GGHSTATS_TRUSTED_PROXIES`.
Otherwise the peer `RemoteAddr` is authoritative.

There is **no** generic REST CRUD layer.

**Security headers** on all HTTP responses: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy` (camera/mic/geolocation disabled), plus **CSP** (default Report-Only; see §2.1).

**CORS** on authenticated JSON success responses: `GGHSTATS_CORS_ORIGINS` (comma list). Empty → `Access-Control-Allow-Origin: *`. When `GGHSTATS_API_ONLY` and CORS is open (`*`), serve logs a startup warn — do not embed the API token in a public SPA (use a BFF/proxy).

### 2.1 CSP (SEC3)

| Mode | Behavior |
|------|----------|
| Default / unset | `Content-Security-Policy-Report-Only` baseline (self + unpkg/jsDelivr Chart.js/Bootstrap + Google Fonts; `'unsafe-inline'` for dashboard scripts) |
| `GGHSTATS_CSP=enforce` | Sends enforcing `Content-Security-Policy` **only** when `GGHSTATS_HEAD_HTML` is empty; otherwise warn and stay Report-Only |

### 2.2 Dogfood contract (API5)

With API-only + token + seeded store, an HTTP client must rebuild **index**, **repo detail**, and **H2H** using only documented JSON routes (covered by `TestDogfoodContract_APIOnly`). Checklist: `/api/repos` (+ optional chart), `/api/v1/repos/{o}/{r}` + traffic + stars + popular, `/api/v1/h2h`.

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
- CORS per §2 (`GGHSTATS_CORS_ORIGINS` / `*`).
- Query: `sort`, `dir`, `q` (name substring). Defaults without sort: **`total_views` / `desc`** (pre-0.11 compat). Pagination when `page` and/or `per_page` present; otherwise all matching items.
- Body: `total_count`, KPI totals, `sort`, `dir`, `q`, `items[]`; when paginating also `page`, `per_page`, `total_pages`.

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

### 3.6 `GET /api/v1/repos/{owner}/{repo}`

- Same auth. **200**: `repo` (`RepoSummary`), `momentum_7d` / `momentum_30d` (float), `momentum_*_pct` (display strings). **404** if unknown.

### 3.7 `GET /api/v1/repos/{owner}/{repo}/stars`

- Same auth. **200**: `name`, `stars[]` (cumulative star history rows).

### 3.8 `GET /api/v1/repos/{owner}/{repo}/popular`

- Same auth. **200**: `name`, `days` (14), `referrers[]`, `paths[]`.

### 3.9 `GET /api/v1/h2h`

- Same auth. Query: `a`, `b` (required `owner/repo`), `w` interval (`7d` default, `30d`, `total`).
- **200**: `a`, `b`, `interval`, `result` (`h2h.Result` with snake_case JSON: `repo_a`, `score_a`, `rows[]`, `suggest`, …), optional `charts` (aligned series). Examples: [docs/api.md](docs/api.md).

### 3.10 `GET /api/v1/charts/index-clones`

- Same auth. Honors same `sort`/`dir`/`q` filter as `/api/repos` (no pagination). **200**: `count`, `series`, echo of filter fields.

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

### 4.7 Incremental star history (when star sync is enabled)

**Why:** `GET /repos/{owner}/{repo}/stargazers` with `Accept: application/vnd.github.v3.star+json` is **O(n / 100)** pages per sync. Re-fetching every stargazer on every cycle burns PAT quota on large repos without improving the dashboard. Incremental sync keeps history fresh while only paging **new** stars.

**Cursor (SQLite `repos`):**

| Column | Meaning |
|--------|---------|
| `last_seen_star_count` | Last successful star-history sync’s `stargazers_count` (`-1` = never synced) |
| `last_starred_at` | Newest `starred_at` observed (RFC3339); empty until first success |

**Algorithm (per repo, when `SyncStars` is on):**

1. Read metadata `stargazers_count` (already fetched).
2. If cursor synced and count **unchanged** → **skip** stargazer HTTP entirely.
3. If never synced or count **decreased** (unstars) → **full** pagination; sort by `starred_at` ascending; rewrite daily cumulative totals; update cursor.
4. If count **increased** by `D` → fetch newest pages until `D` new stars (or past `last_starred_at`); append cumulatives from `last_seen_star_count + 1`; update cursor.

GitHub returns stargazer pages **newest-first**; gghstats always sorts ascending before writing cumulative day totals.

**Operator signal:** logs include `stargazers skipped` (`count_unchanged`) or `stargazers synced` with `mode=full|incremental`.

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

### 6.1 Release quality bar

Before merging to **`main`** or publishing a tagged release, **`make release-check`** must pass. That gate includes (in order):

| Step | Requirement |
|------|-------------|
| **`make lint`** | `gofmt -s`, `go vet`, and the pinned `golang.org/x/net` check |
| **`make test`** | `go test -race ./...` |
| **`make cover`** | Statement coverage **≥ 80%** project-wide (`go tool cover -func`; fails below the floor) |
| **`make security`** | govulncheck, gocyclo, Grype directory scan |
| **`make docker-scan`** | Image build + Grype (`--fail-on high`); requires Docker |

**Hard rule:** do not tag or run `make release` if coverage is below **80%**. Raise coverage with tests (or shrink untested surface) before release — do not lower the floor without a SPEC + CHANGELOG note.

Local check without the full release suite: `make cover`.

---

## 7. Out of scope

See **Non-goals** in [ROADMAP.md](ROADMAP.md). Multi-writer SQLite, GitHub Apps, and production cluster manifests are not part of this spec.

---

## 8. Alerts (target behavior — 0.10.x)

**Status:** Sinks (**slack** / **webhook** / **loki** / **smtp**), **`gghstats alert test`**, **traffic**, **ops**, and **star growth milestones** (§8.3) after sync are implemented. Band checklist: [docs/plan-v0.10.x.md](docs/plan-v0.10.x.md).

Operator-facing copies (README, `contrib/gghstats.env.example`, man page) must stay aligned with this section when the feature ships.

### 8.1 Metrics and windows

Vocabulary operators must see so they know **what is being measured**, not only that “a threshold fired.”

### Base metrics (from SQLite / GitHub Traffic)

| Metric id | What it is | Source in gghstats |
|-------------------|------------|--------------------|
| **`clones`** | How many times the repo was **cloned** (git clone / GitHub Traffic → Clones). Distinct from unique cloners unless using uniques. | Daily rows in `clones` (`count` / `uniques`) |
| **`views`** | How many times repo content was **viewed** on GitHub (Traffic → Views). | Daily rows in `views` (`count` / `uniques`) |
| **`stars`** | Cumulative **stargazer** count (GitHub star button). | `repos.stars` and/or `stars` history totals |
| **`uniques` (optional)** | Distinct cloners or viewers in the window (GitHub unique series). Env/docs must say **count vs uniques** explicitly. | `clones.uniques` / `views.uniques` |

Comparison windows for **drops** (A2) must be named in config/docs:

| Window | Meaning |
|--------|---------|
| **7d / 30d sum** | Sum of daily `count` (or `uniques`) over the last N UTC days ending at last sync “today”. |
| **WoW** | Week-over-week: last 7d sum vs the previous 7d sum. |
| **MoM** | Month-over-month: last 30d vs previous 30d **or** calendar month — implementers pick one rule and document it in README/env. |

### 8.2 Rule kinds — drops and absolute thresholds (0.10.0)

**Drop** = traffic got **worse**. **Absolute high/floor (day or N-day)** = crossed a fixed bar (e.g. “today clones ≥ 225”). Both use the same sinks; both need plain-language docs.

| Concept | Operator meaning | Example |
|---------|------------------|---------|
| **Absolute high** | Window value is **at or above** a fixed number. | Today `hrodrig/pgwd` **clones ≥ 225** |
| **Absolute floor / zero** | Window value is **below** a bar, or **exactly 0** (no traffic). Missing day row → **0**. | Today `hrodrig/groot` **clones == 0** |
| **Relative drop %** | Current window is X% **below** the comparison window. | `clones` WoW drop ≥ **30%** |
| **Scope** | One `owner/name`, each synced repo, or **aggregate** all synced repos (sum). | Only `hrodrig/pgwd`, or **fleet total** |
| **Aggregate (fleet)** | Sum metric across **all repos in the DB** (document whether `GGHSTATS_FILTER` narrows). No single `repo` field. | All clones ever stored **≥ 30000** |

Alert payload / log line must include: **metric**, **window**, **before/after or current vs threshold**, **repo**, **configured rule**. Never opaque “alert fired.”

**Non-goals for A2 traffic rules:** undocumented sink channels, arbitrary custom SQL. Growth **milestones** (fire-once star ladders) are §8.3. **Ops / sync-health** rules are separate (§8.7) — same sinks, different trigger.

### 8.3 Growth milestones

A **growth milestone** means: an upward **goal** the operator set was **crossed**. Implemented for **`metric=stars`** (fire-once per threshold).

| Concept | Operator meaning | Example |
|---------|------------------|---------|
| **Absolute milestone** | Metric reaches or exceeds a fixed total. | `stars` ≥ **100**, then **500**, then **1000** |
| **Fire-once** | Each `(repo, metric, threshold)` notifies **at most once** until a documented reset. | Avoid re-spam on every sync after crossing |
| **Metric choice** | Prefer totals already in DB; each option named (`stars` cumulative vs `clones` 30d sum — never ambiguous). | Document which series each env var uses |

Milestone payload must include: **metric**, **threshold**, **current value**, **repo**.

### 8.4 Plain-language examples → config

Docs (README / env.example) must show **human sentence → what gghstats checks**. Env key names are part of the target contract; rename only with CHANGELOG + this section.

| What you mean | Rule type | Check (after sync) | Draft config sketch |
|---------------|-----------|--------------------|---------------------|
| “Alert me if **today** `hrodrig/pgwd` has **more than 225 clones**.” | Absolute **high** (day) | `clones.count` for UTC today on repo `hrodrig/pgwd` **≥ 225** | `repo=hrodrig/pgwd`, `metric=clones`, `window=1d`, `op=gte`, `value=225` |
| “Alert me if **today** `hrodrig/groot` has had **no clones**.” | Absolute **floor** / zero (day) | `clones.count` for UTC today on `hrodrig/groot` is **0** or **missing** (treat missing day row as 0) | `repo=hrodrig/groot`, `metric=clones`, `window=1d`, `op=eq`, `value=0` |
| “Alert me when **all clones** (every synced repo, all days) **exceed 30000**.” | Absolute **high** (fleet **lifetime**) | `SUM(clones.count)` over **all repos** and **all dates** in SQLite **≥ 30000** | `scope=all_repos`, `metric=clones`, `window=lifetime`, `op=gte`, `value=30000` |
| “Alert me if **today** `hrodrig/pgwd` has **fewer than 10 views**.” | Absolute **floor** (day) | `views.count` for UTC today **&lt; 10** | `repo=…`, `metric=views`, `window=1d`, `op=lt`, `value=10` |
| “Alert me if **this week’s clones** on `pgwd` fell **30%+** vs last week.” | Relative **drop** (WoW) | sum(`clones`, last 7d) ≤ sum(previous 7d) × (1 − 0.30) | `metric=clones`, `window=wow`, `op=drop_pct`, `value=30` |
| “Alert me if **7-day views** across synced repos drop **below 50**.” | Absolute **floor** (7d) | sum(`views`, 7d) **&lt; 50** (per repo or aggregate — document scope) | `metric=views`, `window=7d`, `op=lt`, `value=50`, `scope=each_repo` |
| “Alert me when `pgwd` reaches **100**, then **500** stars.” | Growth **milestone** (A2+) | `repos.stars` crosses 100 (once), later 500 (once) | `repo=hrodrig/pgwd`, `metric=stars`, `milestones=100,500`, `fire=once` |

**Worked example (high clones today):**

> Operator: *Alert me if today pgwd has more than 225 clones.*

Interpretation:

1. **Repo:** `hrodrig/pgwd` (full `owner/name` as stored after sync).
2. **Metric:** `clones` → GitHub Traffic **Clones** `count` for that calendar day (UTC), not uniques unless the rule says so.
3. **Window:** `1d` / “today” = the UTC date of the last successful sync’s “today” row (same day key used when writing `clones`).
4. **Operator:** greater-or-equal **225**.
5. **When evaluated:** after each sync that upserts that day’s clone row (not continuous push from GitHub).
6. **Notify once per day per rule** (recommended default for absolute highs) so hourly sync does not spam Slack every hour after 225 is already true — document debounce (`once_per_utc_day` vs every sync).

Pseudo-env (illustrative only) — **several rules in one JSON array**:

```bash
GGHSTATS_ALERTS_ENABLED=true
# Plain language:
#   - "Alert me if today hrodrig/pgwd has more than 225 clones."
#   - "Alert me if today hrodrig/groot has had no clones."
#   - "Alert me when all clones (every synced repo, all days) exceed 30000."
#   - "Alert me if today hrodrig/pgwd has fewer than 10 views."
#   - "Alert me if this week's clones on pgwd fell 30%+ vs last week."
#   - "Alert me when pgwd reaches 100, then 500 stars."   (A2+ milestone shape)
GGHSTATS_ALERT_RULES='[
  {"repo":"hrodrig/pgwd","metric":"clones","window":"1d","op":"gte","value":225,"debounce":"once_per_utc_day"},
  {"repo":"hrodrig/groot","metric":"clones","window":"1d","op":"eq","value":0,"debounce":"once_per_utc_day"},
  {"scope":"all_repos","metric":"clones","window":"lifetime","op":"gte","value":30000,"debounce":"once","fire":"once"},
  {"repo":"hrodrig/pgwd","metric":"views","window":"1d","op":"lt","value":10,"debounce":"once_per_utc_day"},
  {"repo":"hrodrig/pgwd","metric":"clones","window":"wow","op":"drop_pct","value":30,"debounce":"once_per_utc_day"},
  {"repo":"hrodrig/pgwd","metric":"stars","milestones":[100,500],"fire":"once"}
]'
# Where notifications go (sinks) — secrets via env refs, not literals in JSON:
# (set these in the process environment / .env — never commit real URLs)
GGHSTATS_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
GGHSTATS_DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/...
GGHSTATS_N8N_WEBHOOK_URL=https://n8n.example.com/webhook/gghstats
GGHSTATS_N8N_TOKEN=secret-token

GGHSTATS_ALERT_SINKS='[
  {"type":"slack","webhook_url_env":"GGHSTATS_SLACK_WEBHOOK_URL"},
  {"type":"webhook","url_env":"GGHSTATS_DISCORD_WEBHOOK_URL","body":"discord"},
  {"type":"webhook","url_env":"GGHSTATS_N8N_WEBHOOK_URL","headers_env":{"Authorization":"GGHSTATS_N8N_TOKEN"}}
]'
```

### 8.5 Sinks (delivery)

**Hard rule:** an alert that cannot be **delivered** is not a product feature. Do **not** ship rule evaluation, debounce tables, or “alerts enabled” UX until **at least one sink type** can send a real message in tests (httptest mock) and is documented for operators.

**Secrets policy (normative for A2):**

| Do | Don't |
|----|--------|
| Put webhook URLs / tokens in **environment variables** (or secret files loaded into the env) | Hardcode secrets inside `GGHSTATS_ALERT_SINKS` JSON checked into git |
| Reference them from sink JSON with `*_env` fields (`webhook_url_env`, `url_env`, `headers_env`) | Commit real Slack/Discord URLs in `contrib/*.example` |
| Allow optional inline URL **only** for local throwaway tests (document as insecure) | Log full webhook URLs at info level |

`gghstats.env.example` shows **placeholder** env names + empty values; operators fill real secrets outside the repo (same pattern as `GGHSTATS_GITHUB_TOKEN`).

**Implementation order (A2):**

1. **Sinks** — parse `GGHSTATS_ALERT_SINKS`; resolve `*_env` → `os.Getenv`; send a fixed test payload (`slack` / `webhook` / `loki` / `smtp`); fail closed if enabled but no valid sink.
2. **`gghstats alert test`** — smoke-test delivery to configured sinks **without** waiting for a real rule (§8.8). Required before shipping rule evaluation so operators can prove Slack/Loki wiring.
3. **Rules** — parse `GGHSTATS_ALERT_RULES`; evaluate after sync (traffic + ops); call the same sink layer.
4. **Docs** — glossary + plain-language examples + env.example.

**Rules** = *when* to fire (`GGHSTATS_ALERT_RULES`).  
**Sinks** = *where* the message is sent (`GGHSTATS_ALERT_SINKS`).  
**Default: fan-out** — when a rule fires, gghstats sends to **every** entry in `GGHSTATS_ALERT_SINKS` (Slack + Discord + n8n at once is fine). One sink failing should not block the others (log error per sink; document best-effort).  
Optional later: per-rule `sinks: ["slack", …]` to narrow; MVP = all sinks.

If `GGHSTATS_ALERTS_ENABLED=true` but sinks are empty/invalid → log error, **do not** pretend alerts work (no silent “evaluated but nowhere to send”).

| `type` | What it is | Config fields |
|----------------|------------|------------------------|
| **`slack`** | Slack Incoming Webhook | `webhook_url_env` (preferred) or `webhook_url` (dev-only) |
| **`webhook`** | Generic HTTP POST (JSON body) | `url_env` / `url`; optional `headers_env` map (header → env var **name**); optional `body` preset (`discord` / `teams` / `generic`) |
| **`loki`** | Grafana Loki push API (log stream) | `url_env` / `url` (Push API base or full `/loki/api/v1/push`); optional `headers_env` (e.g. basic auth / tenant); optional `labels` map (static label set, always include `job=gghstats`) |

#### Notifier examples (operator copy-paste shape)

**Several destinations at once (fan-out):**

```bash
GGHSTATS_ALERTS_ENABLED=true
GGHSTATS_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/...
GGHSTATS_DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/.../...
GGHSTATS_N8N_WEBHOOK_URL=https://n8n.example.com/webhook/gghstats

# One firing rule → Slack AND Discord AND n8n
GGHSTATS_ALERT_SINKS='[
  {"type":"slack","webhook_url_env":"GGHSTATS_SLACK_WEBHOOK_URL"},
  {"type":"webhook","url_env":"GGHSTATS_DISCORD_WEBHOOK_URL","body":"discord"},
  {"type":"webhook","url_env":"GGHSTATS_N8N_WEBHOOK_URL"}
]'
```

**Slack only:**

```bash
GGHSTATS_ALERTS_ENABLED=true
GGHSTATS_SLACK_WEBHOOK_URL=https://hooks.slack.com/services/T.../B.../...
GGHSTATS_ALERT_SINKS='[{"type":"slack","webhook_url_env":"GGHSTATS_SLACK_WEBHOOK_URL"}]'
```

**Discord (generic webhook + Discord URL):**

```bash
GGHSTATS_DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/.../...
GGHSTATS_ALERT_SINKS='[{"type":"webhook","url_env":"GGHSTATS_DISCORD_WEBHOOK_URL","body":"discord"}]'
```

**Teams (generic webhook):**

```bash
GGHSTATS_TEAMS_WEBHOOK_URL=https://....logic.azure.com/workflows/...
GGHSTATS_ALERT_SINKS='[{"type":"webhook","url_env":"GGHSTATS_TEAMS_WEBHOOK_URL","body":"teams"}]'
```

**n8n / custom + bearer from env:**

```bash
GGHSTATS_N8N_WEBHOOK_URL=https://n8n.example.com/webhook/gghstats
GGHSTATS_N8N_TOKEN=...
GGHSTATS_ALERT_SINKS='[{
  "type":"webhook",
  "url_env":"GGHSTATS_N8N_WEBHOOK_URL",
  "headers_env":{"Authorization":"GGHSTATS_N8N_TOKEN"}
}]'
# Runtime sends: Authorization: <value of GGHSTATS_N8N_TOKEN>  (prefix "Bearer " only if the env value includes it, or document auto-Bearer)
```

**SMTP (implemented — same pattern as [groot](https://github.com/hrodrig/groot) email notifier):**

```bash
GGHSTATS_SMTP_HOST=smtp.example.com
GGHSTATS_SMTP_PORT=587
GGHSTATS_SMTP_USER=alerts@example.com
GGHSTATS_SMTP_PASSWORD=...
GGHSTATS_SMTP_FROM=alerts@example.com
GGHSTATS_SMTP_TO=you@example.com
GGHSTATS_ALERT_SINKS='[{
  "type":"smtp",
  "host_env":"GGHSTATS_SMTP_HOST",
  "port_env":"GGHSTATS_SMTP_PORT",
  "user_env":"GGHSTATS_SMTP_USER",
  "password_env":"GGHSTATS_SMTP_PASSWORD",
  "from_env":"GGHSTATS_SMTP_FROM",
  "to_env":"GGHSTATS_SMTP_TO"
}]'
```

Port **587** uses STARTTLS; set `"use_tls":true` for implicit TLS (e.g. 465). `from_env` optional (defaults to `user`). Multiple `to` addresses: semicolon or comma separated.

**Loki (first-class sink — same band as Slack/webhook):**

```bash
GGHSTATS_LOKI_URL=https://loki.example.com/loki/api/v1/push
# Optional tenant / basic auth via headers_env
GGHSTATS_LOKI_TENANT=gghstats
GGHSTATS_ALERT_SINKS='[{
  "type":"loki",
  "url_env":"GGHSTATS_LOKI_URL",
  "headers_env":{"X-Scope-OrgID":"GGHSTATS_LOKI_TENANT"},
  "labels":{"job":"gghstats","source":"alert"}
}]'
```

Push body: Loki streams API — one stream with configured labels + line = canonical alert text (and/or structured JSON line). Inspired by **[pgwd](https://github.com/hrodrig/pgwd)** Slack+Loki delivery. Full Grafana/Loki **stack** still lives in **gghstats-selfhosted**; the binary only **pushes** alert lines when this sink is configured.

**How other chat products fit:**

| Product | In gghstats? | How |
|---------|--------------|-----|
| **Slack** | Yes (`type=slack`) | Native Incoming Webhook shape. |
| **Loki** | Yes (`type=loki`) | Push API; labels + log line. Operator queries in Grafana. |
| **Discord** | Via **`webhook`** | Discord Incoming Webhook URL; may need a thin JSON body map (`content` / embeds) — document one working example; no separate `type=discord` required for MVP. |
| **Microsoft Teams** | Via **`webhook`** | Incoming Webhook / Workflows URL; payload often Adaptive Card JSON — document one example; no `type=teams` in MVP unless Slack-like convenience is needed later. |
| **WhatsApp** | **No** (out of scope) | Needs Meta Cloud API / Twilio / BSP — credentials, templates, opt-in. Operator bridges: `webhook` → n8n/Make → WhatsApp. |
| **Email / SMTP** | Yes (`type=smtp`) | Plain-text body = canonical alert text; secrets via `*_env` (groot-style STARTTLS / TLS). |

Off until `GGHSTATS_ALERTS_ENABLED=true` **and** at least one sink is valid.

**Not first-class:** PagerDuty, **WhatsApp**, Discord/Teams *native* types (use generic `webhook`).  
**Sinks:** `slack`, `webhook`, **`loki`**, **`smtp`**.

Plain language → sink:

| What you mean | Sink |
|---------------|------|
| “Send it to my Slack channel.” | `type=slack` + Incoming Webhook URL |
| “Send it to Discord / Teams / n8n.” | `type=webhook` + that product’s webhook URL |
| “Push it to Loki / Grafana logs.” | `type=loki` + Push API URL |
| “Email me.” | `type=smtp` + host/user/password/to via `*_env` |
| “WhatsApp me.” | Not in-app — webhook → external automation |

### 8.6 Notification message style

**Principles:** plain text first (works everywhere); one alert = one short message; always include **gghstats version** (`version.Version`), **what / where / value / threshold / window**; no emoji spam; English (project language). Optional `GGHSTATS_PUBLIC_URL` link to the repo dashboard when set.

**Canonical text body** (Slack `text`, Discord `content`, SMTP body, webhook `text` field):

```text
gghstats alert
version: 0.10.0
repo:    hrodrig/pgwd
metric:  clones
window:  1d (UTC)
value:   241
rule:    gte 225
when:    2026-07-17T04:00:00Z
dash:    https://gghstats.example.com/repo/hrodrig/pgwd
```

Zero / floor example:

```text
gghstats alert
version: 0.10.0
repo:    hrodrig/groot
metric:  clones
window:  1d (UTC)
value:   0
rule:    eq 0
when:    2026-07-17T04:00:00Z
```

Fleet example:

```text
gghstats alert
version: 0.10.0
scope:   all_repos
metric:  clones
window:  lifetime
value:   30112
rule:    gte 30000
when:    2026-07-17T04:00:00Z
```

WoW drop:

```text
gghstats alert
version: 0.10.0
repo:    hrodrig/pgwd
metric:  clones
window:  wow
value:   -34%
rule:    drop_pct >= 30
detail:  this_week=120 last_week=182
when:    2026-07-17T04:00:00Z
```

Milestone (A2+):

```text
gghstats alert
version: 0.10.0
repo:    hrodrig/pgwd
metric:  stars
window:  milestone
value:   100
rule:    crossed 100 (next 500)
when:    2026-07-17T04:00:00Z
```

Ops / sync-health (§8.7):

```text
gghstats alert
version:  0.10.0
kind:     ops
level:    warn
event:    repo_fetch_failed
count:    4
threshold: 3
window:   this_sync
detail:   4/42 repos failed (network/5xx after retries)
when:     2026-07-17T04:00:00Z
```

```text
gghstats alert
version:  0.10.0
kind:     ops
level:    crit
event:    sync_failed
count:    2
threshold: 2
window:   consecutive_runs
detail:   last 2 scheduled syncs ended in error (github unreachable)
when:     2026-07-17T05:00:00Z
```

**Version field:** always the running binary version (`internal/version` / `gghstats version`) so operators know which release produced the alert.

**Slack** (`type=slack`): POST Incoming Webhook JSON:

```json
{
  "text": "gghstats alert\nversion: 0.10.0\nrepo: hrodrig/pgwd\nmetric: clones\nwindow: 1d (UTC)\nvalue: 241\nrule: gte 225\nwhen: 2026-07-17T04:00:00Z"
}
```

Optional later: Block Kit — **not** required for MVP.

**Discord** (`type=webhook`, `body=discord`):

```json
{
  "content": "gghstats alert\nversion: 0.10.0\nrepo: hrodrig/pgwd\nmetric: clones\nwindow: 1d (UTC)\nvalue: 241\nrule: gte 225\nwhen: 2026-07-17T04:00:00Z"
}
```

**Generic webhook** (`body=generic` or default) — machine-friendly + same text:

```json
{
  "source": "gghstats",
  "version": "0.10.0",
  "text": "gghstats alert\nversion: 0.10.0\nrepo: hrodrig/pgwd\nmetric: clones\nwindow: 1d (UTC)\nvalue: 241\nrule: gte 225\nwhen: 2026-07-17T04:00:00Z",
  "alert": {
    "repo": "hrodrig/pgwd",
    "scope": null,
    "metric": "clones",
    "window": "1d",
    "op": "gte",
    "threshold": 225,
    "value": 241,
    "fired_at": "2026-07-17T04:00:00Z",
    "dashboard_url": "https://gghstats.example.com/repo/hrodrig/pgwd"
  }
}
```

**Teams** (`body=teams`): wrap the same facts in a minimal Adaptive Card / MessageCard — document one template at implement time; facts must match the text body above.

Do **not** send HTML email-style blobs to Slack/Discord. Keep messages copy-pasteable into tickets.

Payloads the operator should see (shape, not final JSON) — short one-liners also OK as Slack fallback title; prefer the multi-line form above for clarity:

```text
gghstats/0.10.0 alert: hrodrig/pgwd clones today = 241 (threshold >= 225, window=1d UTC)
gghstats/0.10.0 alert: hrodrig/groot clones today = 0 (threshold == 0, window=1d UTC)
gghstats/0.10.0 alert: all_repos clones lifetime = 30112 (threshold >= 30000)
gghstats/0.10.0 alert: hrodrig/pgwd views today = 3 (threshold < 10, window=1d UTC)
gghstats/0.10.0 alert: hrodrig/pgwd clones WoW drop = 34% (threshold >= 30%)
gghstats/0.10.0 alert: hrodrig/pgwd stars = 100 (milestone crossed; next 500)
gghstats/0.10.0 ops warn: repo_fetch_failed count=4 (threshold >= 3, this_sync)
gghstats/0.10.0 ops crit: sync_failed consecutive=2 (threshold >= 2)
gghstats/0.10.0 ops warn: rate_limit remaining=87 (threshold < 100)
gghstats/0.10.0 ops crit: github_unreachable (this_sync)
```

**Worked example (zero clones today):**

> Operator: *Alert me if today groot has had no clones.*

Interpretation:

1. **Repo:** `hrodrig/groot`.
2. **Metric:** `clones` `count` for UTC today.
3. **Zero means:** row exists with `count=0`, **or** no `clones` row for that repo/date yet → treat as **0** (document this; do not skip the rule as “no data”).
4. **Operator:** equal to **0** (same as `lt` 1 for non-negative counts).
5. **Debounce:** `once_per_utc_day` so a quiet repo does not alert on every hourly sync after midnight UTC.
6. **Same env var:** object in `GGHSTATS_ALERT_RULES` (see array above) — not a separate variable per rule.

**Worked example (fleet lifetime clones ≥ 30000):**

> Operator: *Alert me when all clones exceed 30000.*

Interpretation:

1. **Scope:** `all_repos` — sum over every repo present in SQLite after sync (same set the dashboard would show under current filter, if filter applies — document whether `GGHSTATS_FILTER` narrows the sum).
2. **Metric:** `clones` `count` summed across **all dates** (`window=lifetime`), not “today only.”
3. **Not:** “each repo &gt; 30000” (that would be `scope=each_repo`). **Not:** sum of today’s clones only (`window=1d` + `scope=all_repos`).
4. **Operator:** **≥ 30000**.
5. **Fire-once** recommended (`debounce=once` / `fire=once`): after the fleet crosses 30k, do not re-alert every sync; optional reset documented later.
6. **Payload:** no single repo — say `all_repos` / `fleet` and the **total**.

### 8.7 Ops and sync-health alerts

**Purpose:** notify when gghstats **cannot reliably fetch** metrics — not when traffic numbers cross a bar. Same sinks as traffic rules (§8.5). Distinct `kind=ops` in payloads so operators and Loki labels can filter.

Prometheus gauges (`gghstats_last_sync_*`, `gghstats_github_rate_limit_*`) remain the primary **dashboard** path; this section is **push** alerts for operators without a full Grafana stack, or in addition to it.

#### Events (what can fire)

| `event` | Meaning | Count / quantity |
|---------|---------|------------------|
| **`sync_failed`** | An entire sync run ended in error (or aborted before useful work). | Consecutive failed runs (`window=consecutive_runs`) or failures in a time window. |
| **`repo_fetch_failed`** | One or more repos failed their traffic/star fetch **after retries** in a single run. | Failed repo count in **this sync** (e.g. `count ≥ 3`). |
| **`github_unreachable`** | Transport / DNS / dial failures dominate (no usable GitHub HTTP). | Failures in this sync or consecutive runs — document which. |
| **`rate_limit`** | Core REST remaining below a floor, **or** sustained 429 after retries exhausted. | Remaining ≤ `value`, and/or consecutive rate-limit outcomes ≥ `value`. |

Partial success is normal on large accounts: **do not** alert on every single-repo blip. Rules must use **thresholds** (counts / consecutive / remaining).

#### Levels (severity)

| `level` | When to use | Typical mapping |
|---------|-------------|-----------------|
| **`info`** | Notable but expected noise (optional; default off). | e.g. 1 repo failed once |
| **`warn`** | Degraded but sync mostly worked. | e.g. `repo_fetch_failed` count ≥ N; rate limit remaining low |
| **`crit`** | Operator action likely needed. | e.g. whole `sync_failed` × consecutive ≥ 2; GitHub unreachable |

Rules **must** set `level` (or inherit a documented default per `event`). Sinks that support severity (Slack optional prefix, Loki label `level=…`) carry it through. Message body always includes `level:` and `event:`.

#### Quantities (normative knobs)

Every ops rule names:

| Field | Role |
|-------|------|
| **`event`** | One of the events above. |
| **`op` + `value`** | Threshold on the **count** (or remaining for rate limit), e.g. `gte` / `3`. |
| **`window`** | `this_sync` \| `consecutive_runs` \| optional duration later. |
| **`level`** | `info` \| `warn` \| `crit`. |
| **`debounce`** | Avoid spam (e.g. `once_per_utc_day` for warn; crit may allow every consecutive failure until recover). |

**Recommended starter thresholds** (copy into env.example comments; operators may tighten):

| Starter | Why that number |
|---------|-----------------|
| **≥ 3** repo failures / sync → warn | Ignores one-off 5xx on a single repo; still catches “something’s wrong with a chunk of the fleet.” |
| **2** consecutive full sync failures → crit | One miss can be deploy/restart; two means the loop is broken. |
| Remaining **&lt; 100** → warn | Early signal before hard 429 storms; not “already at zero.” |
| Unreachable **≥ 1** this sync → crit | No GitHub = no product data; do not wait for a second run. |

#### Plain-language examples → config

Docs (README / env.example) must show **human sentence → what gghstats checks**, same style as §8.4.

| What you mean | Event / level | Check (after sync or on sync error) | Draft config sketch |
|---------------|---------------|-------------------------------------|---------------------|
| “Warn me if **this sync** left **3 or more** repos without fresh traffic (after retries) — one flaky repo is OK.” | `repo_fetch_failed` / **warn** | Failed-repo count in **this sync** **≥ 3** | `kind=ops`, `event=repo_fetch_failed`, `window=this_sync`, `op=gte`, `value=3`, `level=warn`, `debounce=once_per_utc_day` |
| “Page me if **two scheduled syncs in a row** die completely — I got no metrics at all.” | `sync_failed` / **crit** | Consecutive full-run failures **≥ 2** | `kind=ops`, `event=sync_failed`, `window=consecutive_runs`, `op=gte`, `value=2`, `level=crit` |
| “Warn me when GitHub REST remaining drops **below 100** after a sync — we are about to burn the quota.” | `rate_limit` / **warn** | `X-RateLimit-Remaining` **&lt; 100** (core REST) | `kind=ops`, `event=rate_limit`, `op=lt`, `value=100`, `level=warn`, `debounce=once_per_utc_day` |
| “Crit if **this sync** cannot reach GitHub at all (DNS/dial/TLS) — not a single API call succeeded.” | `github_unreachable` / **crit** | Unreachable outcome **≥ 1** in this sync | `kind=ops`, `event=github_unreachable`, `window=this_sync`, `op=gte`, `value=1`, `level=crit` |
| “Warn if **half or more** of the fleet failed fetch in one run (large account).” | `repo_fetch_failed` / **warn** | Ratio form = later stretch; MVP uses a fixed high `value` | Prefer fixed `value` in MVP |
| “Info only: note when **exactly one** repo fails (Loki), do not ping Slack.” | `repo_fetch_failed` / **info** | Failed count **≥ 1** | `level=info`; MVP may skip or use Loki-only sinks |

**Worked example (several repos failed this sync):**

> Operator: *Warn me if this sync left 3 or more repos without fresh traffic — one flaky repo is OK.*

Interpretation:

1. **Kind:** `ops` (not a clones/views threshold).
2. **Event:** `repo_fetch_failed` — per-repo step failed **after** GitHub client retries (§4.5).
3. **Window:** `this_sync` — only the run that just finished.
4. **Quantity:** failed repo count **≥ 3** (example: 4 of 42 failed → fire; 1 of 42 → silent).
5. **Level:** `warn` — sync mostly succeeded; check logs / rate limit / token scopes.
6. **Debounce:** `once_per_utc_day` so hourly sync does not re-warn while the same repos stay broken.
7. **Payload:** include `count`, `threshold`, optional capped sample of failed `owner/name`.

**Worked example (two dead syncs in a row):**

> Operator: *Page me if two scheduled syncs in a row die completely — I got no metrics at all.*

Interpretation:

1. **Event:** `sync_failed` — whole run returned error / aborted before useful upserts (exact definition at implement time).
2. **Window:** `consecutive_runs` — increments on failure; resets to 0 on successful sync.
3. **Quantity:** consecutive failures **≥ 2**.
4. **Level:** `crit`.
5. **Debounce:** prefer **fire until recover** for crit (each further consecutive failure may notify); document the choice.
6. **Not the same as:** “3 repos failed but the run finished OK” → that is `repo_fetch_failed`, not `sync_failed`.

**Worked example (quota almost gone):**

> Operator: *Warn me when GitHub REST remaining drops below 100 after a sync.*

Interpretation:

1. **Event:** `rate_limit`.
2. **Value source:** last observed `X-RateLimit-Remaining` for core REST (same series as `gghstats_github_rate_limit_remaining`).
3. **Operator:** **&lt; 100** (`op=lt`, `value=100`).
4. **Level:** `warn`.
5. **When:** end of sync (or when the gauge updates); not a continuous poller.
6. **Debounce:** `once_per_utc_day`.

**Worked example (GitHub unreachable):**

> Operator: *Crit if this sync cannot reach GitHub at all.*

Interpretation:

1. **Event:** `github_unreachable` — dial/DNS/TLS/connection errors dominate; no successful GitHub HTTP in the run.
2. **Quantity:** **≥ 1** (`value=1`).
3. **Level:** `crit`.
4. **Distinct from:** live API **429** → `rate_limit`; some repos **5xx** with others OK → `repo_fetch_failed`.

Pseudo-env (ops + traffic in the **same** `GGHSTATS_ALERT_RULES` array):

```bash
GGHSTATS_ALERTS_ENABLED=true
# Plain language (ops):
#   - "Warn me if this sync left 3+ repos without fresh traffic — one flaky repo is OK."
#   - "Page me if two scheduled syncs in a row die completely."
#   - "Warn me when GitHub REST remaining drops below 100 after a sync."
#   - "Crit if this sync cannot reach GitHub at all."
GGHSTATS_ALERT_RULES='[
  {"kind":"traffic","repo":"hrodrig/pgwd","metric":"clones","window":"1d","op":"gte","value":225,"debounce":"once_per_utc_day"},
  {"kind":"ops","event":"repo_fetch_failed","window":"this_sync","op":"gte","value":3,"level":"warn","debounce":"once_per_utc_day"},
  {"kind":"ops","event":"sync_failed","window":"consecutive_runs","op":"gte","value":2,"level":"crit"},
  {"kind":"ops","event":"rate_limit","op":"lt","value":100,"level":"warn","debounce":"once_per_utc_day"},
  {"kind":"ops","event":"github_unreachable","window":"this_sync","op":"gte","value":1,"level":"crit"}
]'
```

**MVP sequencing:** sinks first (including Loki); **`gghstats alert test`** next (§8.8); traffic rules then ops rules — do not ship rule evaluation without a way for operators to prove delivery. Prometheus remains complementary, not a substitute for operators who only want Slack/Loki push.

### 8.8 `gghstats alert test` — sink smoke test

Operators must be able to **validate notifications before real traffic/ops rules fire** (same need as **pgwd** `-force-notification` and **groot** / **kzero** `notify test`).

**CLI (normative):**

```text
gghstats alert test [--kind traffic|ops] [--sink TYPE] ...
```

| Aspect | Contract |
|--------|----------|
| **Purpose** | POST one **synthetic** payload to configured sinks. Prove URLs, secrets, Loki labels, and message shape. |
| **Does not** | Start `serve`, run sync, open SQLite for rule evaluation, or require GitHub. |
| **Config source** | Same env as serve: `GGHSTATS_ALERT_SINKS` (+ secret env vars). Optional: require `GGHSTATS_ALERTS_ENABLED=true`, **or** allow test when sinks are non-empty even if enabled is false (document one rule — prefer: **sinks non-empty is enough** for test; `ENABLED` gates post-sync evaluation only). |
| **Default payload** | Synthetic traffic-shaped message (`kind=traffic`, metric/repo placeholders, `rule: delivery check`) including **gghstats version**. |
| **`--kind ops`** | Synthetic ops payload (`event=alert_test`, `level=info`, detail = delivery check). |
| **`--sink TYPE`** | Optional filter: only that sink type (`slack` / `webhook` / `loki` / `smtp`). Default = **fan-out all** resolved sinks. |
| **Stdout success** | e.g. `alert test: sent kind "traffic" to N sink(s)`. |
| **Exit codes** | **0** all targeted sinks succeeded; **1** config/parse error or no sinks; **non-zero** (recommend **4**, aligned with groot/kzero) if any targeted sink failed delivery. |
| **Fail closed** | No sinks / empty URL after `*_env` resolve → exit **1**, do not pretend success. |

**Not** a long-running serve flag (unlike pgwd’s `-force-notification` on each check cycle). gghstats `serve` is a daemon; smoke-test is a **one-shot CLI** so operators validate before cron/Compose/Helm turn on rules.

**Plain language:**

> Operator: *Send a test alert to my Slack and Loki so I know webhooks work before I enable clone rules.*

→ `gghstats alert test` (or `… --kind ops`) with `GGHSTATS_ALERT_SINKS` set.

**Related family (same maintainer):**

| Tool | Smoke-test shape |
|------|------------------|
| **pgwd** | `-force-notification` / `PGWD_FORCE_NOTIFICATION` during a check run |
| **groot** | `groot notify test [--event …]` |
| **kzero** | `kzero notify test [--event …]` |
| **gghstats** | `gghstats alert test [--kind …]` |

Payload must still follow §8.6 (version line, English, no emoji spam). Mark synthetic clearly, e.g. `rule: delivery check` or `event: alert_test`.
