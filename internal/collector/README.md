# Anonymous usage collector

Package `internal/collector` sends an **opt-in**, anonymous startup ping about which feature flags are enabled.

## Behavior

- **Off by default.** Enable with `GGHSTATS_ENABLE_COLLECTOR=true`.
- Does **not** send GitHub tokens, API tokens, repo names, filter strings, or file paths.
- Hashes feature-flag names into opaque identifiers before transmit.
- Runs once at process start from `gghstats serve` (see `cmd/gghstats/serve.go`).

## Operator notes

Leave disabled unless you intentionally want aggregate telemetry for the maintainer. Update checks (`GGHSTATS_ENABLE_UPDATE_CHECK`) are separate and only query GitHub Releases for newer gghstats versions.
