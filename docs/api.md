# HTTP JSON API — consumer guide

How to call **gghstats** from scripts, Grafana, or your own frontend (React, Svelte, …).

**Normative contracts** (status codes, sync rules): [SPEC.md](../SPEC.md) §2–§3.  
**Operator env table:** [README.md](../README.md).

As of **v0.11.0**.

---

## Quick start

```bash
export BASE=http://127.0.0.1:8080
export TOKEN=your-api-token   # must match GGHSTATS_API_TOKEN

# Liveness (no auth)
curl -sS "$BASE/api/v1/healthz"
# {"status":"ok"}

# List repos (auth required)
curl -sS -H "x-api-token: $TOKEN" "$BASE/api/repos"
```

**API-only backend** (no HTML dashboard):

```bash
export GGHSTATS_API_ONLY=true
export GGHSTATS_API_TOKEN=your-api-token
# Prefer an explicit browser origin when the SPA talks to the API directly:
# export GGHSTATS_CORS_ORIGINS=https://app.example.com
gghstats serve
```

---

## Authentication

| Situation | Behavior |
|-----------|----------|
| `GGHSTATS_API_TOKEN` **unset** | Authenticated JSON routes return **404** (API disabled). |
| Token set, header missing/wrong | **401** `{"error":"unauthorized"}` |
| Token set, header matches | Request proceeds |

Send the token on every protected call:

```http
x-api-token: your-api-token
```

**Public without token:** `GET /api/v1/healthz`, `GET /metrics` (unless disabled), and badges when `GGHSTATS_BADGE_PUBLIC=true` (default).

**Browser SPAs:** do **not** put `GGHSTATS_API_TOKEN` in a public bundle. Use a BFF/proxy that adds `x-api-token`, or restrict CORS. With `GGHSTATS_API_ONLY` and open CORS (`*`), `serve` logs a startup warning.

A matching `x-api-token` also **bypasses the IP whitelist** on protected paths (token is still validated).

---

## CORS

| Env | Meaning |
|-----|---------|
| `GGHSTATS_CORS_ORIGINS` empty | `Access-Control-Allow-Origin: *` on authenticated JSON success (compat) |
| Comma-separated list | Echo `Origin` when it matches; otherwise omit the header |

Applies to authenticated JSON handlers (repos, traffic, dogfood, sync, …), not to healthz/badges.

---

## Error shapes

| HTTP | Body (typical) | When |
|------|----------------|------|
| 401 | `{"error":"unauthorized"}` | Bad/missing `x-api-token` |
| 404 | `{"error":"not_found"}` or plain 404 | Unknown repo, or API disabled (no token configured) |
| 400 | `{"error":"…"}` | Bad query/path (e.g. H2H missing `a`/`b`) |
| 409 | `{"error":"sync_in_progress",…}` | Sync already running |
| 500 | `{"error":"…"}` | Database / internal |

---

## Route index

| Method | Path | Auth | Role |
|--------|------|------|------|
| GET | `/api/v1/healthz` | No | Liveness |
| GET | `/api/v1/badge/{owner}/{repo}` | Optional | SVG badge |
| GET | `/api/repos` | Yes | Index list + KPIs |
| GET | `/api/v1/charts/index-clones` | Yes | Index clones chart series |
| GET | `/api/v1/repos/{owner}/{repo}` | Yes | Repo summary + momentum |
| GET | `/api/v1/repos/{owner}/{repo}/traffic` | Yes | Clones/views time series |
| GET | `/api/v1/repos/{owner}/{repo}/stars` | Yes | Star history |
| GET | `/api/v1/repos/{owner}/{repo}/popular` | Yes | Referrers + paths (14d) |
| GET | `/api/v1/h2h` | Yes | Head-to-head compare |
| GET/POST | `/api/v1/sync` | Yes | Sync status / trigger |
| GET | `/metrics` | No* | Prometheus |

\* Off with `GGHSTATS_METRICS=false`.

---

## Dogfood map (official UI → API)

Use this when rebuilding the dashboard in another UI.

| UI surface | Calls |
|------------|--------|
| **Index** | `GET /api/repos` (+ optional `sort`/`dir`/`q`/`page`) and `GET /api/v1/charts/index-clones` |
| **Repo page** | `GET /api/v1/repos/{o}/{r}`, `…/traffic?days=365` (or 30), `…/stars`, `…/popular` |
| **H2H** | `GET /api/v1/h2h?a=owner/a&b=owner/b&w=7d` |
| **Sync button** | `POST /api/v1/sync` or `POST /api/v1/sync?repo=owner/name`; poll `GET /api/v1/sync` |

---

## Endpoints

### `GET /api/v1/healthz`

```bash
curl -sS "$BASE/api/v1/healthz"
```

