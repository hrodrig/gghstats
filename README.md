# gghstats

[![Version](https://img.shields.io/badge/version-0.1.0-blue)](https://github.com/hrodrig/gghstats/releases)
[![Release](https://img.shields.io/github/v/release/hrodrig/gghstats)](https://github.com/hrodrig/gghstats/releases)
[![CI](https://github.com/hrodrig/gghstats/actions/workflows/ci.yml/badge.svg)](https://github.com/hrodrig/gghstats/actions)
[![Go 1.26.1](https://img.shields.io/badge/go-1.26.1-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/hrodrig/gghstats)](https://pkg.go.dev/github.com/hrodrig/gghstats)
[![Go Report Card](https://goreportcard.com/badge/github.com/hrodrig/gghstats)](https://goreportcard.com/report/github.com/hrodrig/gghstats)
[![deps.dev](https://img.shields.io/badge/deps.dev-go%20module-blue)](https://deps.dev/go/github.com/hrodrig/gghstats)

**Repo:** [github.com/hrodrig/gghstats](https://github.com/hrodrig/gghstats) · **Releases:** [Releases](https://github.com/hrodrig/gghstats/releases)

Self-hosted dashboard and CLI for GitHub repository traffic stats. GitHub only keeps traffic for 14 days; `gghstats` keeps historical data indefinitely in SQLite.

## Table of contents

- [Features](#features)
- [Quick start](#quick-start)
- [Install](#install)
- [Usage](#usage)
- [Examples](#examples)
- [Configuration](#configuration)
- [Environment file](#environment-file)
- [Typical scenarios](#typical-scenarios)
- [Deployments](#deployments)
- [Troubleshooting](#troubleshooting)
- [Release workflow](#release-workflow)
- [Security and quality](#security-and-quality)
- [Database](#database)
- [Community standards](#community-standards)
- [License](#license)

## Features

- Collects views, clones, referrers, popular paths, and star history
- Auto-discovers repositories (or filters by org/repo rules)
- Web dashboard with Chart.js graphs
- JSON API for external integrations
- CLI mode for fetch/report/export
- Single binary, SQLite storage, no external DB dependency
- Docker and Helm support

[Back to top](#gghstats)

## Quick start

### Docker Compose

```bash
cp .env.example .env
# Edit .env: set GGHSTATS_GITHUB_TOKEN (and optionally GGHSTATS_FILTER, GGHSTATS_PORT, etc.)
docker compose up -d
```

Open <http://localhost:8080>.

The template [`.env.example`](.env.example) lists every variable the server and Compose understand; `.env` stays local and is never committed.

### Plain Docker

```bash
docker run -d \
  -e GGHSTATS_GITHUB_TOKEN=ghp_xxx \
  -e GGHSTATS_FILTER="your-github-user/*" \
  -p 8080:8080 \
  -v ./data:/data \
  --name gghstats \
  ghcr.io/hrodrig/gghstats:0.1.0
```

[Back to top](#gghstats)

## Install

### Go install

```bash
go install github.com/hrodrig/gghstats/cmd/gghstats@latest
```

### Build from source

```bash
git clone https://github.com/hrodrig/gghstats.git
cd gghstats
make install
```

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
| `GGHSTATS_API_TOKEN` | (none) | Protect `/api/*` endpoints |
| `GGHSTATS_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` (slog only; startup banner always prints) |

### Token setup

1. Go to <https://github.com/settings/tokens>
2. Generate a classic token
3. Use `public_repo` scope (or `repo` for private repos)

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

### API

When `GGHSTATS_API_TOKEN` is configured:

```bash
curl -H "x-api-token: your-token" http://localhost:8080/api/repos
```

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

### Docker Compose (deployment)

```yaml
services:
  gghstats:
    image: ghcr.io/hrodrig/gghstats:${GGHSTATS_VERSION:-0.1.0}
    restart: unless-stopped
    environment:
      - GGHSTATS_GITHUB_TOKEN=${GGHSTATS_GITHUB_TOKEN}
      - GGHSTATS_PORT=${GGHSTATS_PORT:-8080}
      - GGHSTATS_FILTER=your-github-user/*
    ports:
      - "${GGHSTATS_PORT:-8080}:${GGHSTATS_PORT:-8080}"
    volumes:
      - ./data:/data
```

### Docker Compose + Traefik (production VPS)

Use [`docker-compose.prod.yml`](docker-compose.prod.yml) for **HTTPS** (Let's Encrypt), **HTTP→HTTPS redirect**, and **no host port** on gghstats (only Traefik listens on 80/443).

1. Point **DNS** `A`/`AAAA` for your hostname to the VPS public IP.
2. In `.env`, set at least: `GGHSTATS_GITHUB_TOKEN`, `GGHSTATS_HOSTNAME` (FQDN, e.g. `stats.example.com`), `ACME_EMAIL` (Let's Encrypt).
3. Start:

```bash
docker compose -f docker-compose.prod.yml up -d
```

Traefik keeps certificates in the volume `traefik_letsencrypt`. Other `GGHSTATS_*` variables work like the simple Compose file.

### Helm (Kubernetes)

Chart path: `charts/gghstats`.

```bash
kubectl create secret generic gghstats-secret \
  --from-literal=github-token=ghp_your_token_here

helm upgrade --install gghstats ./charts/gghstats \
  --set githubToken.existingSecret=gghstats-secret \
  --set image.tag=0.1.0
```

Customize environment and persistence in `charts/gghstats/values.yaml`.

[Back to top](#gghstats)

## Troubleshooting

### `GGHSTATS_GITHUB_TOKEN is required`

Set `GGHSTATS_GITHUB_TOKEN` in your shell or `.env` file before running `serve`.

### Dashboard shows no repositories

- Wait for the initial sync to finish.
- Verify filter rules (`GGHSTATS_FILTER`) are not excluding all repos.
- Confirm token scope includes repository metadata access.

### Port `8080` already in use

Set another listen port via env or flag:

```bash
export GGHSTATS_PORT=9090
gghstats serve
# or: gghstats serve --port 9090
```

### API returns `401 unauthorized`

Confirm request header exactly matches configured token:

```bash
curl -H "x-api-token: $GGHSTATS_API_TOKEN" http://localhost:8080/api/repos
```

[Back to top](#gghstats)

## Release workflow

- Branch policy: day-to-day development on `develop`, production releases from `main`.
- `VERSION` file uses semantic version without `v` (for example `0.1.0`).
- Git tags use `v` prefix (for example `v0.1.0`).

```bash
# 1) finish work on develop
git checkout develop
make lint
make test
make security

# optional strict gate (includes docker image scan)
make release-check STRICT_RELEASE=1

# 2) set release version
echo "0.1.1" > VERSION

# 3) validate release artifacts locally
make snapshot
make test-release

# 4) merge develop -> main and release from main
git checkout main
git merge --ff-only develop
git tag -a v0.1.1 -m "Release 0.1.1"
git push origin main
git push origin v0.1.1

# 5) publish (main only)
make release
```

Developer release checklist:

- Update `CHANGELOG.md` (`[Unreleased]` -> new version section).
- Confirm README version badge matches `VERSION`.
- Ensure CI and Security workflows are green before pushing the release tag.
- Verify GHCR image tags and Helm chart values use the intended version.

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

## License

MIT
