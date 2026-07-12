# Changelog

All notable changes to this project are documented in this file.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- **README comparison table:** expand vs ghstats, git-clone-stats, gh-tracker, and GitHub Traffic; more operator-facing rows (maintenance, CLI, badges, i18n, metrics breadth, ops manifests); ToC link under Features.
- **README Features:** add missing product capabilities with short “what it solves” blurbs (history >14d, sync, momentum, badges, themes, Prometheus, rate limit/whitelist, packaging).
- **ROADMAP / band plans:** post-0.9 park of filtered leftover QW and sync notes into [plan-v0.10.x.md](docs/plan-v0.10.x.md); dogfood contract + CORS warn into [plan-v0.11.x.md](docs/plan-v0.11.x.md); Line A / “next” text updated after 0.9 ship.

## [0.9.0] - 2026-07-12

### Added

- **Repo page trends:** clone momentum (7d / 30d) on the repository detail page, reusing `internal/h2h` (`Momentum7d` / `Momentum30d`) with the same percent formatting as H2H.
- **`gghstats backup` / `gghstats restore`:** snapshot SQLite with `VACUUM INTO` (`--output`); restore by file copy (`--input` onto `--db` / `GGHSTATS_DB`). Stop `serve` before restore if the DB is in use.
- **Demo mode:** `gghstats serve --demo` / `GGHSTATS_DEMO=true` seeds sample repos (`demo/alpha|beta|gamma`) when the DB is empty; no GitHub token, sync, or update check.
- **Security headers baseline:** `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy` on all HTTP responses.
- **SRI** for Chart.js 4.4.1, Luxon 3.5.0, and chartjs-adapter-luxon 1.3.1 (pinned unpkg URLs + `integrity`).
- **Access logs:** `http` slog lines include `status`.
- **README comparison table** vs niche peers; **`internal/collector/README.md`** documents opt-in telemetry.
- **ROADMAP.md** and **SPEC.md:** product direction and normative HTTP API / sync contracts (aligned with worker pool and GitHub retries).
- **docs/plan-v0.9.x.md**, **plan-v0.10.x.md**, **plan-v0.11.x.md**, **plan-v1.0.0.md:** scoped band plans (exit criteria + checklists).
- **docs/README.md:** index for band plans and VHS terminal demo.
- **docs/demo.tape** / **docs/demo.gif:** Charm VHS CLI walkthrough (embedded in README Demo).

### Changed

- **Container runtime:** switch from Alpine 3.24 to **`gcr.io/distroless/static-debian13:nonroot`** (`Dockerfile` + `Dockerfile.release`, same pattern as groot). Builder uses `golang:1.26.5-bookworm` with `CGO_ENABLED=0`. Default in-image `GGHSTATS_DB=/data/gghstats.db`.
- **ROADMAP.md / docs/plan-v0.11.x.md:** 0.11.x primary goal is **API-only mode** + JSON dogfood (Line D); webhooks moved to stretch / 1.1+. Name stays gghstats.
- **ROADMAP.md:** release bands to 1.x with links to per-band plans under `docs/`; priority lines A–D.
- **README Database / sync notes:** document worker pool and GitHub client retries (was still describing serial sync and no backoff).
- **docs/plan-v0.9.x.md:** band checklist complete (trends, backup, demo, comparison, quick wins).

### Fixed

- **CHANGELOG compare link:** `[Unreleased]` pointed at `v0.8.1...HEAD` (was still `v0.8.0...HEAD` during the 0.9 cycle).

## [0.8.1] - 2026-07-09

### Fixed

- **Go toolchain:** Bump `go.mod` and Dockerfile to **1.26.5** (was `go.mod` 1.26.4 / image `golang:1.26.4-alpine`). Fixes **GO-2026-4970** (High, stdlib; `make docker-scan`) and **GO-2026-5856** (`crypto/tls` ECH privacy leak; `govulncheck` in CI).

## [0.8.0] - 2026-06-26

### Added