```json
{"status":"ok"}
```

---

### `GET /api/repos`

List repositories with aggregate KPIs (dogfood for the index).

**Query**

| Param | Default | Notes |
|-------|---------|--------|
| `sort` | `total_views` | `name`, `stars`, `forks`, `total_views`, `total_clones`, `clones_1d`, `clones_7d`, `clones_30d` |
| `dir` | `desc` | `asc` or `desc` |
| `q` | (empty) | Case-insensitive substring on `owner/repo` name |
| `page`, `per_page` | — | Pagination **only if either is present**. Default `per_page`=25, max 100. Without them, all matching items are returned (compat). |

```bash
curl -sS -H "x-api-token: $TOKEN" \
  "$BASE/api/repos?sort=total_clones&dir=desc&q=hrodrig&page=1&per_page=25"
```

**Response (200)** — illustrative:

```json
{
  "total_count": 2,
  "total_stars": 120,
  "total_forks": 8,
  "total_views": 9000,
  "total_clones": 1500,
  "sort": "total_clones",
  "dir": "desc",
  "q": "hrodrig",
  "page": 1,
  "per_page": 25,
  "total_pages": 1,
  "items": [
    {
      "name": "hrodrig/gghstats",
      "description": "Self-hosted GitHub traffic stats",
      "stars": 100,
      "forks": 5,
      "watchers": 100,
      "issues": 2,
      "prs": 1,
      "fork": false,
      "archived": false,
      "total_views": 5000,
      "total_uniques": 800,
      "total_clones": 900,
      "clone_uniques": 200,
      "clones_1d": 12,
      "clones_7d": 80,
      "clones_30d": 300
    }
  ]
}
```

---

### `GET /api/v1/charts/index-clones`

Aggregated daily clones across the **same filter** as `/api/repos` (`sort`/`dir`/`q`). No pagination.

```bash
curl -sS -H "x-api-token: $TOKEN" \
  "$BASE/api/v1/charts/index-clones?q=hrodrig"
```

```json
{
  "count": 90,
  "series": [
    {"date": "2026-04-01", "count": 40, "uniques": 10}
  ],
  "sort": "total_views",
  "dir": "desc",
  "q": "hrodrig"
}
```

Window is capped (~120 days ending at the newest clone date in the filtered set).

---

### `GET /api/v1/repos/{owner}/{repo}`

Repo summary + clone momentum (same idea as the HTML repo page).

```bash
curl -sS -H "x-api-token: $TOKEN" \
  "$BASE/api/v1/repos/hrodrig/gghstats"
```

```json
{
  "repo": {
    "name": "hrodrig/gghstats",
    "description": "…",
    "stars": 100,
    "forks": 5,
    "watchers": 100,
    "issues": 2,
    "prs": 1,
    "fork": false,
    "archived": false,
    "total_views": 5000,
    "total_uniques": 800,
    "total_clones": 900,
    "clone_uniques": 200,
    "clones_1d": 12,
    "clones_7d": 80,
    "clones_30d": 300
  },
  "momentum_7d": 0.15,
  "momentum_30d": -0.05,
  "momentum_7d_pct": "+15%",
  "momentum_30d_pct": "-5%"
}
```

Unknown repo → **404** `{"error":"not_found"}`.

---

### `GET /api/v1/repos/{owner}/{repo}/traffic`

Daily clones and views.

| Query | Default | Notes |
|-------|---------|--------|
| `days` | `30` | UTC rolling window inclusive of today. `0` = all stored days. Max `3660`. |

```bash
curl -sS -H "x-api-token: $TOKEN" \
  "$BASE/api/v1/repos/hrodrig/gghstats/traffic?days=30"
```

```json
{
  "name": "hrodrig/gghstats",
  "days": 30,
  "from": "2026-06-23",
  "to": "2026-07-22",
  "clones": [
    {"date": "2026-07-01", "count": 10, "uniques": 4}
  ],
  "views": [
    {"date": "2026-07-01", "count": 50, "uniques": 20}
  ]
}
```

Missing calendar days are **omitted** (not zero-filled).

---

### `GET /api/v1/repos/{owner}/{repo}/stars`

Cumulative star history (`date`, `total`).

```bash
curl -sS -H "x-api-token: $TOKEN" \
  "$BASE/api/v1/repos/hrodrig/gghstats/stars"
```

```json
{
  "name": "hrodrig/gghstats",
  "stars": [
    {"date": "2026-01-15", "total": 10},
    {"date": "2026-03-01", "total": 42}
  ]
}
```

---

### `GET /api/v1/repos/{owner}/{repo}/popular`

Top referrers and paths for the last **14** UTC days (same window as the HTML repo page).

