# OpenBSD support for gghstats

**New maintainers:** step-by-step FreeBSD + OpenBSD port workflow → **[../BSD-PORTS-STEP-BY-STEP.md](../BSD-PORTS-STEP-BY-STEP.md)**.

[OpenBSD](https://www.openbsd.org) uses **rc.d**, not systemd. This layout uses a **serve wrapper** plus **`rc_bg=YES`** so **`rc.subr`** backgrounds the daemon (no custom `rc_start` with `&`):

| Piece | Role |
|--------|------|
| **`/usr/local/bin/gghstats-serve`** | ksh: `. /etc/gghstats/gghstats.env`, then `exec gghstats serve` |
| **`/etc/rc.d/gghstats`** | `daemon=gghstats-serve`, **`rc_bg=YES`** — `rc.subr` backgrounds via **`rc_exec`** (do not use a custom `rc_start` with `&`) |
| **`pexp`** | `/usr/local/bin/gghstats serve` (after `exec`, `rc_check` matches the real process) |

Optional in **`/etc/rc.conf.local`**:

```sh
gghstats_logger="daemon.info"   # logs via logger(1)
# gghstats_env="EXTRA=value"    # rare; use gghstats.env for normal config
```

## Install (from tarball)

```sh
tar xzf gghstats_*_openbsd_amd64.tar.gz
doas install -m755 gghstats /usr/local/bin/
doas install -m755 gghstats-serve /usr/local/bin/   # if shipped in tarball
doas install -m555 share/openbsd/rc.d/gghstats /etc/rc.d/gghstats

doas mkdir -p /etc/gghstats /var/lib/gghstats
doas cp etc/gghstats/gghstats.env.example /etc/gghstats/gghstats.env
doas vi /etc/gghstats/gghstats.env
doas chmod 600 /etc/gghstats/gghstats.env

doas rcctl enable gghstats
doas rcctl start gghstats
```

**From repo:**

```sh
doas install -m755 contrib/openbsd/gghstats-serve /usr/local/bin/gghstats-serve
doas install -m555 contrib/openbsd/gghstats /etc/rc.d/gghstats
```

## Env file (ksh)

Quote values that contain `!`:

```sh
GGHSTATS_FILTER='your-github-user/*,!fork,!archived'
GGHSTATS_GITHUB_TOKEN='ghp_...'
```

## Health check

```sh
curl -s http://127.0.0.1:8080/api/v1/healthz
```

## Debug

```sh
doas /etc/rc.d/gghstats -d start
doas /usr/local/bin/gghstats-serve    # foreground; shows config errors
```

VM checklist: **`contrib/openbsd/DEBUG-VM.md`**.

## Port (OpenBSD ports tree)

The port installs **`gghstats`**, **`gghstats-serve`**, and **`gghstats-start`**. **`port/pkg/gghstats.rc`** + **`@rcscript`** in **PLIST** install **`/etc/rc.d/gghstats`** for **`rcctl`** (not **`/usr/local/etc/rc.d`**).

Configure **`/etc/gghstats/gghstats.env`**, then:

```sh
rcctl enable gghstats
rcctl start gghstats
```

Optional boot: add **`gghstats`** to **`pkg_scripts`** in **`/etc/rc.conf.local`**.

**Tarball manual install** and **port install** both register **`/etc/rc.d/gghstats`** (port via **`pkg/gghstats.rc`** and **`generate-readmes`**).

**Maintainers:** **[PORT-RELEASE.md](PORT-RELEASE.md)** (release bump + lab `make install`). **Port directory details:** **[port/README.md](port/README.md)**.
