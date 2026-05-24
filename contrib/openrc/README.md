# OpenRC init script for gghstats (Alpine Linux)

Alpine uses **OpenRC**, not systemd. This script runs **`gghstats serve`** as a background daemon, loading **`/etc/gghstats/gghstats.env`** in `start_pre`.

## Install (manual, from linux tarball)

```sh
install -m 755 contrib/openrc/gghstats.initd /etc/init.d/gghstats
mkdir -p /etc/gghstats /var/lib/gghstats
cp contrib/gghstats.env.example /etc/gghstats/gghstats.env
# edit token, filter, port, etc.

rc-service gghstats start
rc-update add gghstats default
```

Release tarballs ship the script as **`share/openrc/gghstats.initd`** (copy to `/etc/init.d/gghstats`).

## Commands

| Command | Action |
|---------|--------|
| `rc-service gghstats start` | Start daemon |
| `rc-service gghstats stop` | Stop daemon |
| `rc-service gghstats status` | Check status |
| `rc-update add gghstats default` | Enable on boot |
| `rc-update del gghstats default` | Disable on boot |

## Health check

```sh
wget -qO- http://127.0.0.1:8080/api/v1/healthz
```

(`wget` is on minimal Alpine; adjust port if `GGHSTATS_PORT` differs.)
