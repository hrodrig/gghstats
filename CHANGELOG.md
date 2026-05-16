# Changelog

All notable changes to this project are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.3.0] - 2026-05-16

### Added

- **Badges (shields-style SVG):** `GET /api/v1/badge/{owner}/{repo}` with `metric` (`clones`, `clones_30d`, `views`, `stars`), optional `.svg` suffix, `Cache-Control`, and public-by-default auth (`GGHSTATS_BADGE_PUBLIC=false` requires `x-api-token`). Repo detail page: **Embed badge** block with preview and **Copy** Markdown. README **Acknowledgments** credit [git-clone-stats](https://github.com/taylorwilsdon/git-clone-stats) for the badge embed pattern.
- **Traffic time series API:** `GET /api/v1/repos/{owner}/{repo}/traffic?days=30` (same auth as `/api/repos`) returns daily `clones` and `views` rows (`date`, `count`, `uniques`). `days=0` returns all stored history; default `days` is 30 (UTC calendar window).
- **Manual sync:** `POST /api/v1/sync` and `GET /api/v1/sync` (same auth as `/api/repos`) trigger or inspect a background GitHub sync; only one run at a time (scheduler skips if busy). Optional `?repo=owner/name` syncs a single repository. Sidebar **Sync all** / **Sync this repo** when `GGHSTATS_API_TOKEN` is set (token via modal, stored in browser sessionStorage).

### Changed

- **README:** live **gghstats** clone badge for `hrodrig/gghstats` (served from the [public demo](https://gghstats.hermesrodriguez.com); requires demo on **≥ 0.3.0** with the repo synced).
- **Web index (`/`):** table columns reordered to **Name | Stars | Forks | Views | Clones | (1d) | (7d) | (30d)**. Clone windows **(1d)** (today UTC) and **(7d)** (last 7 calendar days) are new; **(30d)** keeps the same 30-day sum as before. All three windows are sortable.
- **`GET /api/repos`:** `clones_1d` and `clones_7d` on each repo summary (same UTC window semantics as the index).

## [0.2.1] - 2026-05-17

### Changed

- Bump **modernc.org/sqlite** from v1.47.0 to v1.50.1 (bundled SQLite **3.53.x**), with upstream fixes for pre-update hooks, `Deserialize`, allocator ownership, `Exec` with `RETURNING`, VFS reads, and related hardening (see [modernc.org/sqlite changelog](https://gitlab.com/cznic/sqlite/-/blob/master/CHANGELOG.md)).

## [0.2.0] - 2026-05-16

### Added

- **Optional custom UI theme:** **`GGHSTATS_CUSTOM_CSS`** — path to a regular `.css` file on disk for operators who want a **simpler or custom look** than the default neo-brutalist UI; served at **`GET /theme/custom.css`** and linked **after** `app.css`. Invalid paths log a warning. **Official example gallery:** five starters under [`contrib/themes/`](contrib/themes/README.md) (including a **Bootstrap-plain** look and documentation screenshot).

## [0.1.7] - 2026-05-16

### Added

- **Web index (`/`):** **Clones (30d)** column — sum of daily clone counts from GitHub traffic for the last 30 calendar days (UTC); days without a row count as 0. Sortable; default index sort remains **total clones descending**.

## [0.1.6] - 2026-05-11

### Changed

- Go toolchain **1.26.3** (stdlib fixes reported by vulnerability scanners for 1.26.2); Docker builder `golang:1.26.3-alpine`.
- Docker runtime images (`Dockerfile`, `Dockerfile.release`): **Alpine 3.22** with `apk upgrade` for patched base packages (including Busybox) vs prior 3.21.
- **`Dockerfile` builder:** set `GOPROXY` / `GOSUMDB` so `go mod download` does not fail when the host forwards an invalid or empty `GOPROXY` into the build.

## [0.1.5] - 2026-05-03

### Added

- **Web index (`/`):** **Clones over time** line chart (Chart.js) to the right of the repository table — daily **clone counts** summed across every repo in the current list and search filter (same scope as KPI totals; not only the current page). Data window: up to the last **120** days with clone rows in SQLite.

## [0.1.4] - 2026-04-13

### Changed

- Go toolchain **1.26.2** (addresses stdlib CVEs reported by scanners for 1.26.1); Docker builder image `golang:1.26.2-alpine`.
- **Security** workflow: run Grype via `docker run anchore/grype:latest --pull=always` (same pattern as `make docker-scan`), replacing `curl | sh` install.
- **Docker** runtime image (`Dockerfile`, `Dockerfile.release`): on Alpine 3.21, run `apk update`, install `ca-certificates`, then `apk upgrade` so OpenSSL/base packages pick up security revisions from the Alpine 3.21 repository at build time.
- **Makefile:** `docker run` for Grype uses `--pull=always` (including the `grype` / `dir:.` fallback) so `anchore/grype:latest` is not a stale local cache.
- **`make grype` / `grype dir:.`:** exclude `./dist/**` and `./gghstats` (Grype/Syft glob rules) so scans do not treat locally built binaries (older embedded Go in buildinfo) as the project stdlib version.

## [0.1.3] - 2026-04-04

### Fixed

- CLI `export` and `report`: `--days` and `--output` parse correctly when passed with the subcommand (single `FlagSet` per command).
- Docker image build: include `assets/` in the build context (`.dockerignore` whitelist).

### Added

- Codecov upload in CI; broadened unit tests (CLI dispatch, fetch/serve paths, store, server, report).
- Optional `GGHSTATS_GITHUB_API_BASE_URL` for GitHub Enterprise or integration tests (see `.env.example`).

### Changed

- Dependency updates (including Prometheus client); `govulncheck` reports no known vulnerabilities.

## [0.1.2] - 2026-04-02

### Added

- Prometheus `GET /metrics` endpoint (`gghstats_*` HTTP metrics, build info, Go/process collectors); optional disable via `GGHSTATS_METRICS=false`.

### Changed

- Production Compose, Helm chart, and observability documentation moved to the **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** repository so application versioning stays independent of infrastructure changes.

## [0.1.1] - 2026-04-02

### Changed

- Web UI: Search button uses theme-specific orange tokens (`--brutal-search-cta`) so light mode no longer shows the primary red accent; hover/active states keep the same fill color.

## [0.1.0] - 2026-03-29

### Added

- Self-hosted dashboard and CLI for GitHub repository traffic stats (views, clones, referrers, paths, stars) with SQLite history beyond GitHub’s 14-day window.
- Web UI, JSON API, and CLI (`fetch`, `export`, `report`).
- Docker / Compose / Helm deployment examples; multi-arch images on GHCR via GoReleaser (`dockers_v2`, `Dockerfile.release`).
- Community standard files: `LICENSE`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, GitHub issue/PR templates.
- Release and quality tooling: `VERSION`-driven releases, `snapshot` / `test-release` / `release`, `govulncheck`, `gocyclo`, `grype`; GitHub Actions for CI, security scans, CodeQL, and tag-triggered release.

### Changed

- Project naming and module path finalized as `gghstats` (binary, Docker image, `GGHSTATS_*` environment variables).
- Toolchain and build base image aligned to Go **1.26.1**.

[Unreleased]: https://github.com/hrodrig/gghstats/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/hrodrig/gghstats/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/hrodrig/gghstats/releases/tag/v0.2.1
[0.2.0]: https://github.com/hrodrig/gghstats/releases/tag/v0.2.0
[0.1.7]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.7
[0.1.6]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.6
[0.1.5]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.5
[0.1.4]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.4
[0.1.3]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.3
[0.1.2]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.2
[0.1.1]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.1
[0.1.0]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.0
