# Platform testing for gghstats

Automated multi-platform validation of **native** gghstats installation: packages or release tarballs, environment file, init system (systemd or rc.d), HTTP health check, and clean uninstall — using Ansible on **real machines** (VPS, lab VMs).

**Docker Compose and Helm** are validated in **[gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)** (`testing/platforms/`, `testing/kind/`), not here.

## Scope

| Suite | What it validates |
|-------|-------------------|
| **This directory** | **`.deb` / `.rpm` / OS tarball**; `/etc/gghstats/gghstats.env`; **`/var/lib/gghstats`**; **systemd** (most Linux), **OpenRC** (Alpine), or **rc.d** (FreeBSD/OpenBSD); daemon up; **`GET /api/v1/healthz`**; uninstall |
| **`make test`** (repo root) | Go unit tests |
| **gghstats-selfhosted** | Compose on Linux VPS, Helm on kind |

**Not in scope:** Traefik, observability stacks, in-cluster Kubernetes, or running gghstats inside Docker on the target host.

**Target OS:** Linux with Docker **not** required. **\*BSD** hosts use release tarballs and rc.d — there is no Docker-based platform test for BSD in this repo.

## Supported platforms

| Platform | Install method | Init system |
|----------|----------------|-------------|
| Debian / Ubuntu | `.deb` | systemd |
| AlmaLinux / RHEL family | `.rpm` (dnf) | systemd |
| FreeBSD | tarball | rc.d (`service`) |
| OpenBSD | tarball | rc.d (`rcctl`) |
| Arch Linux | linux tarball | systemd (unit from Ansible template) |
| OpenSUSE | `.rpm` (zypper) | systemd |
| Alpine Linux | linux tarball | OpenRC (`contrib/openrc/gghstats.initd`) |

**Example lab layout:** `inventory/hosts.yml.example` lists eight targets (Debian, Ubuntu, AlmaLinux, OpenSUSE, Arch, Alpine, FreeBSD, OpenBSD). NAT SSH ports often follow **`22` + last two digits of VMID** (e.g. VM **11098** → port **2298**). Set **`ansible_host_lab`** (or per-host **`ansible_host`**) to your SSH bastion or public IP. **Not supported** here: NetBSD, DragonFly, Illumos, Solaris (no gghstats release tarball).

Extend with new **`platform_vars/<name>.yml`** and inventory hosts (see playbooks and roles under this directory).

## Prerequisites

### Control node (laptop or CI runner with SSH)

- **Ansible** 2.14+
- SSH key access to targets (**`root`** or a user with **passwordless `sudo`**)

### Each target host

1. **Git** (install role may install it on Debian/Ubuntu when missing).
2. **Python 3** for Ansible modules (install role runs `apk add python3` on Alpine when missing).
3. **\*BSD:** set `ansible_python_interpreter` in inventory (`/usr/local/bin/python3` on FreeBSD/OpenBSD).
4. **GitHub PAT** with **read** access to repos in **`gghstats_filter`** — set in inventory as **`gghstats_github_token`** (never commit).

Linux **`.deb`/`.rpm`** hosts do not need Docker.

## Quick start

```bash
cd /path/to/gghstats
cp testing/platforms/inventory/hosts.yml.example testing/platforms/inventory/hosts.yml
# Edit: ansible_host, gghstats_github_token, gghstats_filter, optional local packages

make test-platforms-ping
make test-platforms

# Single host
make test-platforms LIMIT=gghstats-freebsd
```

Or from `testing/platforms/`:

```bash
ansible-playbook playbooks/full-cycle.yml
ansible-playbook playbooks/full-cycle.yml --limit gghstats-ubuntu
```

## Playbooks

| Playbook | Description |
|----------|-------------|
| `playbooks/ping.yml` | SSH + remote Python (`ansible.builtin.ping` → `pong`) |
| `playbooks/setup.yml` | Install, deploy env, start daemon |
| `playbooks/test.yml` | HTTP health check; optional one-shot `gghstats fetch` |
| `playbooks/teardown.yml` | Stop, uninstall, verify cleanup |
| `playbooks/full-cycle.yml` | setup → test → teardown |

Leave the stack running after a successful test:

```bash
cd testing/platforms
ansible-playbook playbooks/setup.yml playbooks/test.yml
```

## Inventory

Copy **`inventory/hosts.yml.example`** to **`inventory/hosts.yml`** (gitignored). Required:

- **`ansible_host`**, **`ansible_port`**, **`ansible_user`**
- **`gghstats_github_token`** — PAT (never commit)
- **`gghstats_filter`** — e.g. `your-user/*,!fork,!archived`
- Optional: **`gghstats_version`**, **`gghstats_package_source`** (`auto` \| `release` \| `local`), **`gghstats_local_package`**, **`gghstats_port`**, **`gghstats_sync_on_startup`** (default `false` for faster smoke tests), **`gghstats_test_fetch_repo`** (e.g. `owner/repo` for optional API fetch test)

**Package source:**

- **`local`** (default in `group_vars/all.yml`): install from **`dist/`** on the control node. No GitHub release required.
- **`auto`**: use **`dist/`** only if the artifact file exists (or set **`gghstats_local_package`** per host); otherwise download from **`gghstats_release_url`**.
- **`release`**: always download (requires published **`v{{ gghstats_version }}`** assets).

### Local snapshot (before a GitHub release)

1. From the repo root: **`make snapshot`** (writes **`dist/`** and **`dist/metadata.json`**).
2. Set **`gghstats_version`** in `hosts.yml` to the **`version`** field from **`dist/metadata.json`** (e.g. `0.6.4-next`, not the git tag).
3. Set **`gghstats_package_source: local`** (default in **`inventory/group_vars/all.yml`**).
4. The install role resolves paths automatically:
   - Debian/Ubuntu → `dist/gghstats_<version>_linux_amd64.deb`
   - AlmaLinux / OpenSUSE → `..._linux_amd64.rpm`
   - Arch → `..._linux_amd64.tar.gz`
   - FreeBSD / OpenBSD → `dist/gghstats_<version>_<os>_amd64.tar.gz`
5. Optional **`gghstats_dist_dir`**: absolute path if you run Ansible outside the clone.
6. **`make test-platforms-ping`**, then **`make test-platforms`**.

Override a single host: **`gghstats_local_package: "/path/to/custom.deb"`**.

After you publish **`v<version>`** on GitHub, switch to **`gghstats_package_source: release`** and **`gghstats_version: "<version>"`** (semver without `v`).

Release asset names match GoReleaser: `gghstats_<version>_linux_amd64.deb`, `gghstats_<version>_freebsd_amd64.tar.gz`, etc. (no `v` in the filename; Git tag is `v<version>`).

## Test flow

```
setup.yml     → install → configure (/etc/gghstats, /var/lib/gghstats) → daemon (started)
test.yml      → GET /api/v1/healthz [→ optional gghstats fetch]
teardown.yml  → uninstall → assert binary/config/process gone (OpenBSD: also removes gghstats-serve / gghstats-start)
```

## Troubleshooting

- **`healthz` fails but process runs:** check **`gghstats_port`** matches **`GGHSTATS_PORT`** in the generated env (default `8080`).
- **OpenBSD rc.d:** use **`contrib/openbsd/gghstats`** (`daemon="/usr/local/bin/gghstats-serve"`, **`rc_bg=YES`**) plus **`gghstats-serve`** on **`PATH`**. **`start_cmd`** is not used by OpenBSD `rc.subr`. VM checklist: **`contrib/openbsd/DEBUG-VM.md`**.

- **`listen tcp … address already in use`:** a previous `gghstats` is still on the port. On OpenBSD: `rcctl stop gghstats; pkill -x gghstats`. Setup stops stray processes before each start.
- **`rcctl check` shows `failed` but `pgrep gghstats` works:** the process may have been started outside `rcctl`; stop/kill and re-run setup.
- **OpenBSD without `curl`:** platform tests poll **healthz** with **`ftp`** (see **`platform_vars/openbsd.yml`**). For manual checks: `ftp -o - http://127.0.0.1:8080/api/v1/healthz`.
- **Daemon exits immediately:** verify **`gghstats_github_token`** and **`gghstats_filter`**; read service logs (`journalctl -u gghstats`, `service gghstats status`, or `/var/log/gghstats.log` on FreeBSD).
- **FreeBSD `make` in repo root:** use **`gmake`** for `dist-freebsd` / `port-freebsd-sync` — see `contrib/freebsd/README.md`.

## Relationship to gghstats-selfhosted

| Repo | Native OS (deb/rpm/BSD) | Docker / Helm |
|------|-------------------------|---------------|
| **gghstats** (here) | `make test-platforms` | — |
| **gghstats-selfhosted** | — | `make test-compose-platforms`, `make test-helm-kind` |

Run both before a release if you changed packaging **and** deployment manifests.