- **`data-gghstats-role` semantic HTML attributes:** stable DOM anchors (`sidebar`, `header`, `content`, `footer`, `kpi-card`, `kpi-value`, `stat-card`, `stat-value`, `repo-title`, `fork-label`, `chart-card`, `error-panel`, `error-code`, `nav`) added to dashboard templates. CustomCSS (`GGHSTATS_CUSTOM_CSS`) can target these without depending on Bootstrap utility classes that change between major versions.
- **`gghstats_rate_limited_requests_total{status}`:** counter of rate-limiter outcomes (`allowed` | `blocked`). Helps operators tune `GGHSTATS_RATE_LIMIT_*` and alert on systematic 429s from misconfigured proxies.
- **`gghstats_whitelist_requests_total{status}`:** counter of whitelist decisions (`allowed` | `blocked`). Surfaces whether `GGHSTATS_WHITELIST` is blocking legitimate traffic or silently allowing everything.
- **`gghstats_badge_requests_total{status}` + `gghstats_badge_duration_seconds{status}`:** per-badge-outcome counter and latency histogram (`ok`, `not_found`, `error`, `unauthorized`, `bad_request`). Previously badge requests were only visible in the generic HTTP metrics.
- **Cosign keyless signing:** checksums.txt and container images (`ghcr.io/hrodrig/gghstats:vX.Y.Z`) signed via Sigstore OIDC. Verify with `cosign verify-blob --certificate-identity ... --certificate-oidc-issuer ... checksums.txt`.
- **SBOM (SPDX + CycloneDX):** source-level software bill of materials per release (`gghstats_vX.Y.Z_sbom.spdx.json` + `.cyclonedx.json`). Generated via syft, attached to every GitHub release.

### Changed

- **HTTP transport (GitHub client):** set explicit `MaxIdleConnsPerHost=4`, `MaxIdleConns=16`, `IdleConnTimeout=90s` on the `http.Transport`. Avoids redundant TLS handshakes when 4 sync workers hit `api.github.com` in parallel.
- **SQLite connection pool:** `db.SetMaxOpenConns(4)`, `db.SetMaxIdleConns(2)`, `db.SetConnMaxIdleTime(5m)`. Prevents unbounded pool growth under HTTP bursts while keeping WAL's reader parallelism.
- **SQLite PRAGMA `synchronous=NORMAL`:** add `_pragma=synchronous(NORMAL)` to the DSN. Trades a small crash-safety window (last few transactions can roll back on power loss) for ~2x faster WAL writes; safe for a single-host self-hosted dashboard.

### Fixed

- **SQLite DSN:** switched to `_pragma=...` for all PRAGMAs (`busy_timeout`, `journal_mode`, `synchronous`). The legacy `_busy_timeout=5000` parameter in `modernc.org/sqlite` was silently ignored — the busy timeout was 0 in practice, exposing sync writes to `SQLITE_BUSY` under contention. The 5s wait now actually applies.

### Added

- **`gghstats_github_rate_limit_reset_seconds{resource}`:** gauge populated from the GitHub `X-RateLimit-Reset` header. Lets dashboards plot "minutes until rate-limit reset" via `time() - metric`.
- **`gghstats_sync_repos_processed_total{status}`:** counter of repositories processed per sync cycle, by outcome (`success` = all steps OK, `error` = at least one step failed). Independent of `gghstats_sync_errors_total{kind}` (per-step classifier).

## [0.7.13] - 2026-06-26

### Added

- **`--sync-workers` / `GGHSTATS_SYNC_WORKERS`:** configurable worker pool size for the sync cycle (default 4). Concurrency between repositories keeps large accounts within the GitHub REST rate budget while remaining well below the secondary rate limit.
- **`gghstats_sync_errors_total{kind}`:** Prometheus counter that classifies sync failures by step (`worker`, `repo_meta`, `open_prs`, `views`, `clones`, `referrers`, `paths`, `stargazers`). Useful for dashboards in `gghstats-selfhosted`.
- **Exponential-backoff retries with full jitter:** GitHub API calls retry on HTTP 429, 403 (rate-limited), 5xx, and network errors. Honors `X-RateLimit-Reset` when the API advertises a near-future reset. Default 4 attempts, 1s base, 60s cap. Disable with `client.SetRetry(github.RetryConfig{})`.

### Changed

- **Sync loop is now concurrent:** `internal/sync.Run` dispatches repos through a bounded worker pool instead of serial `for`. Existing single-repo `fetch` command is unchanged.
- **SQLite DSN:** added `_busy_timeout=5000` so brief lock contention during sync writes surfaces as a wait rather than `SQLITE_BUSY`.

### Fixed

- **Large-account sync time:** sync now scales with `min(repos, workers)` HTTP round-trips instead of `repos * 7` sequential requests, removing the silent Secondary Rate Limit cliff for accounts with 200+ repositories.

## [0.7.11] - 2026-06-25

### Added

