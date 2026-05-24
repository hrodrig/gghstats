# Testing

Local and lab validation for **gghstats** (application repo).

| Suite | Command | What it validates |
|-------|---------|-------------------|
| **Unit tests** | `make test` | Go packages (`go test -race ./...`) |
| **Release gate** | `make release-check` | lint, tests, security (and optional `STRICT_RELEASE=1` image scan) |
| **Platform (native OS)** | `make test-platforms` | Install package/tarball, `/etc/gghstats/gghstats.env`, init system, HTTP `/api/v1/healthz`, uninstall — see [platforms/README.md](platforms/README.md) |

**Out of scope here (see [gghstats-selfhosted](https://github.com/hrodrig/gghstats-selfhosted)):**

- Docker Compose minimal stack on Linux VPS (`make test-compose-platforms`)
- Helm chart on **kind** (`make test-helm-kind`)

Those exercise **container/Kubernetes manifests**, not `.deb`/`.rpm`/BSD rc.d installs.

## Platform tests (Ansible)

Requires Ansible 2.14+, SSH to lab VMs, and a GitHub PAT with read access to repos in `gghstats_filter`.

```bash
make snapshot   # dist/ + dist/metadata.json (version e.g. 0.6.4-next)
cp testing/platforms/inventory/hosts.yml.example testing/platforms/inventory/hosts.yml
# Edit: gghstats_version from metadata.json, token, ansible_host_lab
# gghstats_package_source: local (default) — uses dist/gghstats_* artifacts, not GitHub

make test-platforms-ping
make test-platforms
make test-platforms LIMIT=gghstats-ubuntu
```

Pre-release (maintainers): **`make snapshot`** → align **`gghstats_version`** in inventory → ping → **`make test-platforms`**. After tagging GitHub releases, use **`gghstats_package_source: release`**.
