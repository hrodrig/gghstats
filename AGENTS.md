# Agent Guidelines (gghstats)

- Use English for all project artifacts (code, docs, commit messages, UI text).
- Follow git flow: feature work in `develop`; releases from `main`.
- **Never** merge to **`main`**, create/push a release tag, or run `make release` / trigger the Release workflow **without explicit user approval in the current conversation.** Ask first — even if `release-check` is green. Surprises on `main` miss checklist items (SPEC “as of”, badges, README tables, BSD sync, follow-up pins).
- Before release (after user OK), run `make release-check` (lint, tests, **cover ≥80%**, security, **docker-scan**; requires Docker). See **SPEC §6.1**.
- Keep `VERSION`, README badges, and release tags synchronized. Deployment manifests (Compose prod, Helm, observability) belong in the **gghstats-selfhosted** repository, not in this repo.
- Do not commit without first showing the proposed commit message and getting explicit user approval.
- **Supply chain:** Prefer resolving dependency and Dependabot-style work **inside the clone** (read diffs, `go get module@version`, `go mod tidy`, `go test ./...`, merge bot branches from **trusted Git remotes**). Do not replace that with pasted blobs from random sites, `curl | sh` installers, or unknown `GOPROXY` / disabled checksums unless the user explicitly accepts that risk.
- **`golang.org/x/net` pin:** Keep **`golang.org/x/net v0.57.0 // indirect`** in **`go.mod`** (HTTP/2 fix + newer transitive **`x/crypto`**; GO-2026-5942 / SVCB panic fixed in ≥0.56.0). **`go mod tidy`** may drop it; after tidy or Dependabot bumps run **`go get golang.org/x/net@v0.57.0`** and **`make check-x-net-pin`**. CI **`make lint`** enforces the pin (`X_NET_MIN_VERSION` in **`Makefile`**). **GoReleaser** does **not** run **`go mod tidy`** (see **`.goreleaser.yaml`**); release builds use committed **`go.mod`** / **`go.sum`**.
- **`.cursor/` is local-only** — not committed to this repository (same for **gghstats-selfhosted**). Put shared agent and release policy in tracked files such as this **AGENTS.md**, **README**, and **CONTRIBUTING.md**.

## Version bump (on `develop`, before merge/tag)

Do the **VERSION bump as a dedicated commit on `develop`** after the feature/docs work is in, **before** proposing `make release-check` → merge to `main` → annotated tag. Never bump only on `main`. **Stop and ask** before steps 8–9.

| # | Artifact | Action |
|---|----------|--------|
| 1 | **`VERSION`** | New semver without `v` (e.g. `0.10.2`) |
| 2 | **`README.md`** | Static **Version** badge `version-<semver>`; fix any “current release” tables that hard-code the old tag; drop dead badges |
| 3 | **`CHANGELOG.md`** | Move `[Unreleased]` notes into `## [<semver>] - YYYY-MM-DD`; update compare links at bottom |
| 4 | **`SPEC.md`** | Header “as of **v\<semver\>**” when the normative contract matches that release |
| 5 | **`contrib/man/man1/gghstats.1`** | `.TH` line: month/year + `gghstats v<semver>` |
| 6 | **BSD ports** | `gmake port-freebsd-sync` and/or `gmake port-openbsd-sync` so `PORTVERSION` / OpenBSD `PKGNAME` match |
| 7 | **Plans / ROADMAP** | Close or retarget band notes if this release ends a band (optional but preferred) |
| 8 | **Gate** | `make release-check` (includes **cover ≥80%**) — run only after user asks |
| 9 | **Ship** | Merge `develop` → `main`, annotated tag `v<semver>`, push tag — **only after user explicitly approves** |

**Follow-ups (other repos, after the GitHub Release is green):** pin **`GGHSTATS_VERSION`** / Helm `appVersion` in **gghstats-selfhosted**; marketing sites if they pin the app version.

## Man page sync (before each release)

Keep **`contrib/man/man1/gghstats.1`** aligned with the CLI and **`serve`** environment variables. GoReleaser gzips it in the release hook; **`.deb`/`.rpm`/FreeBSD/OpenBSD** packages ship the same file.

| Change | Update in `gghstats.1` |
|--------|-------------------------|
| New CLI flag | **CLI FLAGS** or **SERVE FLAGS** (`.TP` blocks) |
| New / changed `GGHSTATS_*` | **ENVIRONMENT**; mirror in `contrib/gghstats.env.example` when operator-facing |
| **`VERSION` bump** | `.TH GGHSTATS 1` line: current month/year and `gghstats v<VERSION>` |

**Source of truth:** `cmd/gghstats/main.go`, `flags.go`, `serve.go`, `contrib/gghstats.env.example`.

**Before tagging:** run **`gmake port-freebsd-sync`** and/or **`gmake port-openbsd-sync`** when BSD ports should match **`VERSION`** (OpenBSD sync also copies **`contrib/openbsd/*`** → **`contrib/openbsd/port/files/`**). Optional local distfiles: **`gmake dist-freebsd`**, **`gmake dist-openbsd`** (`OPENBSD_ARCH=arm64` if needed). **New maintainers:** **`contrib/BSD-PORTS-STEP-BY-STEP.md`**. Detail: **`contrib/freebsd/README.md`**, **`contrib/openbsd/PORT-RELEASE.md`**.

**Verify:** `make install-man MANDIR=/tmp/gghstats-man` then `MANPATH="/tmp/gghstats-man:$(pwd)/contrib/man" man gghstats`.

## Platform tests (native OS)

Lab validation for **`.deb`/`.rpm`/BSD tarball** installs lives in **`testing/platforms/`** (Ansible). **Not** Docker Compose — that is **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** `make test-compose-platforms`.

- Copy **`testing/platforms/inventory/hosts.yml.example`** → **`hosts.yml`** (gitignored); set **`gghstats_github_token`** and SSH targets.
- **`make test-platforms-ping`** then **`make test-platforms`** (optional **`LIMIT=hostname`**).
- See **`testing/platforms/README.md`**.
