# Changelog

All notable changes to this project are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

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

[Unreleased]: https://github.com/hrodrig/gghstats/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.2
[0.1.1]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.1
[0.1.0]: https://github.com/hrodrig/gghstats/releases/tag/v0.1.0
