# FreeBSD Port: Release and Update Procedure

Steps to create or update the **gghstats** port for a new application release. Operator-facing install docs: **[README.md](README.md)**. Full developer context ( **`gmake` vs `make`**, VM test, distfile names): **[README.md — Developer guide](README.md#developer-guide)**.

## Prerequisites

- FreeBSD machine or VM with a **ports tree** (`/usr/ports` or `~/ports`)
- **`pkg install gmake go`** on FreeBSD when building distfiles from the **gghstats** repo on that host
- Optional: `pkg install sharutils` (`gshar` for legacy shar bundles — Bugzilla prefers **git diff** today)

## Part A: In the gghstats repo (before release)

1. Bump root **`VERSION`**, **CHANGELOG**, README badge, and **`contrib/man/man1/gghstats.1`** (see root **`AGENTS.md`**).

2. Sync the port version (use **`gmake`** on FreeBSD, or **`make`** on macOS/Linux with GNU make):

   ```bash
   cd /path/to/gghstats
   gmake port-freebsd-sync    # FreeBSD
   gmake port-openbsd-sync    # if also shipping OpenBSD (see contrib/openbsd/PORT-RELEASE.md)
   # make port-freebsd-sync   # macOS/Linux if `make` is GNU make
   ```

3. Merge to **`main`**, tag **`v*`**, run release (GoReleaser publishes **`gghstats_<version>_freebsd_amd64.tar.gz`** and arm64). Verify the filename matches **`DISTFILES`** in `contrib/freebsd/Makefile`.

4. Copy port files to your ports tree when testing or submitting:

   ```bash
   cp -r /path/to/gghstats/contrib/freebsd/* /usr/ports/sysutils/gghstats/
   # or: ~/ports/sysutils/gghstats/
   ```

## Part B: On FreeBSD (ports tree)

Use **BSD `make`** inside the ports directory (not `gmake`).

### First-time submission (port not yet in official tree)

1. **Branch** in ports (example):

   ```bash
   cd ~/ports    # or work in a git clone of ports, not only /usr/ports snapshot
   git checkout -b add-gghstats-port
   mkdir -p sysutils/gghstats
   cp -r /path/to/gghstats/contrib/freebsd/* sysutils/gghstats/
   ```

2. **Local distfile** (if GitHub release is not up yet):

   ```bash
   cd /path/to/gghstats
   gmake port-freebsd-sync dist-freebsd
   cp dist/gghstats_*.tar.gz "$(make -C sysutils/gghstats -V DISTDIR)/"
   ```

3. **distinfo + install + portlint:**

   ```bash
   cd sysutils/gghstats
   make makesum
   make install
   portlint
   gghstats version
   curl -s http://127.0.0.1:8080/api/v1/healthz   # after configuring /etc/gghstats/gghstats.env
   ```

4. **Clean** before generating submission artifacts:

   ```bash
   make clean
   ```

5. **Git diff** (preferred — see [Bug 294001 comment](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=294001#c1)):

   ```bash
   cd ~/ports
   git add sysutils/gghstats/
   git commit -m "New port: sysutils/gghstats - GitHub traffic dashboard"
   git diff main..HEAD > ~/gghstats-port.diff
   ```

   Optional shar (older workflow):

   ```bash
   gshar $(find sysutils/gghstats -type f | sort) > ~/gghstats.shar
   ```

6. **Bugzilla:** https://bugs.freebsd.org/submit/ — Product **Ports & Packages**, Component **Individual Port(s)**. Attach **`gghstats-port.diff`**. Mention **Tested on FreeBSD 15 amd64** (or your version/arch).

### Update (port already in official tree)

1. Refresh files from `contrib/freebsd/`
2. `cd sysutils/gghstats && make makesum && make install && portlint`
3. `git diff main..your-branch > gghstats-update.diff`
4. Submit diff to Bugzilla

## Quick reference: distfile naming

| Item | Example |
|------|---------|
| Git tag | `v0.6.4` |
| **`VERSION`** / **PORTVERSION** | `0.6.4` |
| **DISTFILES** | `gghstats_${PORTVERSION}_freebsd_${ARCH:S/aarch64/arm64/}.tar.gz` |
| Example file | `gghstats_<PORTVERSION>_freebsd_amd64.tar.gz` |
| **MASTER_SITES** | `https://github.com/hrodrig/gghstats/releases/download/v${PORTVERSION}/` |

**Filename vs Git tag:** the release tag is **`v<PORTVERSION>`** (e.g. `v0.6.4`), but **`DISTFILES`** has **no `v`** in the tarball name (`gghstats_0.6.4_freebsd_amd64.tar.gz`). Keep [`.goreleaser.yaml`](../../.goreleaser.yaml) **`name_template`** and this port **Makefile** in sync after **`gmake port-freebsd-sync`**.

## Local distfile without release

```bash
# Cross-build on macOS/Linux, copy to amd64 VPS
gmake port-freebsd-sync
GOOS=freebsd GOARCH=amd64 gmake dist-freebsd

# FreeBSD native
pkg install gmake go
gmake port-freebsd-sync dist-freebsd
```