- **Head HTML injection (`GGHSTATS_HEAD_HTML`):** arbitrary HTML injected just before `</head>` on every dashboard page. Enables first-party analytics, extra CSS, meta tags, etc., without modifying templates.
- **Reverse proxy rules (`GGHSTATS_REVERSE_PROXY_RULES`):** JSON array of `{local, url, headers}` mappings that mount reverse proxies on the same HTTP server. Each rule serves a local path prefix from a remote backend, bypassing CSP and mixed-content issues.
- **Anonymous usage collector** (`internal/collector/`): opt-in telemetry package (disabled by default). Collects anonymous feature-flag hashes on startup (no credentials, paths, or repo names). Set `GGHSTATS_ENABLE_COLLECTOR=true` to enable.
- **Update check (`GGHSTATS_ENABLE_UPDATE_CHECK`):** checks GitHub API for newer gghstats releases on startup; logs a warning when a newer version is found. Enabled by default; set `false` to disable.

### Changed

- **`kikolog.go` → `logformat.go`:** renamed internal log handler from `kikoHandler`/`NewKikoLogHandler` to `formatHandler`/`NewFormatLogHandler`; app name moved to `version.AppName`.
- **`PublicMiddlewareSkip()`:** now accepts `[]ReverseProxyRule` and derives proxy prefix exemptions dynamically instead of hardcoding `/kiko/`. Without this, any `local` path other than `/kiko/` would be blocked by rate-limit or whitelist middleware.
- **Log format:** cleaned up to consistently use ` - ` separators (e.g. `gghstats - INFO - message`) without padding artifacts.

### Fixed

- **Reverse-proxy middleware exemption:** `/kiko/` was hardcoded in `PublicMiddlewareSkip`; any rule with a different `local` path got rate-limited or whitelist-blocked. Now all configured `ReverseProxyRule.Local` paths are dynamically added to the skip list.
- **ModifyResponse CSP stripping:** verified for `.css`, `.gif`, and `.js` content types; large body streaming and upstream timeout tested.

### Security

- **Reverse proxy CSP sanitisation:** backend `Content-Security-Policy` headers are stripped from proxied responses, preventing the upstream analytics backend from controlling the dashboard's CSP.

## [0.7.9] - 2026-06-16

### Added

- **`gghstats --print-sample-config`** — writes annotated env template to stdout (same as `contrib/gghstats.env.example`); documented in README.

### Fixed

- **Docker image build:** whitelist **`contrib/`** in `.dockerignore` for embedded sample env template.

## [0.7.8] - 2026-06-15

### Fixed

- **README badge embeds:** exempt `/api/v1/badge/*` from per-IP rate limiting and IP whitelist (with `/metrics` and `/api/v1/healthz`). Fixes broken `![…](badge)` images when `GGHSTATS_WHITELIST_PATHS` includes `/api/` or GitHub/camo proxies share one client IP bucket.

### Changed

- **Docs:** sync README version badge and install examples to **0.7.8**; note badge routes in middleware section; man page and env examples.
- **Docker release image:** Alpine **3.24** in `Dockerfile.release` (aligned with local `Dockerfile`).

## [0.7.7] - 2026-06-14

### Added

- **IP whitelist:** restrict access by client IP/CIDR with `GGHSTATS_WHITELIST` and `GGHSTATS_WHITELIST_PATHS`. Non-matching IPs receive 403. Scoped to specific paths (e.g. protect only `/api/`) or all routes. Exempts `/metrics` and `/api/v1/healthz`.

### Security

- **Query parameter sanitization:** `sort` and `dir` parameters in `parseIndexQueryParams` are now validated against a whitelist of known values. Invalid values fall back to safe defaults (`total_clones` / `desc`). The store layer already enforced this; server-side validation adds defence in depth.

### Changed

