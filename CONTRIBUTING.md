# Contributing to gghstats

Thank you for your interest in contributing.

## How to contribute

- **Bug reports and feature requests:** Open an [issue](https://github.com/hrodrig/gghstats/issues) and use the template that fits best.
- **Code changes:** Open a pull request from a branch (for example `fix/description` or `feat/short-name`).
- **Branch policy:** Use `develop` as integration branch and `main` for stable releases.
- **PR base branch:** Open normal feature/fix PRs against `develop`.
- **Scope:** Keep PRs focused and small when possible.

## Code style

- Format Go code with `gofmt -s`.
- Run checks locally before submitting:
  - `make lint`
  - `make test`
  - `make security`
- For release-related changes, run:
  - `make release-check`

## Release flow

- Version is read from `VERSION` (semantic version without `v`, for example `0.1.2`).
- Release tag format is `v<version>` (for example `v0.1.2`).
- Release PR flow: merge `develop` into `main`, then tag and release from `main`.
- Useful commands:
  - `make snapshot`
  - `make test-release`
  - `make release`

## Questions

If you are unsure, open an issue and describe your proposal before implementing large changes.
