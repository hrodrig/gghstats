# OpenBSD port for gghstats

Port files for submitting gghstats to the official OpenBSD ports tree.

Operator install from tarball: **[../README.md](../README.md)**.

**New maintainers:** **[../BSD-PORTS-STEP-BY-STEP.md](../BSD-PORTS-STEP-BY-STEP.md)** (both BSDs, ordered steps).

**Maintainers:** OpenBSD lab detail → **[../PORT-RELEASE.md](../PORT-RELEASE.md)**. This file is the port-directory reference (CVS/git checkout, `file://` fetch, layout).

## Version bump

From the gghstats repo root, after updating **`VERSION`**, run **`gmake port-openbsd-sync`** (or **`make port-openbsd-sync`** on macOS/Linux with GNU make) to refresh **`DISTNAME`**, **`PKGNAME`**, **`MASTER_SITES`**, and **`DISTFILES`** in this **Makefile**.

## distinfo (not shipped in the gghstats repo)

**`distinfo`** holds **`SHA256`** / **`SIZE`** lines for each **DISTFILES** artifact. Those checksums require the exact tarball bytes (usually from a published GitHub release).

- This skeleton **does not include `distinfo`**. After **`make fetch`** (or a local tarball), run **`make makesum`** in your OpenBSD ports checkout, then include **`distinfo`** in the diff you send to **ports@openbsd.org**.
- **`contrib/openbsd/port/distinfo`** is listed in **`.gitignore`** so a local **`make makesum`** is not committed by mistake.

## Port files vs release tarball

| Source | Installed by port |
|--------|-------------------|
| **DISTFILES** (GitHub / `dist-openbsd`) | `bin/gghstats`, man, LICENSE, `gghstats.env.example` |
| **`files/gghstats-serve`**, **`files/gghstats-start`** | `bin/gghstats-serve`, `bin/gghstats-start` |
| **`pkg/gghstats.rc`** + **`@rcscript ${RCDIR}/gghstats`** in **PLIST** | **`/etc/rc.d/gghstats`** via **`generate-readmes`** (only **`${PKGDIR}/*.rc`** are copied; same as **`www/gitea/pkg/gitea.rc`**) |

Keep **`pkg/gghstats.rc`**, **`files/gghstats-serve`**, and **`files/gghstats-start`** in sync with **`contrib/openbsd/`** (`gmake port-openbsd-sync` copies **`gghstats`** → **`pkg/gghstats.rc`**). Do **not** use **`pkg/gghstats`** (no **`.rc`** suffix → **`generate-readmes`** skips it). Do **not** list **`etc/rc.d/gghstats`** in **PLIST**.

After **`make install`**: copy **`share/examples/gghstats/gghstats.env.example`** → **`/etc/gghstats/gghstats.env`**, edit token/filter, then **`rcctl enable gghstats`** and **`rcctl start gghstats`**.

## Install the full ports tree (test VM)

`make` in **`contrib/openbsd/port/`** alone fails with *Could not find bsd.port.mk* until **`/usr/ports/infrastructure/`** exists. On the OpenBSD host:

**1. Save your port copy** (if you already created **`/usr/ports/sysutils/gghstats`**):

```sh
cp -r /usr/ports/sysutils/gghstats /tmp/gghstats-port
```

**2. Check OpenBSD version** (pick the matching branch):

```sh
uname -r
# e.g. 7.6 → use OPENBSD_7_6 below
```

**3a. Official CVS** (matches installed release; needs network):

```sh
pkg_add -u cvs # if cvs is not installed
export CVSROOT=anoncvs@anoncvs.openbsd.org:/cvs
cd /usr
rm -rf ports
cvs -qd checkout -r OPENBSD_7_6 -P ports
```

Use a **space** after **`-r`** (`-r OPENBSD_7_6`, not `-rOPENBSD_7_6`). Replace **`OPENBSD_7_6`** with **`OPENBSD_X_Y`** for your **`uname -r`**. For **-current**: `cvs -qd checkout -A -P ports`.

**3b. Git mirror** (faster for a lab VM; tree may be newer than the OS):

```sh
pkg_add git
cd /usr
rm -rf ports
git clone --depth 1 https://github.com/openbsd/ports.git ports
```

**4. Restore gghstats port and build:**

```sh
rm -rf /usr/ports/sysutils/gghstats
mkdir -p /usr/ports/sysutils/gghstats
cp -r /tmp/gghstats-port/* /usr/ports/sysutils/gghstats/
export DISTDIR=/usr/ports/distfiles
mkdir -p "$DISTDIR"
cp /tmp/gghstats_0.6.4_openbsd_amd64.tar.gz "$DISTDIR/"
cd /usr/ports/sysutils/gghstats
make makesum
make clean=package clean
make package FETCH_PACKAGES=No
make install
```

