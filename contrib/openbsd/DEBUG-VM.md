# OpenBSD VM debug checklist (gghstats)

Run on an **OpenBSD test host** as **root**. rc.d model: `daemon="/usr/local/bin/gghstats-serve"`, **`rc_bg=YES`** (rc.subr `rc_exec` — no custom `rc_start` with `&`).

After manual checks pass, re-run platform tests from the **control node** (repo root):

```bash
make test-platforms LIMIT=<openbsd-host>
```

Use the host name from **`testing/platforms/inventory/hosts.yml`** (see **`hosts.yml.example`**).

## 0. Binary and config

```sh
which gghstats
gghstats version
ls -l /usr/local/bin/gghstats
ls -l /usr/local/bin/gghstats-serve
ls -l /etc/gghstats/gghstats.env
grep -v GITHUB_TOKEN /etc/gghstats/gghstats.env
ls -ld /var/lib/gghstats
```

## 1. Install rc.d from the repository

On the OpenBSD host (with a clone of the gghstats repo), or copy the files with **`scp`** from the machine where you build:

```sh
install -m 555 /path/to/gghstats/contrib/openbsd/gghstats /etc/rc.d/gghstats
install -m 755 /path/to/gghstats/contrib/openbsd/gghstats-serve /usr/local/bin/gghstats-serve
install -m 755 /path/to/gghstats/contrib/openbsd/gghstats-start /usr/local/bin/gghstats-start
```

Verify rc.d:

```sh
grep -E '^(daemon=|rc_bg=|pexp=)' /etc/rc.d/gghstats
```

Expected:

- `daemon="/usr/local/bin/gghstats-serve"`
- `rc_bg=YES`
- `pexp="/usr/local/bin/gghstats serve"`

**Ports tree install:** `make install` registers **`/etc/rc.d/gghstats`** (from **`pkg/gghstats.rc`** + **`@rcscript`**). Use `rcctl` as usual.

## 2. Clean stop

```sh
rcctl stop gghstats 2>/dev/null
pkill -x gghstats 2>/dev/null
rm -f /var/run/gghstats.pid
```

## 3. Start via rc.d (preferred)

**Show errors:** OpenBSD hides `rc_start` stderr unless you use **`-d`**:

```sh
/etc/rc.d/gghstats -d start
```

```sh
rcctl enable gghstats
rcctl start gghstats
echo "rcctl start rc=$?"
rcctl check gghstats
pgrep -af gghstats
ftp -o - http://127.0.0.1:8080/api/v1/healthz
```

## 4. If rcctl fails, call start directly

```sh
/usr/local/bin/gghstats-serve
# foreground; Ctrl+C when done — shows config errors
```

## 5. Manual helper

```sh
/usr/local/bin/gghstats-start
pgrep -af gghstats
tail -20 /var/lib/gghstats/gghstats.log
```

## 6. Foreground (sanity)

```sh
set -a; . /etc/gghstats/gghstats.env; set +a
gghstats serve
# Ctrl+C when done
```

## Common failures

| Symptom | Cause |
|--------|--------|
| `daemon is not set` | Old `/etc/rc.d/gghstats` without `daemon=` before `rc.subr` |
| `gghstats-serve` exits immediately | Bad `gghstats.env` (FILTER quotes, empty token) |
| `serve` works, rcctl does not | Missing `gghstats-serve` on PATH or rc.d not updated — run step 1 |
| Port install, rcctl fails | Install `gghstats-serve` from port (`bin/gghstats-serve` in PLIST) |

Log file (manual start): **`/var/lib/gghstats/gghstats.log`**.

## 7. Full cleanup

From the **control node** (Ansible):

```bash
cd testing/platforms
ansible-playbook playbooks/teardown.yml --limit <openbsd-host>
# optional: -e gghstats_remove_data_dir=true
```

On the OpenBSD host (root): `rcctl stop gghstats`; `rcctl disable gghstats`; remove `/etc/rc.d/gghstats`, `/usr/local/bin/gghstats`, `/usr/local/bin/gghstats-serve`, `/usr/local/bin/gghstats-start`, `/etc/gghstats`; optionally `rm -rf /var/lib/gghstats`.
