# BSD ports — step-by-step guide (new maintainers)

This document explains **how gghstats FreeBSD and OpenBSD ports are produced and tested**, in order, without assuming you have ported software before.

| If you want… | Read |
|--------------|------|
| **This guide** (first time / recap) | You are here |
| FreeBSD operator install | [freebsd/README.md](freebsd/README.md) |
| FreeBSD Bugzilla / release checklist | [freebsd/PORT-RELEASE.md](freebsd/PORT-RELEASE.md) |
| OpenBSD tarball install | [openbsd/README.md](openbsd/README.md) |
| OpenBSD port lab + pitfalls | [openbsd/PORT-RELEASE.md](openbsd/PORT-RELEASE.md) |
| OpenBSD port directory reference | [openbsd/port/README.md](openbsd/port/README.md) |
| Ansible smoke (tarball, not port) | [../testing/platforms/README.md](../testing/platforms/README.md) |

---

## 1. Big picture (three layers)

Do not mix these up:

| Layer | What it is | Where it lives |
|-------|------------|----------------|
| **Application** | Go binary + man page + env example | Built in the **gghstats** repo; **GoReleaser** (or `gmake dist-*`) publishes a **tarball** on GitHub Releases |
| **Port recipe** | Instructions: “download tarball X, install files Y, register service Z” | **`contrib/freebsd/`** (FreeBSD) · **`contrib/openbsd/port/`** (OpenBSD) |
| **Ports tree** | OS build system that turns the recipe + tarball into a **native package** (`.pkg` / `.tgz`) | **`/usr/ports/sysutils/gghstats/`** on a FreeBSD or OpenBSD VM |

The port **does not compile Go**. It **fetches a release tarball** and runs **`do-install`** (plus plist / rc scripts).

```text
  VERSION (repo root)
       │
       ├─ gmake port-freebsd-sync  ──► contrib/freebsd/Makefile (PORTVERSION)
       ├─ gmake port-openbsd-sync  ──► contrib/openbsd/port/Makefile + synced files/
       │
       ├─ gmake dist-freebsd       ──► dist/gghstats_<ver>_freebsd_<arch>.tar.gz
       ├─ gmake dist-openbsd       ──► dist/gghstats_<ver>_openbsd_<arch>.tar.gz
       │     (or GoReleaser on tag v<ver> — same filenames)
       │
       └─ copy contrib/* port ──► /usr/ports/sysutils/gghstats/
                │
                make makesum  (checksums distfile)
                make install  (or make package + pkg_add)
                │
                └─► native package: binaries, man, /etc/rc.d or rc.d, examples
```

---

## 2. Two different `make` commands (read this once)

| Where you are | Command | Examples |
|---------------|---------|----------|
| **gghstats repo root** (Mac, Linux, or FreeBSD) | **`gmake`** (GNU Make) | `port-freebsd-sync`, `port-openbsd-sync`, `dist-freebsd`, `dist-openbsd`, `make snapshot` |
| **Ports tree** on FreeBSD/OpenBSD | **`make`** (BSD Make) | `makesum`, `install`, `package`, `clean`, `portlint` (FreeBSD) |

On FreeBSD, plain `make port-freebsd-sync` **inside the gghstats repo** fails with syntax errors — use **`pkg install gmake`** and **`gmake`**.

---

## 3. Naming rules (same for both BSDs)

| Field | Example | Notes |
|-------|---------|--------|
| Git tag | `v0.6.4` | Annotated tag on **`main`** after release |
| Repo **`VERSION`** | `0.6.4` | **No** `v` — single source of truth at repo root |
| Tarball on GitHub | `gghstats_0.6.4_freebsd_amd64.tar.gz` | **No** `v` in the filename |
| | `gghstats_0.6.4_openbsd_amd64.tar.gz` | Same pattern |
| Download URL | `https://github.com/hrodrig/gghstats/releases/download/v0.6.4/…` | Tag **has** `v`; filename **does not** |

After every **`VERSION`** bump, run **`gmake port-freebsd-sync`** and/or **`gmake port-openbsd-sync`** so port Makefiles match GoReleaser (see root **`.goreleaser.yaml`**).

