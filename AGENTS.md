# Agent Guidelines (gghstats)

- Use English for all project artifacts (code, docs, commit messages, UI text).
- Follow git flow: feature work in `develop`; releases from `main`.
- Before release, run `make release-check` (or `STRICT_RELEASE=1` for image scan gate).
- Keep `VERSION`, README badges, and release tags synchronized. Deployment manifests (Compose prod, Helm, observability) belong in the **gghstats-selfhosted** repository, not in this repo.
- Do not commit without first showing the proposed commit message and getting explicit user approval.
- **Supply chain:** Prefer resolving dependency and Dependabot-style work **inside the clone** (read diffs, `go get module@version`, `go mod tidy`, `go test ./...`, merge bot branches from **trusted Git remotes**). Do not replace that with pasted blobs from random sites, `curl | sh` installers, or unknown `GOPROXY` / disabled checksums unless the user explicitly accepts that risk.
- **`.cursor/` is local-only** — not committed to this repository (same for **gghstats-selfhosted**). Put shared agent and release policy in tracked files such as this **AGENTS.md**, **README**, and **CONTRIBUTING.md**.

## Man page sync (before each release)

Keep **`contrib/man/man1/gghstats.1`** aligned with the CLI and **`serve`** environment variables. GoReleaser gzips it in the release hook; **`.deb`/`.rpm`/FreeBSD/OpenBSD** packages ship the same file.

| Change | Update in `gghstats.1` |
|--------|-------------------------|
| New CLI flag | **CLI FLAGS** or **SERVE FLAGS** (`.TP` blocks) |
| New / changed `GGHSTATS_*` | **ENVIRONMENT**; mirror in `contrib/gghstats.env.example` when operator-facing |
| **`VERSION` bump** | `.TH GGHSTATS 1` line: current month/year and `gghstats v<VERSION>` |

**Source of truth:** `cmd/gghstats/main.go`, `flags.go`, `serve.go`, `contrib/gghstats.env.example`.

**Before tagging:** run **`gmake port-freebsd-sync`** and/or **`gmake port-openbsd-sync`** when BSD ports should match **`VERSION`** (OpenBSD sync also copies **`contrib/openbsd/*`** → **`contrib/openbsd/port/files/`**). Optional local distfiles: **`gmake dist-freebsd`**, **`gmake dist-openbsd`** (`OPENBSD_ARCH=arm64` if needed). Workflows: **`contrib/freebsd/README.md`**, **`contrib/openbsd/PORT-RELEASE.md`**.

**Verify:** `make install-man MANDIR=/tmp/gghstats-man` then `MANPATH="/tmp/gghstats-man:$(pwd)/contrib/man" man gghstats`.

## Platform tests (native OS)

Lab validation for **`.deb`/`.rpm`/BSD tarball** installs lives in **`testing/platforms/`** (Ansible). **Not** Docker Compose — that is **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** `make test-compose-platforms`.

- Copy **`testing/platforms/inventory/hosts.yml.example`** → **`hosts.yml`** (gitignored); set **`gghstats_github_token`** and SSH targets.
- **`make test-platforms-ping`** then **`make test-platforms`** (optional **`LIMIT=hostname`**).
- See **`testing/platforms/README.md`**.