```bash
curl -sS -H "x-api-token: $TOKEN" \
  "$BASE/api/v1/repos/hrodrig/gghstats/popular"
```

```json
{
  "name": "hrodrig/gghstats",
  "days": 14,
  "referrers": [
    {"name": "github.com", "count": 100, "uniques": 40}
  ],
  "paths": [
    {"name": "/", "count": 80, "uniques": 30}
  ]
}
```

---

### `GET /api/v1/h2h`

Head-to-head scores and chart series.

| Query | Required | Notes |
|-------|----------|--------|
| `a` | yes | `owner/repo` |
| `b` | yes | `owner/repo` (must differ from `a`) |
| `w` | no | `7d` (default), `30d`, or `total` |

```bash
curl -sS -H "x-api-token: $TOKEN" \
  "$BASE/api/v1/h2h?a=hrodrig/gghstats&b=hrodrig/pgwd&w=7d"
```

```json
{
  "a": "hrodrig/gghstats",
  "b": "hrodrig/pgwd",
  "interval": "7d",
  "result": {
    "interval": "7d",
    "repo_a": "hrodrig/gghstats",
    "repo_b": "hrodrig/pgwd",
    "score_a": 58,
    "score_b": 42,
    "delta_pct": 16,
    "leads_a": true,
    "rows": [
      {
        "key": "clones_7d",
        "label": "Clones (7d)",
        "weight_pct": 50,
        "value_a": 80,
        "value_b": 40,
        "leads_a": true
      }
    ],
    "suggest": {
      "confidence": "medium",
      "rationale": "…",
      "show": true
    }
  },
  "charts": {
    "repoA": "hrodrig/gghstats",
    "repoB": "hrodrig/pgwd",
    "showMomentum": true,
    "cloneLabels": ["2026-07-16", "2026-07-17"],
    "clonesA": [10, 12],
    "clonesB": [5, 6],
    "viewLabels": ["2026-07-16", "2026-07-17"],
    "viewsA": [40, 44],
    "viewsB": [20, 22],
    "momentumLabels": ["2026-07-16"],
    "momentumA": [0.1],
    "momentumB": [-0.05]
  }
}
```

Scores are **shares 0–100 that sum to 100** (same formula as the HTML H2H page).

---

### `POST /api/v1/sync` · `GET /api/v1/sync`

```bash
# Full sync (respects GGHSTATS_FILTER)
curl -sS -X POST -H "x-api-token: $TOKEN" "$BASE/api/v1/sync"
# 202 {"status":"started","scope":"all"}

# Single repo
curl -sS -X POST -H "x-api-token: $TOKEN" \
  "$BASE/api/v1/sync?repo=hrodrig/gghstats"
# 202 {"status":"started","scope":"repo","repo":"hrodrig/gghstats"}

# Status
curl -sS -H "x-api-token: $TOKEN" "$BASE/api/v1/sync"
```

Example status:

```json
{
  "running": false,
  "scope": "",
  "repo": "",
  "last_started_at": "2026-07-22T12:00:00Z",
  "last_finished_at": "2026-07-22T12:01:30Z",
  "last_error": ""
}
```

Only one sync at a time → **409** if already running.

---

### `GET /api/v1/badge/{owner}/{repo}`

SVG for README embeds. Public by default.

```bash
curl -sS "$BASE/api/v1/badge/hrodrig/gghstats?metric=clones" -o badge.svg
```

| Query | Values | Default |
|-------|--------|---------|
| `metric` | `clones`, `clones_30d`, `views`, `stars` | `clones` |
| `style` | `flat`, `flat-square` | `flat` |
| `label` | custom left text | metric name |

Set `GGHSTATS_BADGE_PUBLIC=false` to require `x-api-token` (breaks raw GitHub image embeds without a proxy).

---

## Minimal TypeScript client sketch

```ts
const BASE = import.meta.env.VITE_GGHSTATS_BASE; // e.g. https://stats.example.com
// Prefer a same-origin BFF that injects x-api-token — do not ship the token to the browser.

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.headers ?? {}),
      // Only if calling the API from a trusted server:
      // "x-api-token": process.env.GGHSTATS_API_TOKEN!,
    },
  });
  if (!res.ok) throw new Error(`${res.status} ${await res.text()}`);
  return res.json() as Promise<T>;
}

const index = await api<{ items: { name: string }[] }>("/api/repos?sort=total_clones&dir=desc");
const repo = await api(`/api/v1/repos/${owner}/${name}`);
const h2h = await api(`/api/v1/h2h?a=${a}&b=${b}&w=7d`);
```

---

## Related

- [SPEC.md](../SPEC.md) — normative HTTP + sync contracts  
- [README.md](../README.md) — install, env, HTML UI  
- [plan-v0.11.x.md](plan-v0.11.x.md) — API-only band scope  