---

## 4. Scenario A — First lab test (no GitHub release yet)

Use this when you changed packaging and want to validate on a VM **before** tagging.

### Step A1 — On your build machine (Mac or Linux)

```bash
cd /path/to/gghstats

# 1) Confirm version
cat VERSION                    # e.g. 0.6.4

# 2) Sync port Makefiles from VERSION
gmake port-freebsd-sync        # FreeBSD
gmake port-openbsd-sync        # OpenBSD (also copies rc scripts into port/files/)

# 3) Build distfiles locally (same layout as GoReleaser)
gmake dist-freebsd             # → dist/gghstats_<VERSION>_freebsd_<arch>.tar.gz
gmake dist-openbsd             # → dist/gghstats_<VERSION>_openbsd_amd64.tar.gz
# Optional: OPENBSD_ARCH=arm64 gmake dist-openbsd

ls dist/gghstats_*
```

You need **`go`** on the build host. You do **not** need a ports tree on the build host.

### Step A2 — FreeBSD VM

**One-time:** install **`gmake`** and **`go`** if you build distfiles on the VM; clone or install a **ports tree** (`/usr/ports` or `~/ports`).

```bash
# 1) Copy port recipe from your machine (or git pull on VM)
mkdir -p /usr/ports/sysutils/gghstats
cp -r /path/to/gghstats/contrib/freebsd/* /usr/ports/sysutils/gghstats/

# 2) Put distfile where the port expects it
cp /path/to/gghstats/dist/gghstats_*_freebsd_*.tar.gz \
   "$(make -C /usr/ports/sysutils/gghstats -V DISTDIR)/"

# 3) Build and install (BSD make — not gmake)
cd /usr/ports/sysutils/gghstats
make makesum
make install
portlint          # optional but recommended before Bugzilla

# 4) Configure and smoke
cp /usr/local/etc/gghstats/gghstats.env.example /etc/gghstats/gghstats.env
vi /etc/gghstats/gghstats.env    # GGHSTATS_GITHUB_TOKEN, GGHSTATS_FILTER
sysrc gghstats_enable=YES
service gghstats start
gghstats version
curl -s http://127.0.0.1:8080/api/v1/healthz
```

**Operator equivalent after the port is official:** `pkg install gghstats` (not available until the port is in the FreeBSD tree).

### Step A3 — OpenBSD VM

**One-time:** shallow **`/usr/ports`** checkout — see [openbsd/port/README.md](openbsd/port/README.md) or **Part B1** in [openbsd/PORT-RELEASE.md](openbsd/PORT-RELEASE.md).

From the **build host**, copy **only** `contrib/openbsd/port/` (not all of `contrib/openbsd/`):

```bash
rm -rf /tmp/gghstats-port
cp -r contrib/openbsd/port /tmp/gghstats-port
scp -r /tmp/gghstats-port/* root@openbsd-vm:/tmp/gghstats-port/
scp dist/gghstats_*_openbsd_amd64.tar.gz root@openbsd-vm:/tmp/
```

On the **OpenBSD VM**:

```sh
mkdir -p /usr/ports/sysutils/gghstats
cp -r /tmp/gghstats-port/* /usr/ports/sysutils/gghstats/

export DISTDIR=/usr/ports/distfiles
mkdir -p "$DISTDIR"
dist=$(make -C /usr/ports/sysutils/gghstats -V DISTFILES)
cp /tmp/$dist "$DISTDIR/"

cd /usr/ports/sysutils/gghstats
make makesum
make clean=package clean
make package FETCH_PACKAGES=No    # must log "Building package for gghstats-…"
make install                      # expect: new rcscripts … /etc/rc.d/gghstats

mkdir -p /etc/gghstats /var/lib/gghstats
cp /usr/local/share/examples/gghstats/gghstats.env.example /etc/gghstats/gghstats.env
vi /etc/gghstats/gghstats.env
chmod 600 /etc/gghstats/gghstats.env
rcctl enable gghstats
rcctl start gghstats
rcctl check gghstats
ftp -o - http://127.0.0.1:8080/api/v1/healthz
```

