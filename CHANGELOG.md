# Changelog

All notable changes to this project are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added (initial release)

- Community standard files:
  - `LICENSE`
  - `CONTRIBUTING.md`
  - `CODE_OF_CONDUCT.md`
  - `SECURITY.md`
  - `.github` issue and PR templates
- Release and quality workflow hardening:
  - `VERSION`-driven releases
  - `snapshot`, `test-release`, `release` targets
  - `govulncheck`, `gocyclo`, `grype` integration
- Kubernetes packaging via Helm chart under `charts/gghstats`.

### Changed

- Project rename finalized to `gghstats`:
  - module path, CLI binary, Docker naming, UI titles and env vars (`GGHSTATS_*`).
- Toolchain and build base image aligned to patched Go (`1.26.1`).

## [0.1.0] - 2026-03-27

### Added

- Initial public release baseline.
