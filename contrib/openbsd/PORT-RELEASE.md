# OpenBSD port — release and lab validation

How to bump the **gghstats** port for a new app release, validate it on an OpenBSD VM, and (optionally) submit to the official ports tree.

| Doc | Audience |
|-----|----------|
| **[README.md](README.md)** | Install from release tarball or repo scripts (`/etc/rc.d`, manual) |
| **[port/README.md](port/README.md)** | Port directory layout, CVS/git checkout details, `file://` fetch |
| **[DEBUG-VM.md](DEBUG-VM.md)** | rc.d troubleshooting without the ports framework |

---

## Big picture (read this first)

Three layers — easy to mix up:

| Layer | What it is | Where it lives |
|-------|------------|----------------|
| **Application** | `gghstats` binary + `gghstats-serve` + rc.d model | Built in **gghstats** repo; GoReleaser publishes **`gghstats_<ver>_openbsd_amd64.tar.gz`** |
| **Port skeleton** | Recipe to turn that tarball into an OpenBSD **package** | **`contrib/openbsd/port/`** in gghstats (copied into **`/usr/ports/sysutils/gghstats/`**) |
| **Ports tree** | Full OpenBSD build system (`bsd.port.mk`, distfiles, packages) | **`/usr/ports`** on the VM (CVS or git checkout — several GB) |

**Ansible platform tests** use the **tarball + repo scripts** (`contrib/openbsd/*` → `/etc/rc.d/gghstats`). That path does **not** require the ports tree.

**Port validation** checks that **`make install`** in `/usr/ports/sysutils/gghstats` produces a package with the same binaries, **`gghstats-serve`**, and a working **`rcctl`** setup.

---

## Prerequisites

- **gghstats repo** on a **macOS or Linux build host** with **GNU make** (`brew install make` → use **`gmake`** or **`$(brew --prefix make)/bin/gmake`** — there is no Homebrew formula named `gmake`).
- **OpenBSD VM** with disk for **`/usr/ports`** (git shallow clone is enough for one port).
- **`go`** on the build host where you run **`gmake dist-openbsd`**.

---

## Part A — In the gghstats repo (before release)

1. Bump **`VERSION`**, **CHANGELOG**, README badge, **`contrib/man/man1/gghstats.1`** (see root **`AGENTS.md`**).

2. Sync port metadata **and** `port/files/` from `contrib/openbsd/`:

   ```bash
   cd /path/to/gghstats
   gmake port-openbsd-sync
   gmake port-freebsd-sync    # if also shipping FreeBSD
   ```

3. Optional: build a local OpenBSD distfile (same layout as GoReleaser):

   ```bash
   gmake dist-openbsd
   # OPENBSD_ARCH=arm64 gmake dist-openbsd
   ```

   Output: **`dist/gghstats_<version>_openbsd_amd64.tar.gz`**.

4. Merge to **`main`**, tag **`v*`**, run release. GoReleaser must publish a tarball whose name matches **`DISTFILES`** in **`contrib/openbsd/port/Makefile`**.

5. Stage the port for the VM (**only** `contrib/openbsd/port/`, not all of `contrib/openbsd/`):

   ```bash
   rm -rf /tmp/gghstats-port
   cp -r contrib/openbsd/port /tmp/gghstats-port
   scp -r /tmp/gghstats-port/* root@openbsd-test:/tmp/gghstats-port/
   scp dist/gghstats_*_openbsd_amd64.tar.gz root@openbsd-test:/tmp/
   ```

---

## Part B — Lab validation on OpenBSD (redo from scratch)

Use **BSD `make`** inside **`/usr/ports/sysutils/gghstats`**, not **`gmake`** from the app repo.

**Versions:** After **`gmake port-openbsd-sync`**, the port **Makefile** must match the repo root **`VERSION`** file (`PKGNAME`, `DISTFILES`, `MASTER_SITES`). In the shell blocks below, set **`ver`** once from the port (do not hardcode `0.6.4` / `0.7.2` in the doc):

```sh
cd /usr/ports/sysutils/gghstats
ver=$(make -V PKGNAME | sed 's/^gghstats-//')   # e.g. 0.6.4
dist=$(make -V DISTFILES)                        # e.g. gghstats_0.6.4_openbsd_amd64.tar.gz
```

On the **build host**, the tarball name is **`gghstats_<VERSION>_openbsd_<arch>.tar.gz`** (same as **`DISTFILES`**).

**Port layout (must match repo):**

| File | Role |
|------|------|
| **`pkg/gghstats.rc`** | Source for **`generate-readmes`** → **`/etc/rc.d/gghstats`** (only **`*.rc`** files are copied) |
| **`pkg/PLIST`** | First line: **`@rcscript ${RCDIR}/gghstats`** — **no** **`etc/rc.d/gghstats`** |
| **`files/gghstats-serve`**, **`files/gghstats-start`** | Installed to **`/usr/local/bin/`** by **do-install** |

