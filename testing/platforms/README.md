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

**Target OS:** Linux with Docker **not** required. **\*BSD** hosts are tested **natively** (no Docker on the target). Compose on BSD is out of scope — see **gghstats-selfhosted** for Linux-only Compose tests.

## Package sources and install paths

Platform tests install **GoReleaser artifacts** (`.deb`, `.rpm`, OS tarballs). You do **not** run GoReleaser on the target VM.

Inventory variable: **`gghstats_package_source`** (`local` | `auto` | `release`). Related: **`gghstats_version`**, **`gghstats_release_url`**, optional **`gghstats_dist_dir`**, **`gghstats_local_package`**, **`gghstats_use_dist_for_auto`** (default **`true`** in **`inventory/group_vars/all.yml`**).

### How `gghstats_package_source` chooses an artifact

The install role sets **`gghstats_use_local_package`**: copy from the control node vs download on the target (`get_url`).

| Value | When to use | Install role behavior | GitHub release required? |
|-------|-------------|------------------------|---------------------------|
| **`local`** | Pre-release on **`develop`**; packaging still changing | **Always** copy from control node **`dist/`** (or **`gghstats_local_package`**). Fails if the file is missing. | No — run **`make snapshot`** first |
| **`auto`** | Default-friendly mixed lab | If **`gghstats_use_dist_for_auto: true`** and **`dist/gghstats_<version>_…`** exists on the control node → **copy**; else → **download** from **`gghstats_release_url`** on the target | Only when falling back to download |
| **`release`** | Post-tag smoke on real VMs | **Always download** on the target from **`gghstats_release_url`**, even if **`dist/`** exists locally | Yes — tag **`v{{ gghstats_version }}`** and matching assets |

**Decision flow (`auto`):**

```
gghstats_package_source == local     → copy from dist/ (required)
gghstats_package_source == release   → download from GitHub
gghstats_package_source == auto
  → dist/ artifact exists?  copy : download
     (only checks dist/ when gghstats_use_dist_for_auto: true)
```

Set **`gghstats_version`** to semver **without** `v` (e.g. `0.6.4`). Release URL:

`https://github.com/hrodrig/gghstats/releases/download/v{{ gghstats_version }}/gghstats_<version>_…`

Filenames match GoReleaser (no `v` in the file): `gghstats_0.6.4_linux_amd64.deb`, `gghstats_0.6.4_freebsd_amd64.tar.gz`, etc. The role picks the suffix from **`platform_vars`** (`.deb`, `.rpm`, `freebsd`/`openbsd`/`linux` tarball).

Override one host: **`gghstats_local_package: "/path/to/custom.deb"`** (works with **`local`** or **`auto`** when the file exists).

**Defaults:** **`inventory/group_vars/all.yml`** sets **`gghstats_package_source: local`** and **`gghstats_version: "0.6.4-next"`** for lab work. The install role’s internal default if unset is **`auto`** — set **`gghstats_package_source`** explicitly in inventory to avoid surprises.

### What Ansible installs today (by OS)

| Platform | Artifact | How the test installs it | Operator equivalent |
|----------|----------|---------------------------|---------------------|
| Debian / Ubuntu | `.deb` | `dpkg -i` (local copy or download) | `wget` release + `dpkg -i`, or `apt` from file URL |
| AlmaLinux / RHEL | `.rpm` | `dnf install` / `rpm -i` | `dnf install` release URL |
| OpenSUSE | `.rpm` | `zypper install` | Same |
| Arch / Alpine | linux `.tar.gz` | Extract + install binary, unit/init script | Manual tarball path in docs |
| **FreeBSD** | freebsd `.tar.gz` | Binary from tarball; **rc.d from repo checkout** (`contrib/freebsd/rc.d` → target) | **Not yet** `pkg install` from ports |
| **OpenBSD** | openbsd `.tar.gz` | Binary from tarball; **wrappers + rc.d from repo checkout** (`contrib/openbsd/*`; tarball may also ship rc assets) | **Not yet** `pkg_add gghstats` from ports |

**\*BSD note:** with **`release`** or **`auto`** (GitHub fallback), the **binary** comes from the tarball on GitHub, but **rc.d / wrappers** are still copied from your **gghstats clone** on the control node (**`gghstats_daemon`**). Run playbooks from a checkout at or near the tag you are testing. Linux **`.deb`/`.rpm`** from GitHub are self-contained (package + maintainer scripts).