See **[../PORT-RELEASE.md](../PORT-RELEASE.md)** Part B for the full validated flow and pitfalls.

**5. Configure and run:**

```sh
mkdir -p /etc/gghstats /var/lib/gghstats
cp /usr/local/share/examples/gghstats/gghstats.env.example /etc/gghstats/gghstats.env
vi /etc/gghstats/gghstats.env
chmod 600 /etc/gghstats/gghstats.env
rcctl enable gghstats
rcctl start gghstats
rcctl check gghstats
ftp -o - http://127.0.0.1:8080/api/v1/healthz
```

**`rcctl: service gghstats does not exist`** — usually **`pkg/gghstats.rc`** missing, stale **`pkg/gghstats`** left after **`cp -r`**, or the **`.tgz` was not rebuilt**. Use **`pkg/gghstats.rc`** + **`@rcscript ${RCDIR}/gghstats`** only. Force rebuild:

```sh
rm -f pkg/gghstats
make clean=package clean
rm -f /usr/ports/packages/amd64/all/gghstats-*.tgz
make package FETCH_PACKAGES=No   # must print: Installing pkg/gghstats.rc as .../etc/rc.d/gghstats
PKG=$(ls /usr/ports/packages/amd64/no-arch/gghstats-*.tgz 2>/dev/null \
   || ls /usr/ports/packages/amd64/all/gghstats-*.tgz)
tar -tzf "$PKG" | grep etc/rc.d/gghstats
```

If **`make package`** only prints *Link to …*, the package was skipped — run **`make clean=package`** again.

**`Error: change in plist`** after fixing **PLIST** / **`pkg/gghstats.rc`** — the ports tree cached the old packing-list. On a lab VM: **`rm -f /usr/ports/plist/${MACHINE_ARCH}/gghstats-*`**, then **`make package FETCH_PACKAGES=No`** again. You should see *Installing pkg/gghstats.rc* and no plist error.

Disk: a full ports tree is large (several GB). A shallow **git** clone is enough to run **`make install`** for one port.

## Layout

Copy this directory to **`/usr/ports/sysutils/gghstats/`**:

```bash
# On OpenBSD
cd /usr/ports
mkdir -p sysutils/gghstats
cp -r /path/to/gghstats/contrib/openbsd/port/* sysutils/gghstats/
cd sysutils/gghstats
```

## Build and test

```bash
make fetch      # needs distfile (GitHub release or local; see below)
make makesum    # writes distinfo from fetched files
make clean=package clean
make package FETCH_PACKAGES=No
make install
```

After **`make install`**, expect *The following new rcscripts were installed: /etc/rc.d/gghstats*.

## Test with a local tarball (before a GitHub release)

The port normally downloads **DISTFILES** from **MASTER_SITES** (GitHub releases). To validate **install** / **plist** / **rc.d** against a tarball built locally:

1. **Match the filename** in **DISTFILES** for your architecture (e.g. `gghstats_0.6.4_openbsd_amd64.tar.gz` after **`gmake port-openbsd-sync`**).

   From repo root:

   ```bash
   gmake port-openbsd-sync
   gmake dist-openbsd
   # Optional arch override:
   # gmake dist-openbsd OPENBSD_ARCH=arm64
   ```

   Output: **`dist/gghstats_<version>_openbsd_<arch>.tar.gz`** (no **`v`** in the filename; same as FreeBSD / GoReleaser).

2. **Option A — copy into DISTDIR:** From the port directory, run **`make show=DISTDIR`**, copy your tarball there with the **exact** name **DISTFILES** expects, then **`make checksum`** (or **`make makesum`**), then **`make install`**.

3. **Option B — `file://` override:**

   ```bash
   cd /usr/ports/sysutils/gghstats
   cp /path/to/gghstats/dist/gghstats_0.6.4_openbsd_amd64.tar.gz /tmp/gghstats-dist/
   make fetch MASTER_SITES=file:///tmp/gghstats-dist/
   make install
   ```

   Use an **absolute** path after **`file://`**. Do not commit **`MASTER_SITES=file://...`** to the official ports tree.

## Submit to OpenBSD

1. Make changes in a checkout of the ports tree.
2. Generate a diff: **`cvs diff -u`** / **`git diff`** for the ports repo.
3. Send to **ports@openbsd.org** with a descriptive subject.

See the [OpenBSD Porting Guide](https://www.openbsd.org/faq/ports/guide.html).