Same pattern as **`www/gitea/pkg/gitea.rc`**. **`COMMENT`** in the port **Makefile** must be **≤ 60 characters**.

### B1. Ports tree (once per VM)

```sh
pkg_add git
cd /usr
rm -rf ports
git clone --depth 1 https://github.com/openbsd/ports.git ports
```

CVS and checkout pitfalls: **[port/README.md](port/README.md)**.

### B2. Teardown (when redoing the lab test)

```sh
# ver from port Makefile if present; else set manually from repo VERSION (e.g. ver=0.7.0)
if [ -f /usr/ports/sysutils/gghstats/Makefile ]; then
  ver=$(make -C /usr/ports/sysutils/gghstats -V PKGNAME | sed 's/^gghstats-//')
fi
: "${ver:?set ver= to match repo VERSION}"

pkg_delete gghstats-${ver} 2>/dev/null
rm -f /etc/rc.d/gghstats
rm -rf /usr/ports/sysutils/gghstats
rm -f /usr/ports/plist/amd64/gghstats-${ver}
rm -f /usr/ports/packages/amd64/all/gghstats-*.tgz \
      /usr/ports/packages/amd64/no-arch/gghstats-*.tgz \
      /usr/ports/packages/amd64/ftp/gghstats-*.tgz
```

### B3. Install port directory (copy **only** `contrib/openbsd/port/`)

On the **build host** (gghstats repo root, after **`gmake port-openbsd-sync`**):

```bash
rm -rf /tmp/gghstats-port
cp -r contrib/openbsd/port /tmp/gghstats-port
scp -r /tmp/gghstats-port/* root@openbsd-test:/tmp/gghstats-port/
scp dist/gghstats_*_openbsd_amd64.tar.gz root@openbsd-test:/tmp/
```

On the VM:

```sh
mkdir -p /usr/ports/sysutils/gghstats
cp -r /tmp/gghstats-port/* /usr/ports/sysutils/gghstats/

ls /usr/ports/sysutils/gghstats/
# Makefile  README.md  files/  pkg/  test-install-from-dist.sh
# Must NOT include: gghstats (loose file at port root), distinfo (until makesum)

ls /usr/ports/sysutils/gghstats/pkg/
# DESCR  PLIST  gghstats.rc

grep rcscript /usr/ports/sysutils/gghstats/pkg/PLIST
# @rcscript ${RCDIR}/gghstats
```

### B4. Distfile and checksums

```sh
export PORTSDIR=/usr/ports
export DISTDIR=/usr/ports/distfiles
mkdir -p /usr/ports/distfiles
cd /usr/ports/sysutils/gghstats
dist=$(make -V DISTFILES)
cp /tmp/$dist /usr/ports/distfiles/
# Or: cp /tmp/gghstats_*_openbsd_amd64.tar.gz /usr/ports/distfiles/  (one file only)

make makesum
```

If **`make -V FULLDISTDIR`** prints **`${DISTDIR}`**, set **`export DISTDIR=...`** explicitly.

### B5. Build package and install (critical steps)

```sh
cd /usr/ports/sysutils/gghstats
make clean=package clean
make package FETCH_PACKAGES=No
```

**Must appear in the log:**

```text
Installing /usr/ports/sysutils/gghstats/pkg/gghstats.rc as .../fake-amd64/etc/rc.d/gghstats
===>  Building package for gghstats-<VERSION>
```

If you only see **`Link to ...`** and no **Building package**, the **`.tgz` was not rebuilt** — repeat **`make clean=package clean`**.

If **`Error: change in plist`**, remove the cached registry (lab VM):

```sh
ver=$(make -V PKGNAME | sed 's/^gghstats-//')
rm -f /usr/ports/plist/amd64/gghstats-${ver}
make package FETCH_PACKAGES=No
```

Verify the package contains the rc script (path may be **`no-arch`** or **`all`**):

```sh
PKG=$(ls /usr/ports/packages/amd64/no-arch/gghstats-*.tgz 2>/dev/null \
   || ls /usr/ports/packages/amd64/all/gghstats-*.tgz)
tar -tzf "$PKG" | grep etc/rc.d/gghstats
```

Install:

```sh
make install
```

**Success:**

```text
gghstats-<VERSION>: ok
The following new rcscripts were installed: /etc/rc.d/gghstats
```

```sh
ls -l /etc/rc.d/gghstats
rcctl enable gghstats
```

No symlink from **`/usr/local/etc/rc.d`** is required.

### B6. Configure and run