**Operator equivalent after the port is official:** `pkg_add gghstats` (not available until the port is in the OpenBSD tree).

**Lab shortcut (no full ports tree):** [openbsd/port/test-install-from-dist.sh](openbsd/port/test-install-from-dist.sh) mimics **do-install** — useful for debugging, not a substitute for **Part B** in PORT-RELEASE.

---

## 5. Scenario B — Release maintainer (tag on GitHub)

Do this when preparing an **official** app release.

### Step B1 — In the gghstats repo (before tag)

1. Bump **`VERSION`**, **CHANGELOG**, README badge, **`contrib/man/man1/gghstats.1`** (see root **`AGENTS.md`**).
2. Sync ports:

   ```bash
   gmake port-freebsd-sync
   gmake port-openbsd-sync
   ```

3. Run **`make release-check`**, merge **`develop` → `main`**, tag **`v<VERSION>`**, run **`make release`** (GoReleaser).
4. On GitHub Releases, confirm assets exist, for example:
   - `gghstats_0.6.4_freebsd_amd64.tar.gz`
   - `gghstats_0.6.4_openbsd_amd64.tar.gz`
   - `.deb`, `.rpm`, Linux tarballs, etc.

### Step B2 — Validate ports against **published** tarballs

On each BSD VM, copy refreshed port files from **`contrib/`**, then:

```sh
cd /usr/ports/sysutils/gghstats
make clean
make fetch          # downloads from MASTER_SITES (GitHub) — OR copy distfile to DISTDIR
make makesum        # if distfile changed
make install
```

You **do not** need `gmake dist-*` on your laptop if GoReleaser already published the same tarball names.

### Step B3 — Submit to official trees (optional)

