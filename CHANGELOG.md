# Changelog

All notable changes to this project are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

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

[Unreleased]: https://github.com/hrodrig/gghstats/compare/v0.1.4...HEAD
[0.1.4]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.4
[0.1.3]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.3
[0.1.2]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.2
[0.1.1]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.1
[0.1.0]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.0