With **`gghstats_package_source: release`**, you do **not** need **`make snapshot`** on the laptop — the target downloads release assets. That is the usual **post-release** gate (Linux fully; BSD binary only — see note above).

With **`local`**, run **`make snapshot`** once, set **`gghstats_version`** from **`dist/metadata.json`** (e.g. `0.6.4-next`), then **`make test-platforms`**.

### Pre-release vs post-release (quick pick)

**Before tagging (packaging still in flux):**

```yaml
gghstats_package_source: local
gghstats_version: "0.6.4-next"   # from dist/metadata.json after make snapshot
```

**After `v0.6.4` is on GitHub Releases:**

```yaml
gghstats_package_source: release
gghstats_version: "0.6.4"
```

### \*BSD: tarball path today; port / `pkg_add` (planned)

| Path | Status | What it validates |
|------|--------|-------------------|
| **Release tarball + Ansible** | **Implemented** | GoReleaser BSD tarball, env file, rc.d enable/start, healthz, teardown |
| **Port build → `pkg install` / `pkg_add`** using distfile from **GitHub Releases** | **Planned** | `contrib/freebsd` / `contrib/openbsd/port` — PLIST, `@rcscript`, pkg delete (same distfile as release, no local `dist/`) |
| **`pkg_add gghstats` from official mirrors** | **Future** | Only after the port is accepted into the **official** FreeBSD/OpenBSD ports trees |

Today’s playbooks follow the **first row**: they mirror a careful **manual tarball install**, not **`pkg_add`** from the ports tree. Port files live under **`contrib/freebsd/`** and **`contrib/openbsd/port/`**; see **`PORT-RELEASE.md`** in each directory for manual port validation.

When **`install_method: port_pkg`** is added, **`release`** mode will still mean “use the GitHub distfile”, but installation will go through the **port Makefile** (or a built **`.pkg`**) instead of manual `tar xzf`. Until then, use **`release`** for BSD tarball smoke after each tag.

**Port maintainer guide (step-by-step):** [../../contrib/BSD-PORTS-STEP-BY-STEP.md](../../contrib/BSD-PORTS-STEP-BY-STEP.md) — how to run **`gmake port-*-sync`**, build distfiles, and validate **`make install`** / future **`pkg_add`** on FreeBSD and OpenBSD VMs.

## Supported platforms

| Platform | Test artifact | Init system |
|----------|---------------|-------------|
| Debian / Ubuntu | `.deb` | systemd |
| AlmaLinux / RHEL family | `.rpm` (dnf) | systemd |
| FreeBSD | freebsd `.tar.gz` (port/pkg planned) | rc.d (`service`) |
| OpenBSD | openbsd `.tar.gz` (port/pkg planned) | rc.d (`rcctl`) |
| Arch Linux | linux `.tar.gz` | systemd (unit from Ansible template) |
| OpenSUSE | `.rpm` (zypper) | systemd |
| Alpine Linux | linux `.tar.gz` | OpenRC (`contrib/openrc/gghstats.initd`) |

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
- Optional: **`gghstats_version`**, **`gghstats_package_source`**, **`gghstats_local_package`**, **`gghstats_port`**, **`gghstats_sync_on_startup`** (default `false` for faster smoke tests), **`gghstats_test_fetch_repo`** (e.g. `owner/repo` for optional API fetch test)

See **[Package sources and install paths](#package-sources-and-install-paths)** for **`local` / `auto` / `release`**, pre-release **`make snapshot`**, and BSD tarball vs future port/pkg.

### Local snapshot (checklist)

1. **`make snapshot`** → **`dist/`** + **`dist/metadata.json`**
2. **`gghstats_version`**: copy **`version`** from metadata (e.g. `0.6.4-next`)
3. **`gghstats_package_source: local`** (default in **`inventory/group_vars/all.yml`**)
4. Optional **`gghstats_dist_dir`**: absolute path if Ansible runs outside the clone
5. **`make test-platforms-ping`**, then **`make test-platforms`**

After **`v<version>`** is published, switch to **`gghstats_package_source: release`** and **`gghstats_version: "<version>"`**.

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
