# gghstats

![gghstats — self-hosted GitHub traffic beyond the 14-day window](assets/gghstats-poster-devto.png)

[![Version](https://img.shields.io/badge/version-0.7.2-blue)](https://github.com/hrodrig/gghstats/releases)
[![Release](https://img.shields.io/github/v/release/hrodrig/gghstats)](https://github.com/hrodrig/gghstats/releases)
[![CI](https://github.com/hrodrig/gghstats/actions/workflows/ci.yml/badge.svg)](https://github.com/hrodrig/gghstats/actions)
[![codecov](https://codecov.io/gh/hrodrig/gghstats/graph/badge.svg)](https://codecov.io/gh/hrodrig/gghstats)
[![gghstats clones](https://gghstats.hermesrodriguez.com/api/v1/badge/hrodrig/gghstats?metric=clones)](https://gghstats.hermesrodriguez.com/hrodrig/gghstats)
[![Go 1.26.3](https://img.shields.io/badge/go-1.26.3-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/hrodrig/gghstats)](https://pkg.go.dev/github.com/hrodrig/gghstats)
[![Go Report Card](https://goreportcard.com/badge/github.com/hrodrig/gghstats)](https://goreportcard.com/report/github.com/hrodrig/gghstats)
[![deps.dev](https://img.shields.io/badge/deps.dev-go%20module-blue)](https://deps.dev/go/github.com%2Fhrodrig%2Fgghstats)

**Repo:** [github.com/hrodrig/gghstats](https://github.com/hrodrig/gghstats) · **Releases:** [Releases](https://github.com/hrodrig/gghstats/releases)

Self-hosted dashboard and CLI for GitHub repository traffic stats. GitHub only keeps traffic for 14 days; `gghstats` keeps historical data indefinitely in SQLite.

If you want your **own self-hosted** deployment (Docker Compose, Traefik with TLS, Helm, optional Prometheus/Grafana/Loki), use the companion repo **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** — it lists the supported options and example manifests.

**Related (same maintainer):** **[pgwd](https://github.com/hrodrig/pgwd)** — PostgreSQL connection watchdog (Slack/Loki alerts); production manifests in **[pgwd-selfhosted](https://github.com/hrodrig/pgwd-selfhosted)**.

**Releases:** [GitHub Releases](https://github.com/hrodrig/gghstats/releases) ship binaries (tarballs/zip + checksums). **Multi-arch** container images (`linux/amd64`, `linux/arm64`) are on [GHCR](https://github.com/hrodrig/gghstats/pkgs/container/gghstats) as `ghcr.io/hrodrig/gghstats:v<version>` (same `v` prefix as the Git tag, e.g. `v0.6.0`) and `:latest`. Pushing a `v*` tag on `main` triggers the [Release workflow](.github/workflows/release.yml) (GoReleaser). Day-to-day work happens on `develop` (see [Release workflow](#release-workflow)).

## Demo

**Live:** [gghstats.hermesrodriguez.com](https://gghstats.hermesrodriguez.com)

### Beyond GitHub’s 14-day Traffic tab

GitHub **Insights → Traffic** shows clones and views for a **rolling 14 days** only. Older daily data disappears from the UI unless you archive it yourself. **gghstats** syncs the same GitHub API into **SQLite** so you keep daily charts and totals for as long as you run it.

Same repository ([`hrodrig/gghstats`](https://github.com/hrodrig/gghstats)):

![GitHub Traffic — last 14 days only](assets/compare-github-traffic-14d.png)

*GitHub Insights → Traffic — rolling 14-day window (May 2026).*

![gghstats — daily history in SQLite](assets/compare-gghstats-history.png)

*Same repo on the [live demo](https://gghstats.hermesrodriguez.com/hrodrig/gghstats) — daily clones/views from first sync (Mar–May 2026). Totals differ from GitHub’s 14d rollup; see [Repository page charts](#repository-page-charts-clones--views).*

![GGHSTATS dashboard — repository metrics and neobrutalist UI](assets/gghstats-main.png)

## Table of contents

- [Demo](#demo)
  - [Beyond GitHub’s 14-day Traffic tab](#beyond-githubs-14-day-traffic-tab)
- [Features](#features)
- [Repository page charts](#repository-page-charts-clones--views)
- [Quick start](#quick-start)
- [Install](#install)
- [Build](#build)
- [Web UI assets (developers)](#web-ui-assets-developers)
- [Usage](#usage)
- [Examples](#examples)
- [Configuration](#configuration)
- [Web UI languages (i18n)](#web-ui-languages-i18n)
- [Environment file](#environment-file)
- [Custom UI theme (optional)](#custom-ui-theme-optional)
- [HTTP API (JSON)](#http-api-json)
- [Typical scenarios](#typical-scenarios)
- [Troubleshooting](#troubleshooting)
- [Release workflow](#release-workflow)
- [Security and quality](#security-and-quality)
- [Database](#database)
- [Community standards](#community-standards)
- [Star History](#star-history)
- [Acknowledgments](#acknowledgments)
- [License](#license)

## Features

- Collects views, clones, referrers, popular paths, and star history
- Auto-discovers repositories (or filters by org/repo rules)
- Web dashboard with Chart.js graphs
- **Web UI languages (i18n):** English (default), Spanish, German, French, and Brazilian Portuguese — sidebar **EN | ES | DE | FR | PT**, cookie `gghstats_locale`, env defaults (see [Web UI languages](#web-ui-languages-i18n))
- **Head to Head (H2H)** at `/h2h` — compare two repos with weighted share scores (0–100, sum to 100); open *How the H2H score is calculated* on that page for the formula
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

Exact colors depend on light/dark theme (Bootstrap `--bs-primary` / `--bs-info`, overridden in the app’s neo-brutalist CSS). Chart **legends** (e.g. Unique / Count) follow the active UI locale. Use the tooltip on each bar for values.

[Back to top](#gghstats)

## Quick start

The steps below are a **minimal path** to install gghstats, run the dashboard once, and **try the UI** (repository list, charts, H2H, languages). They are **not** a production or server deployment.

**Running gghstats on a VPS, with TLS, Traefik, Compose stacks, Helm, or observability** — use **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**. That repository owns production manifests, env layout (`GGHSTATS_HOST_DATA`), and operator docs; this repo ships the application binary and image only.

**gghstats has many configuration options** (repo filter, sync behavior, API token, locales, bind address, SQLite path, badges, metrics, optional theme, and more). Before running it for real data, read **[Configuration](#configuration)**, **[Environment file](#environment-file)**, and **[Token setup](#token-setup)** in detail. Use **[Install](#install)** for `.deb`, `.rpm`, or tarballs; use **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** when you outgrow these smoke-test commands.

### Homebrew (macOS / Linux)

```bash
brew install hrodrig/gghstats/gghstats
export GGHSTATS_GITHUB_TOKEN=ghp_xxx
gghstats run --open
```

**`run`** is an alias for **`serve`**; **`--open`** opens the default browser when the dashboard is ready. Background daemon on macOS: **[contrib/launchd/README.md](contrib/launchd/README.md)**. Linux packages install **[systemd](contrib/systemd/README.md)**.

Tap: [homebrew-gghstats](https://github.com/hrodrig/homebrew-gghstats). The cask is updated when a new [release](https://github.com/hrodrig/gghstats/releases) is published. Server deployment: **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**.

### Local binary (fastest try)

```bash
# From Releases: extract gghstats_*_linux_* archive, or: go install github.com/hrodrig/gghstats/cmd/gghstats@latest
export GGHSTATS_GITHUB_TOKEN=ghp_xxx
gghstats run --open
```

Open <http://localhost:8080> if you did not use **`--open`**. Data is stored in `./data/gghstats.db` (override with `GGHSTATS_DB`). A first sync may take a while if the default filter includes many repositories — narrow `GGHSTATS_FILTER` in [Configuration](#configuration) when you move beyond this smoke test.

### Docker (one command, no clone — UI smoke test)

Quick validation only. **Production Docker / Compose / Traefik / Helm:** **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**.

```bash
docker run --rm -d \
  -e GGHSTATS_GITHUB_TOKEN=ghp_xxx \
  -e GGHSTATS_FILTER="your-github-user/*" \
  -p 8080:8080 \
  -v gghstats-data:/data \
  --name gghstats \
  ghcr.io/hrodrig/gghstats:latest
```

Open <http://localhost:8080> to validate the interface. Named volume keeps SQLite history between restarts.

### Docker Compose (local development in this repo)

Builds the image from source — **not** the supported production path. For published-image Compose, Traefik + TLS, minimal/observability stacks, and **`GGHSTATS_HOST_DATA`**, use **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**.

```bash
cp .env.example .env
# Edit .env: set GGHSTATS_GITHUB_TOKEN (and optionally GGHSTATS_FILTER, GGHSTATS_PORT, etc.)
docker compose up -d --build
```

Open <http://localhost:8080>. The template [`.env.example`](.env.example) documents variables for the **Go binary** and this **dev-only** Compose file.

[Back to top](#gghstats)

## Install

**Quick install** — get the binary on your machine. **Configuration on a server, systemd, `.deb`/`.rpm` setup, Compose, Traefik, Helm, env files, and VPS deployment** are documented only in **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** ([`run/`](https://github.com/hrodrig/gghstats-selfhosted/tree/main/run), [`run/standalone/linux/`](https://github.com/hrodrig/gghstats-selfhosted/blob/main/run/standalone/linux/README.md) for Linux packages and systemd).

**From source (recommended for developers):**

```bash
go install github.com/hrodrig/gghstats/cmd/gghstats@latest
```

This installs the binary to `$GOBIN` (default `$HOME/go/bin`). Ensure `$GOBIN` is on your `PATH`.

**Package managers and images:**

| Platform | Command / path |
|----------|----------------|
| **Homebrew (macOS / Linux)** | `brew install hrodrig/gghstats/gghstats` — [tap](https://github.com/hrodrig/homebrew-gghstats) |
| **Debian / Ubuntu** | `wget -q -O /tmp/gghstats.deb https://github.com/hrodrig/gghstats/releases/download/v0.7.2/gghstats_0.7.2_linux_amd64.deb && sudo dpkg -i /tmp/gghstats.deb` |
| **Fedora / RHEL / AlmaLinux / Rocky / Oracle Linux** | `sudo dnf install https://github.com/hrodrig/gghstats/releases/download/v0.7.2/gghstats_0.7.2_linux_amd64.rpm` |
| **Linux tarball** | `tar -xzf gghstats_*_linux_*.tar.gz` from [Releases](https://github.com/hrodrig/gghstats/releases); verify `checksums.txt` |
| **FreeBSD** | `gghstats_*_freebsd_*.tar.gz` on [Releases](https://github.com/hrodrig/gghstats/releases); port + developer guide [`contrib/freebsd/README.md`](contrib/freebsd/README.md) (`gmake` in repo, `make` in ports tree) |
| **OpenBSD** | `gghstats_*_openbsd_*.tar.gz` on [Releases](https://github.com/hrodrig/gghstats/releases); [`contrib/openbsd/README.md`](contrib/openbsd/README.md) and port [`contrib/openbsd/port/`](contrib/openbsd/port/) |
| **macOS / Windows archives** | `.tar.gz` / `.zip` on [Releases](https://github.com/hrodrig/gghstats/releases) |
| **OCI image** | `ghcr.io/hrodrig/gghstats:v0.7.2` or `:latest` (multi-arch) |

Replace `v0.7.2` and `amd64` with your [release](https://github.com/hrodrig/gghstats/releases) version and architecture (e.g. `arm64`).

After install, validate the UI locally:

```bash
export GGHSTATS_GITHUB_TOKEN=ghp_xxx
gghstats serve
```

Open <http://127.0.0.1:8080>. Variable semantics: [Configuration](#configuration) below. **Do not use this README for production deployment** — use **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**.

### Always-on (macOS)

To keep gghstats running after you log in (background sync + dashboard on **localhost**), use a **LaunchAgent** — there is **no** `brew services` stanza in the Homebrew cask yet.

1. Install the binary (`brew install hrodrig/gghstats/gghstats` or a [Release](https://github.com/hrodrig/gghstats/releases) archive).
2. Follow **[contrib/launchd/README.md](contrib/launchd/README.md)**: `~/.gghstats.env`, `gghstats-serve.sh` wrapper, plist in `~/Library/LaunchAgents/`, then `launchctl bootstrap`.

**Linux servers:** `.deb`/`.rpm` install a **systemd** unit — [`contrib/systemd/README.md`](contrib/systemd/README.md) and **`systemctl enable --now gghstats`**. Public TLS / Compose / Helm → **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**.

[Back to top](#gghstats)

## Build

```bash
git clone https://github.com/hrodrig/gghstats.git
cd gghstats
make build
make install          # installs to $HOME/go/bin by default; GOBIN=~/bin make install for another path
make install-man      # optional: MANDIR=/usr/share/man for system-wide man page
```

**Release (GitHub):** merge `develop` → `main`, tag `v*`, push — the [Release workflow](.github/workflows/release.yml) runs GoReleaser. Locally: `make release-check` then `make release` (requires [goreleaser](https://goreleaser.com) and tokens). Snapshot without publishing: `make snapshot` → `dist/`.

[Back to top](#gghstats)

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

### Server mode (recommended for local try-out)

```bash
export GGHSTATS_GITHUB_TOKEN="ghp_your_token"
gghstats serve
```

On a **VPS or production host**, run the published image via **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** instead of ad-hoc `gghstats serve` on the public internet.

Server behavior:

- Runs initial sync when database is empty
- Re-syncs on schedule (default `1h`)
- Serves dashboard on <http://localhost:8080>
- Stores data in `./data/gghstats.db`
- Liveness/readiness: `GET /api/v1/healthz` → `{"status":"ok"}` (no auth; Kubernetes-style)
- Prometheus: `GET /metrics` (disable with `GGHSTATS_METRICS=false`)
- Listen port: `GGHSTATS_PORT` (default `8080`) or `gghstats serve --port <port>` (or `gghstats run --open` for local try)
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

### Run pre-release checks (includes container scan)

```bash
make release-check
```

### Local release dry-run flow

Snapshot and test-release versions come from the repo **`VERSION`** file (for example `0.5.0` → artifacts `0.5.0-next`), not from the latest git tag.

```bash
make snapshot        # GoReleaser snapshot → dist/ (no Docker; no publish)
make test-release    # same version source; --skip=publish; still no Docker on snapshot
```

On a real release, push tag **`v<VERSION>`** (must match **`VERSION`**) so GoReleaser and CI use that semver.

[Back to top](#gghstats)

## Configuration

**Application reference** — what each `GGHSTATS_*` variable means (for `gghstats serve`, CLI, and API behavior).

**Operator implementation** (where to put env files, Compose, Traefik, Helm, systemd, `/etc/gghstats/gghstats.env`, `GGHSTATS_HOST_DATA`, upgrades, observability) → **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** only:

- [`run/common/.env.example`](https://github.com/hrodrig/gghstats-selfhosted/blob/main/run/common/.env.example) — production Compose env template  
- [`run/standalone/linux/README.md`](https://github.com/hrodrig/gghstats-selfhosted/blob/main/run/standalone/linux/README.md) — `.deb`/`.rpm`, systemd, bare-metal paths  
- [`run/docker-compose/`](https://github.com/hrodrig/gghstats-selfhosted/tree/main/run/docker-compose/) — minimal, Traefik, observability  
- [`run/kubernetes/`](https://github.com/hrodrig/gghstats-selfhosted/tree/main/run/kubernetes/) — Helm and manifests  

This repo’s [`.env.example`](.env.example) is for **local smoke tests** and **dev** `docker-compose.yml` in the gghstats clone only.

### Environment file (local try-out)

Copy [`.env.example`](.env.example) → `.env` in this repository when running `gghstats serve` on your laptop. **Servers:** use **gghstats-selfhosted**, not this file.

### Environment variables (serve)

| Variable | Default | Description |
| --- | --- | --- |
| `GGHSTATS_GITHUB_TOKEN` | (required) | GitHub personal access token |
| `GGHSTATS_DB` | `./data/gghstats.db` | SQLite database path |
| `GGHSTATS_HOST` | `127.0.0.1` | Bind address (localhost only on bare metal). **Production Compose** sets `0.0.0.0` in **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** |
| `GGHSTATS_PORT` | `8080` | Listen port |
| `GGHSTATS_FILTER` | `*` | Repo filter expression |
| `GGHSTATS_INCLUDE_PRIVATE` | `false` | Include private repos |
| `GGHSTATS_SYNC_INTERVAL` | `1h` | Sync frequency |
| `GGHSTATS_SYNC_ON_STARTUP` | `true` | Full sync when the process starts; set `false` to serve immediately using existing SQLite data |
| `GGHSTATS_OPEN_BROWSER` | `false` | Open the default browser when the server is ready (same as `gghstats serve --open`) |
| `GGHSTATS_API_TOKEN` | (none) | If set, `GET /api/repos` requires matching `x-api-token` header (see [HTTP API (JSON)](#http-api-json)) |
| `GGHSTATS_BADGE_PUBLIC` | `true` | Set to `false` to require `x-api-token` on badge URLs (breaks `![…](url)` in GitHub READMEs unless you use a proxy) |
| `GGHSTATS_BADGE_CACHE_SECONDS` | `300` | `Cache-Control: max-age` for badge SVG responses |
| `GGHSTATS_PUBLIC_URL` | (none) | Optional public base URL for embed snippets (e.g. `https://gghstats.example.com`); if unset, uses the request `Host` |
| `GGHSTATS_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` (slog only; startup banner always prints) |
| `GGHSTATS_METRICS` | (enabled) | Set to `false` to disable `GET /metrics` |
| `GGHSTATS_METRICS_PER_REPO` | `false` | Set to `true` to expose per-repo Prometheus gauges (`owner`, `repo` labels); higher cardinality |
| `GGHSTATS_CUSTOM_CSS` | (none) | Optional **regular** `.css` file: loaded **after** built-in `app.css` at `/theme/custom.css` so you can tone down neo-brutalism or replace accents (see [Custom UI theme](#custom-ui-theme-optional)) |
| `GGHSTATS_DEFAULT_LOCALE` | `en` | Default **dashboard** language when no cookie, `?lang=`, or `Accept-Language` match (see [Web UI languages](#web-ui-languages-i18n)) |
| `GGHSTATS_ENABLED_LOCALES` | `en,es,de` | Comma-separated locales shown in the sidebar selector and accepted from `?lang=` / cookie |

### Web UI languages (i18n)

**Scope:** dashboard HTML and a small set of **browser UI strings** (sync modal, theme toggle label, chart legends). **Not** translated: HTTP API JSON, CLI output, structured logs, or embed badge SVG text.

**Shipped locales:** `en`, `es`, `de`, `fr`, `pt-br` (enable via `GGHSTATS_ENABLED_LOCALES`; default list includes all five).

#### How the active locale is chosen

| Priority | Source |
| --- | --- |
| 1 | Query `?lang=es` (bookmarkable; sets cookie on response) |
| 2 | Cookie `gghstats_locale` (1 year, `Path=/`, `SameSite=Lax`) |
| 3 | `Accept-Language` header (first tag that matches an enabled locale) |
| 4 | `GGHSTATS_DEFAULT_LOCALE` |

Theme (light/dark) stays in **`localStorage`** (`gghstats-theme`); language uses the **cookie** so the first HTML response is already translated.

#### Operator examples

Spanish-first instance (no selector click required for new visitors):

```bash
GGHSTATS_DEFAULT_LOCALE=es
GGHSTATS_ENABLED_LOCALES=en,es,de
```

Force English for one visit: open `https://gghstats.example.com/?lang=en`.  
Permalink in Spanish: `https://gghstats.example.com/h2h?lang=es&a=owner/repoA&b=owner/repoB`.

Set **`GGHSTATS_DEFAULT_LOCALE`** and **`GGHSTATS_ENABLED_LOCALES`** in **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** (`run/common/.env.example`) for production instances.

#### Adding a new locale (contributors)

Example: **Portuguese (Brazil)** as `pt-br`.

1. **Copy the canonical file** (same key set as English):

   ```bash
   cp internal/i18n/locales/en.json internal/i18n/locales/pt-br.json
   ```

2. **Translate values only** — keep every key ID unchanged (`nav.repositories`, `h2h.help_p1`, …). Use `\"` inside JSON strings for embedded quotes (same as `en.json` / `es.json`).

3. **Leave formula lines in English** (copy verbatim from `en.json`):

   - `h2h.help_formula_share`
   - `h2h.help_formula_score`

   Prose around the formulas is translated; notation stays `share_A`, `score_A`, etc.

4. **Do not translate** technical placeholders: `owner/repo` in search fields, API field names, or repo names in charts.

5. **Enable the locale** in config:

   ```bash
   GGHSTATS_ENABLED_LOCALES=en,es,de,pt-br
   # optional default for a PT-BR-first install:
   GGHSTATS_DEFAULT_LOCALE=pt-br
   ```

   `GGHSTATS_DEFAULT_LOCALE` must appear in `GGHSTATS_ENABLED_LOCALES`.

6. **Optional code tweaks** (only when needed):

   | File | When |
   | --- | --- |
   | [`internal/i18n/i18n.go`](internal/i18n/i18n.go) `LangAttr` | Map locale code → BCP 47 for `<html lang="…">` (e.g. `pt-br` → `pt-BR`; already stubbed) |
   | [`internal/server/locale.go`](internal/server/locale.go) `buildLocaleLinks` | Short sidebar label (e.g. `pt-br` → `PT`); if omitted, the code uses `strings.ToUpper(code)` |

7. **Browser-only strings:** if you add keys used from `web/static/app.js`, append the key names to `jsI18nPayload` in [`internal/server/locale.go`](internal/server/locale.go) and translate them in **every** locale file.

8. **Verify key parity and tests:**

   ```bash
   go test ./internal/i18n/...
   ```

9. **Smoke-test in the browser:** `/?lang=pt-br`, `/h2h?lang=pt-br`, a repo page, 404 — confirm sidebar highlight and chart legends.

#### Key conventions

- Nested JSON flattened to dot keys: `nav.repositories`, `chart.legend_unique`.
- H2H help: **one key per paragraph** in JSON; shared formula lines stay English.
- Missing translation in a non-English locale → fallback to **English** for that key.

[Back to top](#gghstats)

### Custom UI theme (optional)

The shipped look is **neo-brutalist** on purpose—not every user or org wants heavy borders and loud chrome. If you prefer something **flatter, calmer, or closer to your brand**, you can supply your own CSS and keep the same binary and data layout.

**Production (Compose / `GGHSTATS_HOST_DATA`):** file placement and **`GGHSTATS_CUSTOM_CSS`** — **[gghstats-selfhosted — Custom UI theme](https://github.com/hrodrig/gghstats-selfhosted#custom-ui-theme-optional)**.

**Local try-out** (no rebuild):

1. Copy one of the **five official example themes** from [`contrib/themes/`](contrib/themes/README.md) (for a stock-Bootstrap feel use **`example-bootstrap-plain.css`**; or write your own CSS targeting `body.app-brutalist` and `html[data-bs-theme="dark"] body.app-brutalist`).
2. Set **`GGHSTATS_CUSTOM_CSS`** to the file path and restart **`gghstats serve`**. The layout adds `<link href="/theme/custom.css?…">` after `/static/app.css`.

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
[![gghstats clones](https://gghstats.example.com/api/v1/badge/your-user/your-repo?metric=clones)](https://gghstats.example.com/your-user/your-repo)
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
| `clones_1d` | number | Clone count for the **latest UTC day with data** among **today and yesterday** (GitHub often omits today's bucket until later). |
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

**Domain series** (besides HTTP and Go runtime): `gghstats_repos_total`, `gghstats_db_size_bytes`, `gghstats_last_sync_timestamp_seconds`, `gghstats_sync_duration_seconds`, `gghstats_github_api_requests_total`, `gghstats_github_rate_limit_remaining`. Refreshed on each scrape and after each successful sync.

**Per-repo gauges** (optional, `GGHSTATS_METRICS_PER_REPO=true`): `gghstats_repo_stars`, `gghstats_repo_forks`, `gghstats_repo_clones`, `gghstats_repo_views`, `gghstats_repo_clones_1d`, `gghstats_repo_clones_7d`, `gghstats_repo_clones_30d` — same semantics as the dashboard **(1d)/(7d)/(30d)** columns. Use with [gghstats-selfhosted observability](https://github.com/hrodrig/gghstats-selfhosted/tree/main/run/docker-compose/observability).

See [Security and quality](#security-and-quality) for the local tooling that scans this surface in CI.

[Back to top](#gghstats)

## Typical scenarios

Examples for **local** `gghstats serve` or CLI. **Production filter and sync defaults** on a server: set them in **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** (`run/common/.env.example` or [`run/standalone/linux/`](https://github.com/hrodrig/gghstats-selfhosted/blob/main/run/standalone/linux/README.md)).

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

### Run always on macOS (LaunchAgent)

Keep sync and the UI running in the background on a Mac (single-user, **127.0.0.1**). Step-by-step: **[contrib/launchd/README.md](contrib/launchd/README.md)** — env file at **`~/.gghstats.env`**, wrapper script, LaunchAgent plist, **`launchctl bootstrap`**. Logs under **`~/Library/Logs/`**.

**Linux (VPS / bare metal):** install `.deb`/`.rpm`, edit **`/etc/gghstats/gghstats.env`**, then **`systemctl enable --now gghstats`** — see **[contrib/systemd/README.md](contrib/systemd/README.md)**.

[Back to top](#gghstats)

## Troubleshooting

### `GGHSTATS_GITHUB_TOKEN is required`

Set `GGHSTATS_GITHUB_TOKEN` in your shell or local `.env` before running `serve`. On a **server** (systemd, Compose): **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**.

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

### systemd / `.deb` / server deployment issues

See **[gghstats-selfhosted — Linux standalone](https://github.com/hrodrig/gghstats-selfhosted/blob/main/run/standalone/linux/README.md)** and [gghstats `contrib/systemd/README.md`](contrib/systemd/README.md) (unit file source only).

[Back to top](#gghstats)

## Release workflow

- Branch policy: day-to-day development on `develop`; **tagged releases** are cut from **`main`**.
- **`VERSION`** file: semantic version **without** `v` (for example `0.3.2`). Must match the static **Version** badge at the top of this README.
- **Git tags:** annotated tag **with** `v` prefix (for example `v0.3.2`), on the commit you want released.

### Default: publish from GitHub Actions (no local GoReleaser required)

Pushing a tag matching `v*` runs [`.github/workflows/release.yml`](.github/workflows/release.yml): `make release-check`, then `goreleaser release --clean` (GitHub release assets, **GHCR** image, `.deb`/`.rpm`, and an updated Homebrew cask on [homebrew-gghstats](https://github.com/hrodrig/homebrew-gghstats)). CI uses the automatic `GITHUB_TOKEN` for releases and the container registry.

**Local `make release`** (maintainers only — needs a GitHub token with release permissions):

```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
make release
```

```bash
# 1) On develop: land changes, bump version if needed
git checkout develop
make release-check                    # lint, test, security, docker-scan (Docker required)
make test-release                     # optional: dry-run GoReleaser (VERSION → *-next; no publish)

# 2) Update VERSION, README version badge, CHANGELOG; commit on develop

# 3) Merge into main (PR or fast-forward), then tag and push
git checkout main && git pull origin main
git merge --ff-only develop           # or: merge via GitHub PR
git push origin main

git tag -a v0.3.2 -m "Release 0.3.2"
git push origin v0.3.2                # triggers Release workflow — builds and publishes artifacts
```

For the **next** release after `0.5.2`, bump `VERSION`, update the badge and [CHANGELOG](CHANGELOG.md), then tag `main` with the matching `v*` tag.

### Optional: publish from your machine

If you run GoReleaser locally instead of relying on CI, checkout **`main`** at the tagged commit, export **`GITHUB_TOKEN`** (or **`GH_TOKEN`**) with `repo` and **packages** access to push GHCR, then:

```bash
make release                          # runs release-check then goreleaser release --clean
```

### Developer checklist

- Update **`CHANGELOG.md`** (move `[Unreleased]` into the new version section).
- Keep **`VERSION`** (no `v`), README **Version** badge, and [CHANGELOG](CHANGELOG.md) in sync; the OCI tag uses the same `v` prefix as the Git tag. Deployment image pins live in **gghstats-selfhosted**.
- Update **`contrib/man/man1/gghstats.1`** — `.TH` date and `gghstats v<VERSION>` (see **[AGENTS.md](AGENTS.md)** — Man page sync). Optional: `make install-man` and `man gghstats`.
- If shipping BSD ports: **`gmake port-freebsd-sync`** and/or **`gmake port-openbsd-sync`** after bumping **`VERSION`** (GNU make in repo root).
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

### Concurrency (reads while sync writes)

Clarification for operators and contributors — not a scalability guarantee.

- **WAL mode** is enabled when the database is opened (`?_journal_mode=WAL`), so the HTTP UI and API can **read** while the background sync **writes** daily traffic rows.
- **SQLite** allows many readers with WAL, but still **one writer at a time** per database file (not row-level locking like PostgreSQL).
- **At most one sync cycle** runs at a time (`sync.Coordinator`): scheduled, startup, and manual sync share that lock; a tick is skipped if a run is already in progress.
- **Single process, single DB file** is the intended deployment: one `gghstats serve` instance per `GGHSTATS_DB`. Do not point multiple writers at the same SQLite file.
- **Consistency during a long sync:** each repo is upserted independently; the index may briefly show a mix of old and new rows until the run finishes. There is no snapshot transaction across the whole repo list.
- **Pragmatic scope:** sync time is dominated by the **GitHub API**, not SQLite; typical self-hosted load (one dashboard, periodic sync) fits this model. Very high write concurrency or multi-instance writes would need extra tuning (for example `busy_timeout`) or a different store — out of scope for the default design.

### Sync serialization (coordinator)

Clarification — not a separate DB write lock.

- **`sync.Coordinator` uses a `sync.Mutex`** so only **one full sync cycle** runs at a time (startup, scheduled tick, or `POST /api/v1/sync`). That is application-level **mutual exclusion between sync runs**, not a mutex around every `Upsert*`.
- **Inside a run**, `sync.Run` iterates repos **sequentially** (`for _, repo := range repos`); each repo triggers several GitHub GETs, then SQLite upserts. There is **no** worker pool or parallel repo sync.
- **SQLite** still enforces one writer at a time; the coordinator avoids overlapping sync goroutines, and the sequential loop avoids multiplying concurrent writers from a single process.

### GitHub API usage and rate limits

Clarification — no built-in backoff today.

- **Authentication:** a **personal access token** via `GGHSTATS_GITHUB_TOKEN` (`Authorization: Bearer …` on REST calls). There is **no** GitHub App or OAuth flow in-tree.
- **Scheduler:** `GGHSTATS_SYNC_INTERVAL` (default **`1h`**) starts the next cycle only when the previous one finished; if a run is still in progress, the tick is **skipped** (`ErrInProgress`). Set **`GGHSTATS_SYNC_ON_STARTUP=false`** to skip the blocking full sync at process start (UI uses existing DB; trigger sync via the dashboard or `POST /api/v1/sync`).
- **Per repo**, a typical sync issues several requests (metadata, open PRs, views, clones, referrers, paths; optional full stargazer history when star sync is enabled). Failures on individual endpoints are logged and the repo loop **continues** (`slog.Warn`, no abort of the whole run).
- **No** explicit handling of `429`, `403` rate-limit responses, `Retry-After`, or exponential backoff in `internal/github`. A non-200 response becomes an error for that call; traffic endpoints are best-effort per repo.
- **Pragmatic scope:** for a personal or small-org PAT and hourly (or slower) sync, GitHub limits are usually enough. Very large repo lists, aggressive intervals, or star-history on huge repos can hit limits — then increase the interval, narrow `GGHSTATS_FILTER`, or expect partial data until a later run succeeds.

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

## Star History

[![Star History Chart](https://api.star-history.com/chart?repos=hrodrig/gghstats&type=date&legend=bottom-right)](https://www.star-history.com/?repos=hrodrig%2Fgghstats&type=date&legend=bottom-right)

[Back to top](#gghstats)

## Acknowledgments

Hats off to **[ghstats](https://github.com/vladkens/ghstats)** by [vladkens](https://github.com/vladkens): a self-hosted GitHub traffic dashboard in **Rust** that also keeps historical traffic beyond GitHub’s short default window, with SQLite and a small deployment story. `gghstats` is a separate **Go** implementation and design, but that project deserves credit as important prior work in the same problem space.

Thanks also to **[git-clone-stats](https://github.com/taylorwilsdon/git-clone-stats)** by [taylorwilsdon](https://github.com/taylorwilsdon): a self-hosted GitHub clone and traffic analytics stack in **Python** with SQLite (or Firestore), a minimal HTML/JS dashboard, and **shields.io-style badges** for README embeds. The badge endpoint and “copy Markdown” embed flow in `gghstats` follow a similar idea; this project is independent Go code, not a port.

[Back to top](#gghstats)

## License

MIT
