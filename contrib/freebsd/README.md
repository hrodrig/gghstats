# FreeBSD port for gghstats

This directory contains the [FreeBSD](https://www.freebsd.org) port files for **gghstats**. Operators install a package built from these files; **developers** maintain the port here and validate it against GitHub release tarballs before submitting to the official ports tree.

| Audience | Start here |
|----------|------------|
| **Developers** (port, distfile, Bugzilla) | [Developer guide](#developer-guide) below |
| **Operators** (install, config, rc.d) | [Install from port](#install-from-port) |
| **Release maintainer** (version bump, submit diff) | [PORT-RELEASE.md](PORT-RELEASE.md) |

**Production deployment** (Compose, Traefik, TLS): **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**. This port is for **bare-metal** FreeBSD with **rc.d** and **`/etc/gghstats/gghstats.env`**.

---

## Developer guide

### What lives in this directory

| File | Role |
|------|------|
| `Makefile` | FreeBSD **ports** Makefile (`NO_BUILD`, fetch distfile, `do-install`) |
| `pkg-plist` | Files installed under `/usr/local` |
| `pkg-descr` | Short package description for `pkg search` |
| `rc.d/gghstats` | `service gghstats {start,stop,status}` via `daemon(8)` |
| `README.md` | This file |
| `PORT-RELEASE.md` | Release bump + Bugzilla submission |

The **application** is built by **GoReleaser** (or `gmake dist-freebsd` locally). This port **does not compile Go**; it unpacks a release tarball and installs binaries plus metadata.

### Two different `make` commands (important)

FreeBSD ships **BSD make** (`bmake`) as `/usr/bin/make`. The **gghstats repository root** `Makefile` is **GNU make** (uses `$(shell …)`, `define`/`endef`, etc.).

| Context | Command | Example targets |
|---------|---------|-----------------|
| **gghstats repo root** (any OS, or FreeBSD) | **`gmake`** | `port-freebsd-sync`, `dist-freebsd`, `release-check` |
| **Ports tree** (`/usr/ports/...` or `~/ports/...`) | **`make`** (BSD make) | `makesum`, `install`, `deinstall`, `portlint` |

On FreeBSD, if you run plain `make port-freebsd-sync` in the repo you will see errors such as `Unknown modifier ":%M"` or `Invalid line "define …"` — install **`gmake`** and use it:

```bash
pkg install gmake go
cd /path/to/gghstats
gmake port-freebsd-sync
gmake dist-freebsd
```

**Ports work** under `contrib/freebsd/` always uses **BSD `make`** once files are copied into `sysutils/gghstats` in a ports tree.

### Distfile naming (must match GoReleaser)

| Item | Value |
|------|--------|
| Git tag | `v0.6.4` |
| Repo **`VERSION`** | `0.6.4` (no `v`) |
| **PORTVERSION** in this `Makefile` | Same as **`VERSION`** |
| **DISTFILES** | `gghstats_${PORTVERSION}_freebsd_${ARCH:S/aarch64/arm64/}.tar.gz` |
| Example amd64 file | `gghstats_0.6.4_freebsd_amd64.tar.gz` |
| **MASTER_SITES** | `https://github.com/hrodrig/gghstats/releases/download/v${PORTVERSION}/` |

There is **no `v`** inside the tarball filename (the Git tag is still **`v<VERSION>`**). The port **Makefile** must stay aligned with [`.goreleaser.yaml`](../../.goreleaser.yaml) **`name_template`** and **`files`** layout after **`gmake port-freebsd-sync`**.

Tarball layout (top level after extract):

```text
gghstats
share/man/man1/gghstats.1
share/doc/gghstats/LICENSE
etc/gghstats/gghstats.env.example
```

### Where the version comes from

**Only** the file **`VERSION`** at the **gghstats repository root** (e.g. `0.6.4`, no `v` prefix). There is **no** fallback to an old semver in the Makefile.

```bash
cat VERSION          # must show the release you intend (e.g. 0.6.4)
gmake port-freebsd-sync   # copies that into contrib/freebsd/Makefile PORTVERSION=
gmake dist-freebsd        # builds dist/gghstats_<VERSION>_freebsd_<arch>.tar.gz
```

If **`VERSION` is missing or empty**, `port-freebsd-sync` and `dist-freebsd` **fail** (by design). If you copied an incomplete tree to a VM, sync **`VERSION`** from your dev machine or `echo 0.6.4 > VERSION` before running those targets.

**Also required for `go build`:** the **`assets/`** directory (embedded favicons). Copy the **full** tracked repo; a partial copy causes `no required module provides package github.com/hrodrig/gghstats/assets`.

### Developer checklist (new port or version bump)

1. Bump root **`VERSION`**, **`CHANGELOG.md`**, README badge, and **`contrib/man/man1/gghstats.1`** (`.TH` line) — see root **`AGENTS.md`** (Man page sync).
2. Confirm **`cat VERSION`**, then **`gmake port-freebsd-sync`** (sets **PORTVERSION** in `contrib/freebsd/Makefile`).
3. Merge to **`main`**, tag **`v*`**; CI/GoReleaser publishes **`gghstats_*_freebsd_*`** on GitHub Releases (after `freebsd` is in `.goreleaser.yaml`).
4. On a FreeBSD host (or VM), test the port (below) before Bugzilla.
5. Submit **git diff** to ports — [PORT-RELEASE.md](PORT-RELEASE.md); [Bug 294001 comment](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=294001#c1) notes triage prefers **git diff** over shar alone.

### Build a test distfile (before GitHub has the release)

**On macOS or Linux** (cross-compile for VPS amd64):

```bash
cd /path/to/gghstats
gmake port-freebsd-sync
GOOS=freebsd GOARCH=amd64 gmake dist-freebsd
ls dist/gghstats_*_freebsd_amd64.tar.gz
```

**On FreeBSD** (native arch — omit `GOOS`/`GOARCH`):

```bash
pkg install gmake go
cd /path/to/gghstats
gmake port-freebsd-sync
gmake dist-freebsd
ls dist/gghstats_*_freebsd_*.tar.gz
```

Output path: **`dist/gghstats_<version>_freebsd_<arch>.tar.gz`**.

### Test the port on a FreeBSD VM (full path)

You need:

1. A **clone of gghstats** (with this `contrib/freebsd/` tree — can be `scp`/`rsync` from your dev machine; need not be pushed to GitHub yet for a private test).
2. A **ports tree** — either:
   - **`/usr/ports`** (already on many images), or
   - **`~/ports`** from `git clone https://git.FreeBSD.org/ports.git`

Example using **`/usr/ports`** (tested on **FreeBSD 15 amd64**):

```bash
# 1) Build distfile in repo
pkg install gmake go
cd ~/gghstats
gmake port-freebsd-sync
gmake dist-freebsd

# 2) Install port files into ports tree (custom path — not in official tree yet)
mkdir -p /usr/ports/sysutils/gghstats
cp -r ~/gghstats/contrib/freebsd/* /usr/ports/sysutils/gghstats/

# 3) Feed local distfile (no GitHub fetch)
cp ~/gghstats/dist/gghstats_*.tar.gz "$(make -C /usr/ports/sysutils/gghstats -V DISTDIR)/"

# 4) Build package
cd /usr/ports/sysutils/gghstats
make makesum
make install

# 5) Configure and start
mkdir -p /etc/gghstats /var/db/gghstats
cp /usr/local/etc/gghstats/gghstats.env.example /etc/gghstats/gghstats.env
vi /etc/gghstats/gghstats.env    # GGHSTATS_GITHUB_TOKEN, GGHSTATS_DB=/var/db/gghstats/gghstats.db, etc.
chmod 600 /etc/gghstats/gghstats.env
sysrc gghstats_enable=YES
service gghstats start
service gghstats status
curl -s http://127.0.0.1:8080/api/v1/healthz
```

**Note:** A manual binary at `/usr/local/bin/gghstats` from an earlier smoke test can conflict with the package. Before `make install`:

```bash
pkg delete -f gghstats 2>/dev/null || true
rm -f /usr/local/bin/gghstats
```

**Alternative ports tree path** — replace `/usr/ports` with `~/ports` in the commands above.

**Copy distfile only** — if the repo on the VM is old, build the tarball on the build host and `scp` `dist/gghstats_*.tar.gz`; you still need `contrib/freebsd/*` in the ports tree for `make install`.

### Manual sync / distfile (no `gmake`)

If `gmake` is unavailable, sync version and build the tarball by hand:

```bash
cd /path/to/gghstats
ver=$(cat VERSION | tr -d '\n\r')
sed -i.bak "s/^PORTVERSION=.*/PORTVERSION=	${ver}/" contrib/freebsd/Makefile
rm -f contrib/freebsd/Makefile.bak

arch=$(uname -m | sed 's/aarch64/arm64/')
stage=/tmp/gghstats-dist-$$
mkdir -p "$stage/share/man/man1" "$stage/share/doc/gghstats" "$stage/etc/gghstats" dist
GOOS=freebsd GOARCH=$arch CGO_ENABLED=0 go build -ldflags "-s -w" -o "$stage/gghstats" ./cmd/gghstats
cp contrib/man/man1/gghstats.1 "$stage/share/man/man1/"
cp LICENSE "$stage/share/doc/gghstats/"
cp contrib/gghstats.env.example "$stage/etc/gghstats/"
tar -C "$stage" -czf "dist/gghstats_${ver}_freebsd_${arch}.tar.gz" .
rm -rf "$stage"
```

### Smoke test without the ports tree

Fastest runtime check (no `make install` from ports):

```bash
gmake dist-freebsd
tar -xzf dist/gghstats_*_freebsd_*.tar.gz -C /tmp/gghstats-ex
install -m 755 /tmp/gghstats-ex/gghstats /usr/local/bin/gghstats
# export vars or use /etc/gghstats/gghstats.env, then:
gghstats serve
```

### gghstats port specifics

| Topic | gghstats |
|-------|----------|
| Config | `/etc/gghstats/gghstats.env` (shell env) |
| Daemon | `gghstats serve` (env from file; rc.d may use a wrapper — see **`rc.d/gghstats`**) |
| Distfile | `gghstats_${PORTVERSION}_freebsd_…` (no **`v`** in filename) |
| Repo helpers | **`gmake port-freebsd-sync`**, **`gmake dist-freebsd`** (GNU make at repo root) |
| OpenBSD sibling | **`contrib/openbsd/`**, **`gmake dist-openbsd`** |

### Troubleshooting (developers)

| Symptom | Cause | Fix |
|---------|--------|-----|
| `make: Unknown modifier ":%M"` in repo | BSD `make` parsing GNU `Makefile` | Use **`gmake`** |
| `make install` cannot fetch distfile | No GitHub release yet or wrong name | Copy tarball to **`DISTDIR`**; run **`make makesum`** |
| `PORTVERSION` / checksum mismatch | **`VERSION`** not synced | **`gmake port-freebsd-sync`** |
| `service gghstats` fails immediately | Missing token or bad **`gghstats.env`** | Check **`/var/log/gghstats.log`**; fix **`GGHSTATS_GITHUB_TOKEN`** |
| `make install` overwrites nothing | Stale package | **`make deinstall clean`** then **`make install`** |
| **`PORTVERSION` → 0.3.2** (wrong) | **`VERSION` missing** on VM; old Makefile fallback | **`cat VERSION`**; copy full repo; use updated Makefile (no fallback) |
| `package …/assets` build error | Incomplete repo copy (no **`assets/`**) | **`rsync -a`** or full clone including **`assets/`** |

---

## Install from port

When the port is in the official FreeBSD ports tree:

```bash
cd /usr/ports/sysutils/gghstats
make install
```

When using a **local** port (not yet in the tree):

```bash
mkdir -p /usr/ports/sysutils/gghstats   # or ~/ports/sysutils/gghstats
cp -r /path/to/gghstats/contrib/freebsd/* /usr/ports/sysutils/gghstats/
cd /usr/ports/sysutils/gghstats
make install
```

**Reinstall after port file changes:** `make deinstall`, `make clean`, `make install`.

### Verify install

```bash
gghstats version
pkg info gghstats
```

If an older package is installed: `make reinstall`.

### Test with a local distfile (summary)

See [Developer guide — Test the port on a FreeBSD VM](#test-the-port-on-a-freebsd-vm-full-path). Short form:

```bash
gmake port-freebsd-sync && gmake dist-freebsd
cp dist/gghstats_*.tar.gz "$(make -C /usr/ports/sysutils/gghstats -V DISTDIR)/"
cd /usr/ports/sysutils/gghstats && make makesum && make install
```

The port installs:

- `/usr/local/bin/gghstats`
- `/usr/local/share/man/man1/gghstats.1.gz`
- `/usr/local/share/doc/gghstats/LICENSE`
- `/usr/local/etc/gghstats/gghstats.env.example`
- `/usr/local/etc/rc.d/gghstats`

---

## Config (required)

gghstats uses **environment variables** (main README **Configuration**). Copy the example and set at least **`GGHSTATS_GITHUB_TOKEN`**:

```bash
mkdir -p /etc/gghstats /var/db/gghstats
cp /usr/local/etc/gghstats/gghstats.env.example /etc/gghstats/gghstats.env
vi /etc/gghstats/gghstats.env
chmod 600 /etc/gghstats/gghstats.env
```

Use **`GGHSTATS_DB=/var/db/gghstats/gghstats.db`** (or another persistent path) on servers.

---

## Daemon (rc.d)

```bash
sysrc gghstats_enable=YES
service gghstats start
```

| Command | Action |
|---------|--------|
| `service gghstats start` | Start |
| `service gghstats stop` | Stop |
| `service gghstats restart` | Restart |
| `service gghstats status` | Status |

### rc.conf variables

| Variable | Default | Description |
|----------|---------|-------------|
| `gghstats_enable` | `NO` | `YES` to start on boot |
| `gghstats_envfile` | `/etc/gghstats/gghstats.env` | Env file (`required_files`) |
| `gghstats_logfile` | `/var/log/gghstats.log` | Log file for `daemon(8)` |

### Logging

```bash
tail -f /var/log/gghstats.log
```

**/etc/newsyslog.conf`:**

```txt
/var/log/gghstats.log   644  5  100  *  B
```

---

## Submitting to official ports

See **[PORT-RELEASE.md](PORT-RELEASE.md)**. Prefer a **git diff** against the ports tree ([Bug 294001 comment](https://bugs.freebsd.org/bugzilla/show_bug.cgi?id=294001#c1) on submission format).
