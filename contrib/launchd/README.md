# launchd (macOS) for gghstats

Run **`gghstats serve`** in the background on macOS with a **LaunchAgent** (user session, no root). Linux servers use **[systemd](../systemd/README.md)** instead.

**Default SQLite path for local macOS** is still **`./data/gghstats.db`** relative to **`WorkingDirectory`** until **`~/.config/gghstats/`** becomes the default (planned before **v1.0.0**). Set **`GGHSTATS_DB`** explicitly in your env file — the example below uses **`~/Library/Application Support/gghstats/`**.

## Files

| File | Purpose |
|------|---------|
| `gghstats-serve.sh` | Wrapper: loads **`~/.gghstats.env`**, then **`exec gghstats serve`** |
| `com.github.hrodrig.gghstats.plist` | LaunchAgent template (edit **`REPLACE_WITH_HOME`** paths) |

## Quick setup

```bash
mkdir -p ~/bin ~/Library/Application\ Support/gghstats ~/Library/Logs
cp contrib/gghstats.env.example ~/.gghstats.env
chmod 600 ~/.gghstats.env
# Edit token, filter, and paths — example local DB:
#   GGHSTATS_DB=$HOME/Library/Application Support/gghstats/gghstats.db

cp contrib/launchd/gghstats-serve.sh ~/bin/
chmod +x ~/bin/gghstats-serve.sh

sed "s|REPLACE_WITH_HOME|$HOME|g" contrib/launchd/com.github.hrodrig.gghstats.plist \
  > ~/Library/LaunchAgents/com.github.hrodrig.gghstats.plist

launchctl bootstrap "gui/$(id -u)" ~/Library/LaunchAgents/com.github.hrodrig.gghstats.plist
open http://127.0.0.1:8080
```

Logs: **`~/Library/Logs/gghstats.log`** and **`gghstats.error.log`**.

## Control

```bash
launchctl bootout "gui/$(id -u)" ~/Library/LaunchAgents/com.github.hrodrig.gghstats.plist
launchctl bootstrap "gui/$(id -u)" ~/Library/LaunchAgents/com.github.hrodrig.gghstats.plist
```

## Homebrew

After **`brew install hrodrig/gghstats/gghstats`**, **`gghstats`** is on **`PATH`**. The wrapper and plist above still apply; there is **no** `brew services` stanza in the cask yet — use LaunchAgent or run interactively:

```bash
export GGHSTATS_GITHUB_TOKEN=ghp_xxx
gghstats run --open
```

**`run`** is an alias for **`serve`**; **`--open`** (or **`GGHSTATS_OPEN_BROWSER=1`**) opens the default browser when the HTTP server is ready.

## Production on macOS

For TLS, reverse proxy, or multi-user hosting, use **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)**. This LaunchAgent is for **local / single-user** use on a Mac.

## See also

- **[contrib/systemd/README.md](../systemd/README.md)** — Linux `.deb`/`.rpm` and **`systemctl enable --now gghstats`**
- **[contrib/gghstats.env.example](../gghstats.env.example)** — all **`GGHSTATS_*`** variables
