# docs/

Working notes and assets that support the product docs at the repository root.

| Document | Role |
|----------|------|
| [ROADMAP.md](../ROADMAP.md) | Product direction and release bands |
| [SPEC.md](../SPEC.md) | Normative product behavior (API, sync, CLI, alerts, …) — **what** and **how** |
| [CHANGELOG.md](../CHANGELOG.md) | Release notes (Keep a Changelog) |
| [README.md](../README.md) | Operator-facing install and usage |
| [api.md](api.md) | HTTP API consumer guide (auth, examples, dogfood map); contracts in SPEC |
| [catalog-and-featured.md](catalog-and-featured.md) | Operator reference: `repo` pins + `featured` showcase (v1.1.0+) |
| [themes.md](themes.md) | Built-in light/dark/midnight palettes (token→hex, swatches, component contract) |
| `docs/plan-v*.md` | Band plans — **what we will implement** this band (scope, order, exit, checklist) |

## Release-band plans

Scoped implementation checklists per SemVer band (behavior details live in SPEC):

| Band | Plan |
|------|------|
| **0.9.x** | [plan-v0.9.x.md](plan-v0.9.x.md) (shipped as **v0.9.0**) |
| **0.10.x** | [plan-v0.10.x.md](plan-v0.10.x.md) (closed **v0.10.1**) |
| **0.11.x** | [plan-v0.11.x.md](plan-v0.11.x.md) |
| **1.0.0** | [plan-v1.0.0.md](plan-v1.0.0.md) |
| **1.1.0** | [plan-v1.1.0.md](plan-v1.1.0.md) |
| **1.4.0** | [plan-v1.4.0.md](plan-v1.4.0.md) |
| **1.5.0** | [plan-v1.5.0.md](plan-v1.5.0.md) (closed) |
| **1.6.x** | [plan-v1.6.x.md](plan-v1.6.x.md) (open — post-1.6.1 DX / docs / UX) |

When a band ships, fold `[Unreleased]` into CHANGELOG, bump `VERSION` / badges / man / BSD ports, and mark the plan checklist complete.

## Terminal demo (VHS)

[`demo.tape`](demo.tape) drives [Charm VHS](https://github.com/charmbracelet/vhs) to record a short CLI walkthrough into `docs/demo.gif`.

```bash
make install
PATH="$(go env GOPATH)/bin:$PATH" bash -c "vhs docs/demo.tape"
```

Commit an updated `docs/demo.gif` when the tape or CLI surface changes in a user-visible way.