- **Docs:** clarify `/metrics` is public by default and should be protected at the network edge.
- **Docs:** add middleware chain order to rate limiting section.
- Bump **`modernc.org/sqlite`** from 1.51.0 to 1.52.0 (SQLite 3.53.2; Dependabot #8).

## [0.7.5] - 2026-06-14

### Added

- **`scripts/install.sh`:** one-liner install from GitHub releases (`curl -fsSL https://raw.githubusercontent.com/hrodrig/gghstats/main/scripts/install.sh | sh`).
- **Per-IP rate limiting:** protects the HTTP server from abuse with a configurable token-bucket middleware. Enabled by default (120 req/min, burst 20). Skipped for `/metrics` and `/api/v1/healthz`. Configure via `GGHSTATS_RATE_LIMIT_*` env vars; set `GGHSTATS_RATE_LIMIT_ENABLED=false` to disable.

### Security

- Rate limiting mitigates M7 from the professionalism audit (no protection on `POST /api/v1/sync`).

## [0.7.4] - 2026-06-10

### Added

- HTML **`<link rel="canonical">`** and **`<meta name="description">`** on dashboard pages (uses **`GGHSTATS_PUBLIC_URL`** when set). Index canonical is `/` without `lang` / sort / pagination; **404** pages get **`noindex`**.

### Changed

- **GoReleaser:** `nfpms.builds` → `nfpms.ids` (removes deprecation warning).
- Go toolchain **1.26.4** (Docker builder `golang:1.26.4-alpine`).

### Security

- Go **1.26.4**: **CVE-2026-42504** (`mime` encoded-word DoS, via Prometheus `expfmt`) and **CVE-2026-27145** (`crypto/x509` hostname verification); rebuild required.

## [0.7.3] - 2026-05-29

### Added

- **`GET /robots.txt`** and **`GET /sitemap.xml`**: per-deployment SEO (uses **`GGHSTATS_PUBLIC_URL`** when set; localhost gets `Disallow: /` and an empty sitemap).

### Changed

- Bump pinned **`golang.org/x/net`** to **v0.55.0** (transitive **`x/crypto`**; Snyk noise reduction).

### Fixed

- H2H **Compare** button uses the same orange CTA style as **Search** in light mode (`app-repo-search-submit`).

## [0.7.2] - 2026-05-29

### Changed

- Bump **`modernc.org/sqlite`** to **v1.51.0** (Dependabot; manual merge — not PR #5).

### Security

- Keep explicit **`golang.org/x/net v0.45.0`** pin (Snyk / 0.7.1 fix). Dependabot PR #5 would have dropped the pin and resolved **v0.43.0** via Prometheus; do not merge bot sqlite-only bumps without re-adding **`x/net`**.

## [0.7.1] - 2026-05-25

### Security

- Bump transitive **`golang.org/x/net`** to **v0.45.0** (HTTP/2 infinite-loop fix; via Prometheus client).
- Bump transitive **`golang.org/x/sys`** to **v0.44.0** (Windows `NewNTUnicodeString` overflow fix; via `modernc.org/sqlite`).

### Changed

- **FreeBSD port:** replace incorrect **`NO_ARCH=yes`** with **`ONLY_FOR_ARCHS= amd64 aarch64`** (matches GoReleaser **`freebsd`** amd64/arm64 tarballs; same pattern as pgwd).
- **`make release-check`:** always runs **`docker-scan`** (same as pgwd); removed opt-in **`STRICT_RELEASE=1`** gate.

## [0.7.0] - 2026-05-24

### Added

- **FreeBSD:** `contrib/freebsd/` port (Makefile, pkg-plist, `rc.d/gghstats`, README developer guide, PORT-RELEASE); GoReleaser **`freebsd`** archives; **`gmake dist-freebsd`** / **`gmake port-freebsd-sync`** (GNU make; BSD `make` for `/usr/ports` only).
- **OpenBSD:** `contrib/openbsd/` (rc.d, `gghstats-serve`, `port/` with `files/` + PLIST, README, PORT-RELEASE); GoReleaser **`openbsd`** archives; **`gmake dist-openbsd`** / **`gmake port-openbsd-sync`** (`OPENBSD_ARCH` default `amd64`). Port installs **`gghstats-serve`** required by rc.d.
- **`contrib/BSD-PORTS-STEP-BY-STEP.md`:** end-to-end guide (tarball → port → install) for FreeBSD and OpenBSD newcomers.
- **AGENTS.md:** man page sync checklist before release (keep `contrib/man/man1/gghstats.1` aligned with `VERSION` and env/CLI).
- **`.deb`/`.rpm` maintainer scripts:** `contrib/deb/prerm.sh` (stop/disable `gghstats.service` on remove) and `contrib/deb/postrm.sh` (remove `/etc/gghstats` on **purge**; SQLite under `/var/lib/gghstats` is kept).
- **systemd (Linux):** `contrib/systemd/gghstats.service`, `contrib/gghstats.env.example` → `/etc/gghstats/gghstats.env`, and [contrib/systemd/README.md](contrib/systemd/README.md). **.deb/.rpm** install the unit to `/lib/systemd/system/`.
- **macOS (launchd):** `contrib/launchd/` — wrapper script, LaunchAgent plist template, and README for always-on local use.
- **CLI local UX:** `gghstats run` (alias for `serve`); **`--open`** and **`GGHSTATS_OPEN_BROWSER`** open the default browser when the dashboard is ready.
- **README Install:** package-manager table, separate **Build** section, **Always-on (macOS)**, and **Debian / Ubuntu** + **AlmaLinux / Fedora / RHEL** install notes.
- **README demo:** side-by-side screenshots — GitHub Traffic (14 days) vs gghstats historical SQLite charts; ecosystem links to **pgwd** / **pgwd-selfhosted**.
- **Docs:** Install/Quick start = quick commands only; **systemd**, Debian/RHEL package setup, and deployment guides moved to **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** (`run/standalone/linux/`, `run/docker-compose/`, `run/common/`).
- **Alpine (OpenRC):** `contrib/openrc/gghstats.initd` and README; linux release tarballs include `share/openrc/gghstats.initd`.
- **Platform tests:** Ansible support for **Alpine** (`platform_vars/alpine.yml`, OpenRC in `gghstats_daemon` / uninstall); unified **`gghstats_package_source`** docs (`local` / `auto` / `release`).

### Changed

- **Default bind address:** `gghstats serve` listens on **`127.0.0.1`** (localhost only) instead of `0.0.0.0`. Set **`GGHSTATS_HOST=0.0.0.0`** if you need LAN access without a reverse proxy. Docker Compose in this repo and **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** still set `GGHSTATS_HOST=0.0.0.0` for containers.

## [0.6.4] - 2026-05-21

### Added

- **Sidebar sync status i18n:** localized “no sync yet” / last sync messages in all enabled locales.
- **Homebrew:** official tap [`hrodrig/homebrew-gghstats`](https://github.com/hrodrig/homebrew-gghstats) — `brew install hrodrig/gghstats/gghstats`; cask updated automatically on each release.
- **Linux packages:** `.deb` and `.rpm` via GoReleaser `nfpms`; `contrib/man/man1/gghstats.1` and `make install-man`.
- **README quick start:** Homebrew, local binary download, and one-line `docker run`.

### Changed

- **H2H score subline:** only the leading repo shows a margin line (`lead_margin`); ties or small deltas show the interval label only.
- **Release workflow:** publishes Homebrew cask updates to the tap repo on each tagged release.

## [0.6.3] - 2026-05-21

### Fixed

- **German (de) UI:** complete pass on leftover English — H2H title **Direktvergleich (H2H)**, **Repositorien** labels, footer **Motor**, chart legend **Eindeutig**, embed **Badge einbetten**, and related copy (formulas in the score help remain English, same as other locales).

### Added

- **`internal/i18n` tests:** locale resolution, env helpers, and H2H localization helpers — statement coverage back above **86%** project-wide (`make cover`).

## [0.6.2] - 2026-05-21

### Added

- **UI locales:** French (`fr`) and Brazilian Portuguese (`pt-br`); sidebar **EN | ES | DE | FR | PT** when enabled.
- Default **`GGHSTATS_ENABLED_LOCALES`:** `en,es,de,fr,pt-br`.

## [0.6.1] - 2026-05-21

### Fixed

- **Locale cookie:** set `Secure` on `gghstats_locale` (CodeQL `go/cookie-secure-not-set` / alert #5).
- **Server:** shared `requestScheme()` for HTTPS detection (`TLS` / `X-Forwarded-Proto`), reused by badge base URL logic.

## [0.6.0] - 2026-05-20

### Added

- **Web UI i18n:** English (default), Spanish, and German via JSON locales under `internal/i18n/locales/`.
- **Language selector** in the sidebar (EN | ES | DE); preference stored in cookie `gghstats_locale` (theme stays in `localStorage`).
- **Locale resolution:** `?lang=` → cookie → `Accept-Language` → `GGHSTATS_DEFAULT_LOCALE`.
- **`GGHSTATS_ENABLED_LOCALES`** and **`GGHSTATS_DEFAULT_LOCALE`** environment variables.
- **README:** [Web UI languages (i18n)](README.md#web-ui-languages-i18n) — operator examples and a step-by-step guide to add a locale (e.g. `pt-br`).

### Changed

- **H2H** metric labels, suggestions, and chart titles are localized with the rest of the UI.
- **Chart legends** (Unique / Count, index clones series) and repo chart titles follow the active locale via server + `window.gghstatsI18n`.
- **H2H help formulas** (`share_A`, `score_A`, …) stay in English in all locales; surrounding paragraphs are translated.

## [0.5.2] - 2026-05-20

### Fixed

- **H2H momentum chart:** pad the daily calendar from the clone fetch start so repos with sparse traffic (recent spike only) still get a rolling momentum series; align chart card heights across clones, views, and momentum.

### Changed

- **H2H momentum chart:** dual Y-axes when scales differ (e.g. large positive vs negative %); always use separate axes on the momentum chart.
- **H2H copy:** document that very high momentum % (prior window near zero + recent spike) is expected, not a calculation error.

## [0.5.0] - 2026-05-19

### Added

- **Head to Head (H2H):** compare two repositories at `/h2h` with weighted share scores (7d / 30d / all time), metrics table, Chart.js charts, and an in-page explanation of how scores are calculated.
- **`GGHSTATS_SYNC_ON_STARTUP`:** set `false` (or `0` / `no` / `off`) to skip the blocking full sync when the process starts; default `true`. Scheduled sync and manual `POST /api/v1/sync` are unchanged.

### Changed

- **Repository detail:** chart card headers, legends, tooltips, and canvas labels include the repository name (e.g. `Clones - owner/repo`).

## [0.4.0] - 2026-05-18

### Added

- **Prometheus domain metrics** on `GET /metrics`: `gghstats_repos_total`, `gghstats_db_size_bytes`, `gghstats_last_sync_timestamp_seconds`, `gghstats_sync_duration_seconds`, `gghstats_github_api_requests_total`, `gghstats_github_rate_limit_remaining`. Refreshed on scrape and after each successful sync.
- **Optional per-repo gauges** (`GGHSTATS_METRICS_PER_REPO=true`): `gghstats_repo_stars`, `_forks`, `_clones`, `_views`, `_clones_1d`, `_clones_7d`, `_clones_30d` (aligned with dashboard windows).

## [0.3.2] - 2026-05-17

### Fixed

- **`clones_1d` / index (1d) column:** use the latest UTC day with clone data among **today and yesterday** instead of **today only**. GitHub traffic often omits the current UTC bucket until later, which made **(1d)** show `0` despite activity in **(7d)** / **(30d)**.
- **Sync:** persist traffic dates with explicit **UTC** when upserting daily views/clones.

## [0.3.1] - 2026-05-17

### Changed

- **Embed badge:** Markdown snippet and live preview link to the repository page (clickable badge), consistent with the README badge pattern.

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

[Unreleased]: https://github.com/hrodrig/gghstats/compare/v0.9.0...HEAD
[0.9.0]: https://github.com/hrodrig/gghstats/compare/v0.8.1...v0.9.0
[0.8.1]: https://github.com/hrodrig/gghstats/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/hrodrig/gghstats/compare/v0.7.11...v0.8.0
[0.7.11]: https://github.com/hrodrig/gghstats/compare/v0.7.10...v0.7.11
[0.7.10]: https://github.com/hrodrig/gghstats/compare/v0.7.9...v0.7.10
[0.7.9]: https://github.com/hrodrig/gghstats/compare/v0.7.8...v0.7.9
[0.7.8]: https://github.com/hrodrig/gghstats/compare/v0.7.7...v0.7.8
[0.7.7]: https://github.com/hrodrig/gghstats/compare/v0.7.6...v0.7.7
[0.7.6]: https://github.com/hrodrig/gghstats/compare/v0.7.5...v0.7.6
[0.7.5]: https://github.com/hrodrig/gghstats/compare/v0.7.4...v0.7.5
[0.7.4]: https://github.com/hrodrig/gghstats/compare/v0.7.3...v0.7.4
[0.7.3]: https://github.com/hrodrig/gghstats/compare/v0.7.2...v0.7.3
[0.7.2]: https://github.com/hrodrig/gghstats/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/hrodrig/gghstats/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/hrodrig/gghstats/compare/v0.6.4...v0.7.0
[0.6.4]: https://github.com/hrodrig/gghstats/compare/v0.6.3...v0.6.4
[0.6.3]: https://github.com/hrodrig/gghstats/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/hrodrig/gghstats/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/hrodrig/gghstats/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/hrodrig/gghstats/compare/v0.5.2...v0.6.0
[0.5.2]: https://github.com/hrodrig/gghstats/compare/v0.5.0...v0.5.2
[0.5.0]: https://github.com/hrodrig/gghstats/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/hrodrig/gghstats/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/hrodrig/gghstats/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/hrodrig/gghstats/compare/v0.3.0...v0.3.1
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
