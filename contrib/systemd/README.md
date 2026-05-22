# systemd unit for gghstats

Run **`gghstats serve`** under systemd on Linux. Configuration is an **environment file** at `/etc/gghstats/gghstats.env`.

**One instance = one SQLite file and one filter.** For multiple isolated dashboards, use separate env files, units, and data directories (e.g. `gghstats-team-a.service` + `/etc/gghstats/team-a.env`).

## Prerequisites

1. **Environment file** — **.deb/.rpm** install `contrib/gghstats.env.example` to **`/etc/gghstats/gghstats.env`**. From source:

   ```bash
   sudo mkdir -p /etc/gghstats /var/lib/gghstats
   sudo cp contrib/gghstats.env.example /etc/gghstats/gghstats.env
   # Edit: GGHSTATS_GITHUB_TOKEN, GGHSTATS_FILTER, GGHSTATS_DB, etc.
   sudo chmod 600 /etc/gghstats/gghstats.env
   ```

2. **Binary path** — Units use **`/usr/bin/gghstats`** (where .deb/.rpm install). For manual install to `/usr/local/bin`, edit **`ExecStart`** in the unit.

3. **Data directory** — Default **`/var/lib/gghstats`** (`WorkingDirectory` + `GGHSTATS_DB=/var/lib/gghstats/gghstats.db`). Create and own it if using a dedicated user:

   ```bash
   sudo chown -R gghstats:gghstats /var/lib/gghstats
   ```

## Files

| Unit | Function |
|------|----------|
| `gghstats.service` | Daemon — HTTP UI, scheduled GitHub sync, SQLite storage |

**.deb/.rpm** install the unit to **`/lib/systemd/system/`**. Skip the `cp` step below; enable and start only.

Units order after **`network.target`** (not `network-online.target`). That avoids **`systemctl enable --now`** appearing to hang while systemd waits for **`systemd-networkd-wait-online`** (common on minimal or static-IP installs). GitHub API calls still need working DNS/routing before sync succeeds.

## Quick test (before enabling)

```bash
# Validate token and filter without enabling systemd (uses your shell env or a copy of the env file)
set -a && source /etc/gghstats/gghstats.env && set +a
gghstats serve
# Ctrl+C when OK, then enable the unit
```

## Daemon mode

```bash
# .deb/.rpm: units already in /lib/systemd/system/
# From source:
# sudo cp contrib/systemd/gghstats.service /etc/systemd/system/
# If manual install: edit ExecStart=/usr/local/bin/gghstats serve
sudo systemctl daemon-reload
sudo systemctl enable --now gghstats
journalctl -u gghstats -f
```

## Optional: dedicated user

```bash
sudo useradd -r -d /var/lib/gghstats -s /usr/sbin/nologin gghstats
sudo chown -R gghstats:gghstats /var/lib/gghstats
sudo chmod 600 /etc/gghstats/gghstats.env
sudo chown root:gghstats /etc/gghstats/gghstats.env   # or root:root if only root reads secrets
```

Uncomment in **`gghstats.service`**:

```conf
User=gghstats
Group=gghstats
```

Ensure **`gghstats`** can read **`/etc/gghstats/gghstats.env`** and read/write **`/var/lib/gghstats`**.

## Bind address

| `GGHSTATS_HOST` | Use when |
|-----------------|----------|
| `127.0.0.1` | Reverse proxy (nginx, Traefik, Caddy) on the same host — **recommended** for servers |
| `0.0.0.0` | Direct access on all interfaces (firewall carefully) |

| Public HTTPS, Traefik, Compose stacks | **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** — preferred for production |
| Bare metal + systemd (this unit) | `127.0.0.1` + optional reverse proxy on the same host |
| Dev Docker in the **gghstats** repo | Local smoke test only — not production |

**gghstats-selfhosted** Compose sets **`0.0.0.0`** inside the container; that is separate from this bare-metal unit.

## Troubleshooting

### `systemctl enable --now` seems stuck; service stays `inactive (dead)`; empty journal

Often **`network-online.target`** on an older copied unit. Use current **`contrib/systemd/gghstats.service`** (`network.target`), then:

```bash
sudo systemctl daemon-reload
sudo systemctl reset-failed gghstats.service
sudo systemctl start gghstats.service
sudo systemctl status gghstats.service
```

**Drop-in override** without replacing the unit:

```bash
sudo mkdir -p /etc/systemd/system/gghstats.service.d
sudo tee /etc/systemd/system/gghstats.service.d/override.conf << 'EOF'
[Unit]
After=network.target
Wants=network.target
EOF
sudo systemctl daemon-reload
sudo systemctl reset-failed gghstats.service
sudo systemctl start gghstats.service
```

### Service fails immediately: token or permissions

- **`GGHSTATS_GITHUB_TOKEN is required`** — set token in **`/etc/gghstats/gghstats.env`**.
- **SQLite permission denied** — fix ownership of **`/var/lib/gghstats`** for the unit **`User=`**.
- **`ExecStart` not found** — binary is under **`/usr/local/bin`**; update the unit or symlink to **`/usr/bin/gghstats`**.

### No logs until the first start

`journalctl -u gghstats` stays empty until systemd has started the process at least once.