```sh
mkdir -p /etc/gghstats /var/lib/gghstats
cp /usr/local/share/examples/gghstats/gghstats.env.example /etc/gghstats/gghstats.env
vi /etc/gghstats/gghstats.env    # GGHSTATS_GITHUB_TOKEN, GGHSTATS_FILTER (quote ! in ksh)
chmod 600 /etc/gghstats/gghstats.env

rcctl start gghstats
rcctl check gghstats    # expect gghstats(ok)
ftp -o - http://127.0.0.1:8080/api/v1/healthz
```

Optional boot: add **`gghstats`** to **`pkg_scripts`** in **`/etc/rc.conf.local`**.

### B7. Port vs tarball (rc.d)

| Install method | rc.d on disk | `rcctl` |
|----------------|--------------|---------|
| **Port / pkg_add** | **`/etc/rc.d/gghstats`** (via **`pkg/gghstats.rc`** + **`@rcscript`**) | **`rcctl enable gghstats`** |
| **Tarball / Ansible** | **`/etc/rc.d/gghstats`** (manual **`install`**) | Same |

---

## Part C — Submit to ports@openbsd.org (optional)

The port is **not** in the official tree yet. When ready:

1. **`gmake port-openbsd-sync`** and copy **`contrib/openbsd/port/*`** into your ports checkout under **`sysutils/gghstats/`**.
2. Use the **published** GitHub release tarball (or a local copy with the same bytes) → **`make makesum`** → commit **`distinfo`** in the ports diff (not in the gghstats repo — see **`port/README.md`**).
3. **`make clean`**, **`make install`**, verify **B5** on a clean VM.
4. Generate a ports-tree diff and email **ports@openbsd.org** ([porting guide](https://www.openbsd.org/faq/ports/guide.html)).

---

## Updating an existing port (after first submission)

1. **`gmake port-openbsd-sync`** in gghstats.
2. **`cp -r contrib/openbsd/port/*`** → **`/usr/ports/sysutils/gghstats/`**.
3. New distfile in **`$DISTDIR`**, **`make makesum`**, **`make install`**, send updated diff.

---

## Distfile naming

| Field | Pattern (after sync) |
|-------|----------------------|
| **DISTFILES** | `gghstats_<VERSION>_openbsd_<arch>.tar.gz` |
| **MASTER_SITES** | `https://github.com/hrodrig/gghstats/releases/download/v<VERSION>/` |
| **PKGNAME** | `gghstats-<VERSION>` |

No **`v`** in the tarball filename (same as FreeBSD and GoReleaser **`name_template`**).

---

## Common pitfalls (lab notes)

| Symptom | Cause | Fix |
|---------|--------|-----|
| `Could not find bsd.port.mk` | No full **`/usr/ports`** tree | **B1** — git or CVS checkout |
| `CVSROOT "P" must be absolute` | **`cvs -qdP`** merged wrong | Use **`cvs -qd checkout -r OPENBSD_X_Y -P ports`** — see **port/README.md** |
| `cp: ${PORTSDIR}/distfiles/` | **`DISTDIR`** not set / not created | **`export DISTDIR=/usr/ports/distfiles`** and **`mkdir -p`** |
| `comment is too long` | **COMMENT** > 60 chars in port **Makefile** | Shorten (current: *GitHub traffic dashboard and CLI (SQLite)*) |
| `rcctl: service gghstats does not exist` | Stale **`pkg/gghstats`**, no **`pkg/gghstats.rc`**, or **`.tgz` not rebuilt** | **`rm pkg/gghstats`**; **`make clean=package clean`**; **`make package FETCH_PACKAGES=No`**; expect *Installing pkg/gghstats.rc* |
| **`make package`** only *Link to …* | Package cookie / cached **`.tgz`** | **`make clean=package`**, **`rm`** **`.tgz`**, **`make package FETCH_PACKAGES=No`** |
| **`Error: change in plist`** | Old **`/usr/ports/plist/amd64/gghstats-*`** (e.g. had **`etc/rc.d/gghstats`**, new has **`@rcscript`**) | **`rm -f /usr/ports/plist/amd64/gghstats-${ver}`** (lab; set **`ver`** from **`make -V PKGNAME`**) or bump **`REVISION`** for ports@ |
| `rcctl start` fails, manual serve OK | Missing **`gghstats-serve`** | Port **do-install** must install **`bin/gghstats-serve`** |

---

## What is already validated (OpenBSD lab)

- **Ansible** full cycle on an OpenBSD inventory host (tarball + **`contrib/openbsd`** scripts).
- **Ports** full **Part B** redo: **`pkg/gghstats.rc`**, **`make package FETCH_PACKAGES=No`**, **`make install`** → **`/etc/rc.d/gghstats`**, **`rcctl enable`**, no symlink.