| OS | Submit to | Details |
|----|-----------|---------|
| FreeBSD | [Bugzilla](https://bugs.freebsd.org/) | Git diff preferred — [freebsd/PORT-RELEASE.md](freebsd/PORT-RELEASE.md) |
| OpenBSD | ports@openbsd.org | Ports-tree diff includes generated **`distinfo`** — [openbsd/PORT-RELEASE.md](openbsd/PORT-RELEASE.md) |

Until submission is accepted, **`pkg install gghstats`** / **`pkg_add gghstats`** from **official mirrors will not work**. Use the lab steps above.

---

## 6. What each `gmake` target does (repo root)

| Target | What it updates / builds |
|--------|---------------------------|
| **`port-freebsd-sync`** | Sets **`PORTVERSION=`** in **`contrib/freebsd/Makefile`** from **`VERSION`** |
| **`port-openbsd-sync`** | Sets **`DISTNAME`**, **`PKGNAME`**, **`MASTER_SITES`**, **`DISTFILES`** in **`contrib/openbsd/port/Makefile`**; copies **`gghstats`**, **`gghstats-serve`**, **`gghstats-start`** into **`port/files/`** and **`pkg/gghstats.rc`** |
| **`dist-freebsd`** | Builds **`dist/gghstats_<VERSION>_freebsd_<arch>.tar.gz`** (native arch, or cross via **`GOOS`/`GOARCH`**) |
| **`dist-openbsd`** | Builds **`dist/gghstats_<VERSION>_openbsd_<arch>.tar.gz`** (**`OPENBSD_ARCH`**, default **`amd64`**) |

---

## 7. FreeBSD vs OpenBSD — cheat sheet

| Topic | FreeBSD | OpenBSD |
|-------|---------|---------|
| Port files in repo | **`contrib/freebsd/`** | **`contrib/openbsd/port/`** |
| Service script | **`contrib/freebsd/rc.d/gghstats`** → **`/usr/local/etc/rc.d/gghstats`** | **`pkg/gghstats.rc`** + **`@rcscript`** → **`/etc/rc.d/gghstats`** |
| Serve wrapper | env in **`rc.d`** / daemon | **`/usr/local/bin/gghstats-serve`** (required for **`rcctl`**) |
| Enable / start | **`sysrc gghstats_enable=YES`**, **`service gghstats start`** | **`rcctl enable gghstats`**, **`rcctl start gghstats`** |
| Build package | **`make install`** (often enough in lab) | **`make package FETCH_PACKAGES=No`** then **`make install`** |
| Lint | **`portlint`** | (manual checklist in PORT-RELEASE) |
| **`distinfo` in gghstats repo? | No — generated in ports tree | No — **`contrib/openbsd/port/distinfo`** is gitignored |

---

## 8. Ansible platform tests vs port validation

These are **different paths**:

| Path | Command / doc | What it proves |
|------|----------------|----------------|
| **Ansible** (tarball + repo scripts) | **`make test-platforms`** — [testing/platforms/README.md](../testing/platforms/README.md) | Release tarball installs; **rc.d copied from your gghstats clone**; healthz; uninstall |
| **Port lab** (this guide) | **`make install`** in **`/usr/ports/sysutils/gghstats`** | Port **Makefile**, **PLIST**, **`pkg_add`/`pkg delete`**, **`@rcscript`** — what operators get **after** the port is published |

**Planned (not implemented yet):** Ansible mode **`install_method: port_pkg`** — build/install via port using the **GitHub distfile** without manual `tar xzf`. Until then, use this guide for port smoke and Ansible for fast multi-OS regression.

---

## 9. Common mistakes (newbies)

| Mistake | Fix |
|---------|-----|
| Running **`make port-freebsd-sync`** in app repo on FreeBSD | Use **`gmake`** |
| Running **`gmake makesum`** inside **`contrib/freebsd/`** | Copy port into **`/usr/ports/...`**, then BSD **`make makesum`** |
| **`VERSION`** out of sync with port Makefile | Always **`gmake port-*-sync`** after bumping **`VERSION`** |
| OpenBSD: copied all of **`contrib/openbsd/`** into ports | Copy **only** **`contrib/openbsd/port/*`** |
| OpenBSD: **`make package`** only prints *Link to …* | **`make clean=package clean`**, then **`make package FETCH_PACKAGES=No`** |
| OpenBSD: **`Error: change in plist`** | Remove **`/usr/ports/plist/amd64/gghstats-<ver>`** on lab VM (see PORT-RELEASE) |
| Expect **`pkg_add gghstats`** to work today | Port is **not** in official trees yet — lab install only |
| Tarball filename with **`v`** prefix | Filename is **`gghstats_0.6.4_…`**, tag is **`v0.6.4`** |

---

## 10. Quick decision tree

```text
Need to test packaging before tag?
  → Scenario A: gmake port-*-sync + dist-* → copy to VM → make makesum install

Tagged and GoReleaser published BSD tarballs?
  → Scenario B: refresh contrib/ on VM → make fetch (or DISTDIR) → makesum → install

Want automated multi-OS smoke without ports tree?
  → make test-platforms (tarball path; see testing/platforms README)

Ready to publish port to FreeBSD/OpenBSD official tree?
  → PORT-RELEASE.md for your OS → Bugzilla or ports@
```

---

## 11. File map (where things live)

```text
gghstats/
├── VERSION                          ← bump first
├── Makefile                         ← port-*-sync, dist-* targets
├── contrib/
│   ├── BSD-PORTS-STEP-BY-STEP.md    ← this file
│   ├── freebsd/
│   │   ├── Makefile                 ← FreeBSD port (PORTVERSION, MASTER_SITES, DISTFILES)
│   │   ├── pkg-plist, pkg-descr, rc.d/
│   │   ├── README.md, PORT-RELEASE.md
│   └── openbsd/
│       ├── gghstats, gghstats-serve, gghstats-start   ← source scripts
│       ├── README.md, PORT-RELEASE.md, DEBUG-VM.md
│       └── port/
│           ├── Makefile             ← OpenBSD port
│           ├── pkg/PLIST, pkg/gghstats.rc, pkg/DESCR
│           └── files/               ← synced by port-openbsd-sync
└── dist/                            ← local distfiles (gitignored); gmake dist-* / snapshot
```
