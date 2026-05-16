# gghstats

![gghstats — self-hosted GitHub traffic beyond the 14-day window](assets/gghstats-poster-devto.png)

[![Version](https://img.shields.io/badge/version-0.2.1-blue)](https://github.com/hrodrig/gghstats/releases)
[![Release](https://img.shields.io/github/v/release/hrodrig/gghstats)](https://github.com/hrodrig/gghstats/releases)
[![CI](https://github.com/hrodrig/gghstats/actions/workflows/ci.yml/badge.svg)](https://github.com/hrodrig/gghstats/actions)
[![codecov](https://codecov.io/gh/hrodrig/gghstats/graph/badge.svg)](https://codecov.io/gh/hrodrig/gghstats)
[![Go 1.26.3](https://img.shields.io/badge/go-1.26.3-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/hrodrig/gghstats)](https://pkg.go.dev/github.com/hrodrig/gghstats)
[![Go Report Card](https://goreportcard.com/badge/github.com/hrodrig/gghstats)](https://goreportcard.com/report/github.com/hrodrig/gghstats)
[![deps.dev](https://img.shields.io/badge/deps.dev-go%20module-blue)](https://deps.dev/go/github.com%2Fhrodrig%2Fgghstats)

**Repo:** [github.com/hrodrig/gghstats](https://github.com/hrodrig/gghstats) · **Releases:** [Releases](https://github.com/hrodrig/gghstats/releases)

Self-hosted dashboard and CLI for GitHub repository traffic stats. GitHub only keeps traffic for 14 days; `gghstats` keeps historical data indefinitely in SQLite.

If you want your **own self-hosted** deployment (Docker Compose, Traefik with TLS, Helm, optional Prometheus/Grafana/Loki), use the companion repo **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** — it lists the supported options and example manifests.

**Releases:** [GitHub Releases](https://github.com/hrodrig/gghstats/releases) ship binaries (tarballs/zip + checksums). **Multi-arch** container images (`linux/amd64`, `linux/arm64`) are on [GHCR](https://github.com/hrodrig/gghstats/pkgs/container/gghstats) as `ghcr.io/hrodrig/gghstats:v<version>` (same `v` prefix as the Git tag, e.g. `v0.2.1`) and `:latest`. Pushing a `v*` tag on `main` triggers the [Release workflow](.github/workflows/release.yml) (GoReleaser). Day-to-day work happens on `develop` (see [Release workflow](#release-workflow)).

## Demo

**Live:** [gghstats.hermesrodriguez.com](https://gghstats.hermesrodriguez.com)

![GGHSTATS dashboard — repository metrics and neobrutalist UI](assets/gghstats-main.png)

## Table of contents

- [Demo](#demo)
- [Features](#features)
- [Repository page charts](#repository-page-charts-clones--views)
- [Quick start](#quick-start)
- [Install](#install)
- [Web UI assets (developers)](#web-ui-assets-developers)
- [Usage](#usage)
- [Examples](#examples)
- [Configuration](#configuration)
- [Environment file](#environment-file)
- [Custom UI theme (optional)](#custom-ui-theme-optional)
- [HTTP API (JSON)](#http-api-json)
- [Typical scenarios](#typical-scenarios)
- [Deployments](#deployments)
- [Troubleshooting](#troubleshooting)
- [Release workflow](#release-workflow)
- [Security and quality](#security-and-quality)
- [Database](#database)
- [Community standards](#community-standards)
- [Acknowledgments](#acknowledgments)
- [License](#license)

## Features

- Collects views, clones, referrers, popular paths, and star history
- Auto-discovers repositories (or filters by org/repo rules)
- Web dashboard with Chart.js graphs
- JSON API for external integrations
- CLI mode for fetch/report/export
- Single binary, SQLite storage, no external DB dependency
- Docker image on GHCR; Compose / Helm examples live in **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**

### Repository page charts (Clones & Views)

On each repository’s detail page, the **Clones** and **Views** bar charts are **stacked** from GitHub’s daily traffic API:

| Segment | Meaning | GitHub field |
|--------|---------|--------------|
| **Lower** (theme primary color) | Unique visitors or cloners that day | `uniques` |
| **Upper** (theme info color) | Total views or total clones that day | `count` |

Exact colors depend on light/dark theme (Bootstrap `--bs-primary` / `--bs-info`, overridden in the app’s neo-brutalist CSS). The legend is hidden on the chart; use the tooltip on each bar for values.

[Back to top](#gghstats)

## Quick start

### Docker Compose (build from this repo)

```bash
cp .env.example .env
# Edit .env: set GGHSTATS_GITHUB_TOKEN (and optionally GGHSTATS_FILTER, GGHSTATS_PORT, etc.)
docker compose up -d --build
```

Open <http://localhost:8080>.

The template [`.env.example`](.env.example) lists variables for the **Go binary** and this dev Compose file. Production (Traefik, published image, Helm, observability) is in **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**.

### Plain Docker

```bash
docker run -d \
  -e GGHSTATS_GITHUB_TOKEN=ghp_xxx \
  -e GGHSTATS_FILTER="your-github-user/*" \
  -p 8080:8080 \
  -v ./data:/data \
  --name gghstats \
  ghcr.io/hrodrig/gghstats:v0.2.1
```

[Back to top](#gghstats)

## Install

### Go install

```bash
go install github.com/hrodrig/gghstats/cmd/gghstats@latest
```

### Pre-built binary and container

- **Binary archives:** [Releases](https://github.com/hrodrig/gghstats/releases) (pick OS/arch; verify `checksums.txt`).
- **OCI image:** `ghcr.io/hrodrig/gghstats:v0.2.1` or `ghcr.io/hrodrig/gghstats:latest` (image tag matches the Git release tag; multi-arch manifest).

### Build from source

```bash
git clone https://github.com/hrodrig/gghstats.git
cd gghstats
make install
```

### Web UI assets (developers)

Favicons and the [web app manifest](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Manifest) live under [`assets/favicons/`](assets/favicons/) and are embedded at build time via [`assets/embed.go`](assets/embed.go) (`go:embed favicons/*`). The HTTP server exposes each file under `/static/<filename>` (see table). Other UI assets (CSS, Bootstrap) remain under [`web/static/`](web/static/) via [`web/embed.go`](web/embed.go).

| File | Role |
|------|------|
| [`assets/favicons/favicon.svg`](assets/favicons/favicon.svg) | **Source artwork** (vector). Edit this when changing the mark; regenerate the raster files below. |
| [`assets/favicons/favicon-16x16.png`](assets/favicons/favicon-16x16.png) | PNG **16×16** (tabs, legacy). |
| [`assets/favicons/favicon-32x32.png`](assets/favicons/favicon-32x32.png) | PNG **32×32**. |
| [`assets/favicons/favicon.ico`](assets/favicons/favicon.ico) | Multi-size **ICO** (16 + 32). |
| [`assets/favicons/apple-touch-icon.png`](assets/favicons/apple-touch-icon.png) | **180×180** (iOS / “Add to Home Screen”). |
| [`assets/favicons/android-chrome-192x192.png`](assets/favicons/android-chrome-192x192.png) | **192×192** (PWA / Android). |
| [`assets/favicons/android-chrome-512x512.png`](assets/favicons/android-chrome-512x512.png) | **512×512** (PWA splash / install). |
| [`assets/favicons/manifest.json`](assets/favicons/manifest.json) | [Web app manifest](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Manifest) (`/static/manifest.json`; linked from `layout.html`). |
| [`assets/gghstats-main-theme-bootstrap-plain.png`](assets/gghstats-main-theme-bootstrap-plain.png) | **Documentation only:** screenshot of the optional **Bootstrap-plain** theme ([`contrib/themes/example-bootstrap-plain.css`](contrib/themes/example-bootstrap-plain.css)); not embedded in the binary or served by the app. |

**Regenerating rasters after you change `favicon.svg`:** from the repository root, with [librsvg](https://wiki.gnome.org/Projects/LibRsvg) (`rsvg-convert`) and [ImageMagick](https://imagemagick.org/) (`magick`) on your `PATH`:

```bash
SVG=assets/favicons/favicon.svg
rsvg-convert -w 16  -h 16  "$SVG" -o assets/favicons/favicon-16x16.png
rsvg-convert -w 32  -h 32  "$SVG" -o assets/favicons/favicon-32x32.png
rsvg-convert -w 180 -h 180 "$SVG" -o assets/favicons/apple-touch-icon.png
rsvg-convert -w 192 -h 192 "$SVG" -o assets/favicons/android-chrome-192x192.png
rsvg-convert -w 512 -h 512 "$SVG" -o assets/favicons/android-chrome-512x512.png
magick assets/favicons/favicon-16x16.png assets/favicons/favicon-32x32.png assets/favicons/favicon.ico
```

Commit everything under `assets/favicons/` together so all icons stay in sync.

[Back to top](#gghstats)

## Usage

### Server mode (recommended)

```bash
export GGHSTATS_GITHUB_TOKEN="ghp_your_token"
gghstats serve
```

Server behavior:

- Runs initial sync when database is empty
- Re-syncs on schedule (default `1h`)
- Serves dashboard on <http://localhost:8080>
- Stores data in `./data/gghstats.db`
- Liveness/readiness: `GET /api/v1/healthz` → `{"status":"ok"}` (no auth; Kubernetes-style)
- Prometheus: `GET /metrics` (disable with `GGHSTATS_METRICS=false`)
- Listen port: `GGHSTATS_PORT` (default `8080`) or `gghstats serve --port <port>`
- First stderr line on start: version, build date, `GOOS`/`GOARCH`, listen address, masked GitHub token (`XXXX....YYYY`); then slog at `GGHSTATS_LOG_LEVEL` (default `info`). Every structured slog line is prefixed with `gghstats ` so it is easy to grep in shared log streams.

### CLI mode

```bash
gghstats fetch --repo your-github-user/my-app --token "$GGHSTATS_GITHUB_TOKEN"
gghstats report --repo your-github-user/my-app --token "$GGHSTATS_GITHUB_TOKEN"
gghstats export --repo your-github-user/my-app --token "$GGHSTATS_GITHUB_TOKEN" --output traffic.csv
```

[Back to top](#gghstats)

## Examples

### Start server with explicit DB path and interval

```bash
GGHSTATS_GITHUB_TOKEN=ghp_xxx \
GGHSTATS_DB=./data/gghstats.db \
GGHSTATS_SYNC_INTERVAL=30m \
gghstats serve
```

### Fetch/report/export for one repository

Use your repository as `owner/repo` (example below uses a placeholder).

```bash
gghstats fetch --repo your-github-user/my-app --token "$GGHSTATS_GITHUB_TOKEN"
gghstats report --repo your-github-user/my-app --token "$GGHSTATS_GITHUB_TOKEN" --days 14
gghstats export --repo your-github-user/my-app --token "$GGHSTATS_GITHUB_TOKEN" --days 30 --output traffic-30d.csv
```

### Run strict pre-release checks (includes container scan)

```bash
make release-check STRICT_RELEASE=1
```

### Local release dry-run flow

```bash
make snapshot
make test-release
```

[Back to top](#gghstats)

## Configuration

All runtime configuration uses env vars (`serve`) or flags (`fetch/report/export`).

### Environment file

- **Template:** [`.env.example`](.env.example) — copy to `.env` and fill in secrets. `.env` is gitignored (dotfiles are excluded by default in this repo).
- **Compose:** `docker compose` loads `.env` from the project directory automatically.

### Environment variables (serve)

| Variable | Default | Description |
| --- | --- | --- |
| `GGHSTATS_GITHUB_TOKEN` | (required) | GitHub personal access token |
| `GGHSTATS_DB` | `./data/gghstats.db` | SQLite database path |
| `GGHSTATS_HOST` | `0.0.0.0` | Bind address |
| `GGHSTATS_PORT` | `8080` | Listen port |
| `GGHSTATS_FILTER` | `*` | Repo filter expression |
| `GGHSTATS_INCLUDE_PRIVATE` | `false` | Include private repos |
| `GGHSTATS_SYNC_INTERVAL` | `1h` | Sync frequency |
| `GGHSTATS_API_TOKEN` | (none) | If set, `GET /api/repos` requires matching `x-api-token` header (see [HTTP API (JSON)](#http-api-json)) |
| `GGHSTATS_BADGE_PUBLIC` | `true` | Set to `false` to require `x-api-token` on badge URLs (breaks `![…](url)` in GitHub READMEs unless you use a proxy) |
| `GGHSTATS_BADGE_CACHE_SECONDS` | `300` | `Cache-Control: max-age` for badge SVG responses |
| `GGHSTATS_PUBLIC_URL` | (none) | Optional public base URL for embed snippets (e.g. `https://gghstats.example.com`); if unset, uses the request `Host` |
| `GGHSTATS_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` (slog only; startup banner always prints) |
| `GGHSTATS_METRICS` | (enabled) | Set to `false` to disable `GET /metrics` |
| `GGHSTATS_CUSTOM_CSS` | (none) | Optional **regular** `.css` file: loaded **after** built-in `app.css` at `/theme/custom.css` so you can tone down neo-brutalism or replace accents (see [Custom UI theme](#custom-ui-theme-optional)) |

### Custom UI theme (optional)

The shipped look is **neo-brutalist** on purpose—not every user or org wants heavy borders and loud chrome. If you prefer something **flatter, calmer, or closer to your brand**, you can supply your own CSS and keep the same binary and data layout.

Self-hosted installs can override colors and chrome **without rebuilding**:

1. Copy one of the **five official example themes** from [`contrib/themes/`](contrib/themes/README.md) (for a stock-Bootstrap feel use **`example-bootstrap-plain.css`**; or write your own CSS targeting `body.app-brutalist` and `html[data-bs-theme="dark"] body.app-brutalist`).
2. Place the file where the process can read it (e.g. mount `./data/custom-theme.css` in Docker to `/data/custom-theme.css`).
3. Set **`GGHSTATS_CUSTOM_CSS=/data/custom-theme.css`** (absolute or relative path; relative paths resolve from the process working directory).
4. Restart **`gghstats serve`**. The layout adds `<link href="/theme/custom.css?…">` after `/static/app.css`. The query string bumps when the file’s modification time changes.

**Bootstrap-plain example** ([`example-bootstrap-plain.css`](contrib/themes/example-bootstrap-plain.css) with `GGHSTATS_CUSTOM_CSS`): repository index in light mode — closer to stock Bootstrap (sans-serif, thin borders, no offset shadows):

![gghstats dashboard with Bootstrap-plain optional theme](assets/gghstats-main-theme-bootstrap-plain.png)

If the variable is set but the path is not a readable regular file, startup logs a **warning** and the UI stays default (no extra link).

### Token setup

Create a **GitHub personal access token** the app will use for [`/user/repos`](https://docs.github.com/en/rest/repos/repos#list-repositories-for-the-authenticated-user) and [repository traffic](https://docs.github.com/en/rest/metrics/traffic) (views, clones, referrers, paths) plus stars and related metadata.

1. Go to **[GitHub → Settings → Developer settings → Personal access tokens](https://github.com/settings/tokens)** (classic or fine-grained, see below).
2. Create the token and store it only in env / secret storage (`GGHSTATS_GITHUB_TOKEN`).

#### Classic tokens (“Generate new token (classic)”)

| Scope | When to use it |
| --- | --- |
| **`repo`** | **Recommended default** if you sync **private** repositories (`GGHSTATS_INCLUDE_PRIVATE=true`) or you hit **403** on traffic endpoints. Full `repo` covers private repos, traffic, and listing for repos your account can access (subject to GitHub’s own rules). |
| **`public_repo`** | Only **public** repositories and `GGHSTATS_INCLUDE_PRIVATE` is **not** `true`. Narrow with `GGHSTATS_FILTER` if needed. Traffic APIs require **push/admin** on each repo; for repos **you own**, this scope is often enough for public traffic. If traffic calls fail with **403**, switch to **`repo`**. |

Optional: **`read:org`** if you rely on organization membership to see org repos not returned by default (uncommon for a personal token on your own org).

#### Fine-grained tokens

Create at **[Fine-grained tokens](https://github.com/settings/personal-access-tokens)**. Pick the **resource owner** (user or org), then either **only selected repositories** or **all** this token may access. Grant **read-only** (or higher) permissions that allow:

- Listing and reading those repositories (metadata / contents as required by GitHub for your setup).
- Access to **traffic** metrics for each repo (GitHub’s permission names change over time; if sync logs show **403** on `/traffic/*`, widen repository permissions or use a **classic** token with **`repo`** for that account).

Fine-grained tokens **cannot** be mixed with classic scope names; follow GitHub’s UI for the minimum set that allows traffic reads on your repos.

#### References

- GitHub: [Authenticating to the REST API](https://docs.github.com/en/rest/authentication/authenticating-to-the-rest-api), [Repository traffic](https://docs.github.com/en/rest/metrics/traffic).

### Filter examples

Replace `your-github-user` with your GitHub username or organization, and `my-app` / `other-repo` / `legacy-repo` with your real repository names.

```bash
GGHSTATS_FILTER="your-github-user/*"
GGHSTATS_FILTER="your-github-user/my-app,your-github-user/other-repo"
GGHSTATS_FILTER="*,!fork"
GGHSTATS_FILTER="*,!archived"
GGHSTATS_FILTER="your-github-user/*,!fork,!archived"
GGHSTATS_FILTER="*,!your-github-user/legacy-repo"
```

### HTTP API (JSON)

gghstats exposes a **small read-only JSON surface** for probes and integrations. There is **no** generic REST CRUD layer; everything else is the HTML UI or the CLI.

#### `GET /api/v1/healthz`

| | |
| --- | --- |
| **Purpose** | Liveness / readiness style probe (same path string as many Kubernetes configs). |
| **Auth** | None — **public**. |
| **Response** | **`200`** with body `{"status":"ok"}` and `Content-Type: application/json`. |

```bash
curl -sS http://localhost:8080/api/v1/healthz
# {"status":"ok"}
```

#### `GET /api/v1/badge/{owner}/{repo}`

| | |
| --- | --- |
| **Purpose** | shields.io-style **SVG badge** for embedding in a repository README (`![label](url)`). |
| **Auth** | **Public by default** (`GGHSTATS_BADGE_PUBLIC` unset or not `false`). Set `GGHSTATS_BADGE_PUBLIC=false` to require the same `x-api-token` as `/api/repos` (not usable from GitHub image embeds without a proxy). |
| **Response** | **`200`** `image/svg+xml` with `Cache-Control: public, max-age=…` (default 300s). |
| **Alias** | Same handler for `…/repo.svg`. |

**Query parameters:**

| Parameter | Values | Default |
| --- | --- | --- |
| `metric` | `clones`, `clones_30d`, `views`, `stars` | `clones` |
| `style` | `flat`, `flat-square` | `flat` |
| `label` | Custom left label (URL-encoded) | Metric name (`clones`, `clones 30d`, …) |

**Semantics** match the web UI / `GET /api/repos`: `clones` and `views` are lifetime sums in SQLite; `clones_30d` is the rolling 30-day UTC window; `stars` is the latest synced metadata value.

```bash
curl -sS 'http://localhost:8080/api/v1/badge/your-user/your-repo?metric=clones' -o /tmp/badge.svg
```

```markdown
![gghstats clones](https://gghstats.example.com/api/v1/badge/your-user/your-repo?metric=clones)
```

On each repository page, the **Embed badge** card builds this Markdown (metric selector + copy button). Optional **`GGHSTATS_PUBLIC_URL`** sets the host in snippets when the app sits behind a reverse proxy.

#### `GET /api/v1/repos/{owner}/{repo}/traffic`

| | |
| --- | --- |
| **Purpose** | Daily **clone** and **view** time series for one repository (for Grafana, scripts, or external charts). |
| **Auth** | Same as **`GET /api/repos`**: requires **`GGHSTATS_API_TOKEN`** and header **`x-api-token`**. Returns **`404`** when the API is disabled (token unset). |
| **CORS** | **`Access-Control-Allow-Origin: *`** on success. |

**Query parameters:**

| Parameter | Meaning |
| --- | --- |
| `days` | Rolling window in **UTC calendar days**, inclusive of today. Default **`30`**. Use **`0`** for all dates stored in SQLite for this repo. Maximum **`3660`**. |

**Response (`200`):**

| Field | Type | Meaning |
| --- | --- | --- |
| `name` | string | `owner/repo` |
| `days` | number | Echo of the `days` query (after defaulting). |
| `from`, `to` | string | `YYYY-MM-DD` bounds used for the query (inclusive). |
| `clones` | array | Daily clone rows: `date`, `count`, `uniques` (GitHub traffic semantics). |
| `views` | array | Daily view rows: same shape. |

Missing days in the window are omitted (not zero-filled). This matches the repo detail charts, which only plot days with rows in the database.

```bash
curl -sS -H "x-api-token: $GGHSTATS_API_TOKEN" \
  'http://localhost:8080/api/v1/repos/your-user/your-repo/traffic?days=30'
```

#### `POST /api/v1/sync` and `GET /api/v1/sync`

| | |
| --- | --- |
| **Purpose** | **Manual sync** with GitHub (same job as the scheduler): list repos, refresh metadata, pull traffic. |
| **Auth** | Same as **`GET /api/repos`**: **`GGHSTATS_API_TOKEN`** + **`x-api-token`**. Returns **`404`** when the API is disabled. |
| **Concurrency** | Only one sync runs at a time. **`POST`** while a run is active returns **`409`** with `sync_in_progress`. The scheduler skips its tick if a manual sync is running. |

**`POST /api/v1/sync`** — starts a background **full** sync (all repos matching `GGHSTATS_FILTER`); responds **`202 Accepted`** with `{"status":"started","scope":"all"}` when accepted.

**`POST /api/v1/sync?repo=owner/name`** — syncs **only that repository** (fast; does not wait for the full list). Response includes `"scope":"repo"` and `"repo":"owner/name"`.

**`GET /api/v1/sync`** — status JSON:

| Field | Meaning |
| --- | --- |
| `running` | `true` while a sync is in progress |
| `scope` | `all` or `repo` while running |
| `repo` | `owner/name` when `scope` is `repo` |
| `last_started_at`, `last_finished_at` | RFC3339 timestamps (UTC) of the last run |
| `last_error` | Non-empty if the last run failed |

```bash
# Sync all repos (respects GGHSTATS_FILTER)
curl -sS -X POST -H "x-api-token: $GGHSTATS_API_TOKEN" \
  http://localhost:8080/api/v1/sync

# Sync one repo only
curl -sS -X POST -H "x-api-token: $GGHSTATS_API_TOKEN" \
  'http://localhost:8080/api/v1/sync?repo=your-user/your-repo'

# Poll status
curl -sS -H "x-api-token: $GGHSTATS_API_TOKEN" \
  http://localhost:8080/api/v1/sync
```

When **`GGHSTATS_API_TOKEN`** is set, the sidebar shows **Sync all** on the index and **Sync this repo** on a repository page. The first click opens a modal to enter the token; it is stored in **`sessionStorage`** (same origin only). After a successful single-repo sync, the repo page reloads to refresh charts.

#### `GET /api/repos`

| | |
| --- | --- |
| **Purpose** | Snapshot of all **non-hidden** repositories in the local SQLite DB with aggregate counters. |
| **Auth** | **Required** when `GGHSTATS_API_TOKEN` is set: send header **`x-api-token: <value>`** matching that env var exactly. If `GGHSTATS_API_TOKEN` is **unset**, requests to this path return **`404 Not Found`** (API disabled by default). |
| **CORS** | Successful responses include **`Access-Control-Allow-Origin: *`** so browser dashboards on another origin can read the JSON (you still must keep the API token secret). |
| **Sort order** | Items are always returned in **`total_views` descending** (see `handleAPIRepos` in the server code). This is **independent** of the web index `sort=` query parameter. |
| **Errors** | **`401`** with JSON `{"error":"unauthorized"}` if the token header is missing or wrong. **`500`** with JSON `{"error":"…"}` on database or encoding failures. |

**Response shape (`200`):**

| Field | Type | Meaning |
| --- | --- | --- |
| `total_count` | number | Count of repos in `items`. |
| `total_stars` | number | Sum of `stars` across repos. |
| `total_forks` | number | Sum of `forks` across repos. |
| `total_views` | number | Sum of `total_views` across repos. |
| `total_clones` | number | Sum of `total_clones` across repos. |
| `items` | array | One object per repository (see table below). |

**Each element of `items`** matches [`RepoSummary`](internal/store/store.go) JSON tags:

| Field | Type | Notes |
| --- | --- | --- |
| `name` | string | `owner/repo` |
| `description` | string | |
| `stars`, `forks`, `watchers`, `issues`, `prs` | number | From last GitHub metadata sync. |
| `fork` | boolean | |
| `parent_full_name` | string | Upstream if fork (may be empty / omitted). |
| `archived` | boolean | |
| `total_views`, `total_uniques` | number | Lifetime sums of daily GitHub **view** traffic stored in SQLite. |
| `total_clones`, `clone_uniques` | number | Lifetime sums of daily **clone** traffic. |
| `clones_1d` | number | Sum of daily clone counts for **today (UTC)**; missing days count as `0`. |
| `clones_7d` | number | Sum of daily clone counts in the **last 7 calendar days (UTC)**; missing days count as `0`. |
| `clones_30d` | number | Sum of daily clone counts in the **last 30 calendar days (UTC)**; missing days count as `0`. |

**Example request:**

```bash
curl -sS -H "x-api-token: $GGHSTATS_API_TOKEN" http://localhost:8080/api/repos
```

**Example response (truncated to one repo):**

```json
{
  "total_count": 1,
  "total_stars": 10,
  "total_forks": 2,
  "total_views": 150,
  "total_clones": 42,
  "items": [
    {
      "name": "your-github-user/my-app",
      "description": "Example",
      "stars": 10,
      "forks": 2,
      "watchers": 3,
      "issues": 1,
      "prs": 0,
      "fork": false,
      "archived": false,
      "total_views": 150,
      "total_uniques": 80,
      "total_clones": 42,
      "clone_uniques": 12,
      "clones_1d": 2,
      "clones_7d": 5,
      "clones_30d": 7
    }
  ]
}
```

#### `GET /metrics`

| | |
| --- | --- |
| **Purpose** | [Prometheus](https://prometheus.io/) text / OpenMetrics exposition for scraping. |
| **Auth** | None — treat network access like any other unauthenticated metrics endpoint. |
| **Disabled** | When `GGHSTATS_METRICS=false`, the route is omitted (returns **`404`**). |

See [Security and quality](#security-and-quality) for the local tooling that scans this surface in CI.

[Back to top](#gghstats)

## Typical scenarios

### Track all repositories for one owner

```bash
export GGHSTATS_FILTER="your-github-user/*"
gghstats serve
```

### Exclude forks and archived repositories

```bash
export GGHSTATS_FILTER="your-github-user/*,!fork,!archived"
gghstats serve
```

### Protect API with token

Full field list, error codes, and probe endpoint are documented under **[HTTP API (JSON)](#http-api-json)**.

```bash
export GGHSTATS_API_TOKEN="my-api-token"
gghstats serve
curl -H "x-api-token: my-api-token" http://localhost:8080/api/repos
```

### Generate periodic CSV report

```bash
gghstats export --repo your-github-user/my-app --days 30 --output traffic-30d.csv
```

[Back to top](#gghstats)

## Deployments

Production and optional observability (Traefik + TLS, Prometheus / Grafana stack, Helm) live in a separate repository so release versioning applies to the **application** only. For self-hosted setups, start here:

**[github.com/hrodrig/gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**

Clone that repo on your server, copy `.env.example` → `.env`, and follow its README for the deployment path you choose. For the optional metrics/logs stack, see **[run/docker-compose/observability/README.md](https://github.com/hrodrig/gghstats-selfhosted/blob/main/run/docker-compose/observability/README.md)** (on the default branch).

[Back to top](#gghstats)

## Troubleshooting

### `GGHSTATS_GITHUB_TOKEN is required`

Set `GGHSTATS_GITHUB_TOKEN` in your shell or `.env` file before running `serve`.

### Dashboard shows no repositories

- Wait for the initial sync to finish.
- Verify filter rules (`GGHSTATS_FILTER`) are not excluding all repos.
- Confirm [token scopes](#token-setup) allow listing repos and reading **traffic** (see **403** note there).

### Port `8080` already in use

Set another listen port via env or flag:

```bash
export GGHSTATS_PORT=9090
gghstats serve
# or: gghstats serve --port 9090
```

### API returns `401 unauthorized`

Confirm request header exactly matches configured token. For `404` on `/api/repos`, the API is disabled until you set `GGHSTATS_API_TOKEN` (see [HTTP API (JSON)](#http-api-json)).

```bash
curl -H "x-api-token: $GGHSTATS_API_TOKEN" http://localhost:8080/api/repos
```

[Back to top](#gghstats)

## Release workflow

- Branch policy: day-to-day development on `develop`; **tagged releases** are cut from **`main`**.
- **`VERSION`** file: semantic version **without** `v` (for example `0.2.1`). Must match the static **Version** badge at the top of this README.
- **Git tags:** annotated tag **with** `v` prefix (for example `v0.2.1`), on the commit you want released.

### Default: publish from GitHub Actions (no local GoReleaser required)

Pushing a tag matching `v*` runs [`.github/workflows/release.yml`](.github/workflows/release.yml): `make release-check`, then `goreleaser release --clean` with `GITHUB_TOKEN` (releases + **GHCR**).

```bash
# 1) On develop: land changes, bump version if needed
git checkout develop
make release-check                    # optional: STRICT_RELEASE=1 (adds docker image scan)
make test-release                     # optional: dry-run GoReleaser + local Docker build

# 2) Update VERSION, README version badge, CHANGELOG; commit on develop

# 3) Merge into main (PR or fast-forward), then tag and push
git checkout main && git pull origin main
git merge --ff-only develop           # or: merge via GitHub PR
git push origin main

git tag -a v0.2.1 -m "Release 0.2.1"
git push origin v0.2.1                # triggers Release workflow — builds and publishes artifacts
```

For the **next** release after `0.2.1`, set `VERSION` to `0.2.2` (etc.), update the badge and [CHANGELOG](CHANGELOG.md), then repeat with `v0.2.2`.

### Optional: publish from your machine

If you run GoReleaser locally instead of relying on CI, checkout **`main`** at the tagged commit, export **`GITHUB_TOKEN`** (or **`GH_TOKEN`**) with `repo` and **packages** access to push GHCR, then:

```bash
make release                          # runs release-check then goreleaser release --clean
```

### Developer checklist

- Update **`CHANGELOG.md`** (move `[Unreleased]` into the new version section).
- Keep **`VERSION`** (no `v`), README **Version** badge, and [CHANGELOG](CHANGELOG.md) in sync; the OCI tag uses the same `v` prefix as the Git tag. Deployment image pins live in **gghstats-selfhosted**.
- Ensure **CI** and **Security** workflows are green before pushing the release tag.
- **Docker:** `Dockerfile` is for local `make docker-build` / `docker-scan`. **GoReleaser** uses **`Dockerfile.release`** (pre-built Linux binaries; same pattern as multi-arch release images).

[Back to top](#gghstats)

## Security and quality

```bash
make tools
make lint
make test
make security
make release-check
```

Security tooling:

- `govulncheck`
- `gocyclo` (complexity gate)
- `grype` (filesystem image/source scanning)

[Back to top](#gghstats)

## Database

SQLite path comes from `GGHSTATS_DB`. Main tables: `repos`, `views`, `clones`, `referrers`, `paths`, `stars`.

- Upserts are idempotent
- Startup migration uses `PRAGMA user_version`

[Back to top](#gghstats)

## Community standards

- License: `LICENSE`
- Contributing: `CONTRIBUTING.md`
- Code of conduct: `CODE_OF_CONDUCT.md`
- Security policy: `SECURITY.md`
- Changelog: `CHANGELOG.md`
- CODEOWNERS: `.github/CODEOWNERS`

Thanks for using and contributing to `gghstats`.

[Back to top](#gghstats)

## Acknowledgments

Hats off to **[ghstats](https://github.com/vladkens/ghstats)** by [vladkens](https://github.com/vladkens): a self-hosted GitHub traffic dashboard in **Rust** that also keeps historical traffic beyond GitHub’s short default window, with SQLite and a small deployment story. `gghstats` is a separate **Go** implementation and design, but that project deserves credit as important prior work in the same problem space.

Thanks also to **[git-clone-stats](https://github.com/taylorwilsdon/git-clone-stats)** by [taylorwilsdon](https://github.com/taylorwilsdon): a self-hosted GitHub clone and traffic analytics stack in **Python** with SQLite (or Firestore), a minimal HTML/JS dashboard, and **shields.io-style badges** for README embeds. The badge endpoint and “copy Markdown” embed flow in `gghstats` follow a similar idea; this project is independent Go code, not a port.

[Back to top](#gghstats)

## License

MIT
