# docs/

Working notes and assets that support the product docs at the repository root.

| Document | Role |
|----------|------|
| [ROADMAP.md](../ROADMAP.md) | Product direction and release bands |
| [SPEC.md](../SPEC.md) | Normative product behavior (API, sync, CLI, alerts, …) — **what** and **how** |
| [CHANGELOG.md](../CHANGELOG.md) | Release notes (Keep a Changelog) |
| [README.md](../README.md) | Operator-facing install and usage |
| `docs/plan-v*.md` | Band plans — **what we will implement** this band (scope, order, exit, checklist) |

## Release-band plans

Scoped implementation checklists per SemVer band (behavior details live in SPEC):

| Band | Plan |
|------|------|
| **0.9.x** | [plan-v0.9.x.md](plan-v0.9.x.md) (shipped as **v0.9.0**) |
| **0.10.x** | [plan-v0.10.x.md](plan-v0.10.x.md) (closed **v0.10.1**) |
| **0.11.x** | [plan-v0.11.x.md](plan-v0.11.x.md) |
| **1.0.0** | [plan-v1.0.0.md](plan-v1.0.0.md) |

When a band ships, fold `[Unreleased]` into CHANGELOG, bump `VERSION` / badges / man / BSD ports, and mark the plan checklist complete.

## Terminal demo (VHS)

[`demo.tape`](demo.tape) drives [Charm VHS](https://github.com/charmbracelet/vhs) to record a short CLI walkthrough into `docs/demo.gif`.

```bash
make install
PATH="$(go env GOPATH)/bin:$PATH" bash -c "vhs docs/demo.tape"
```

Commit an updated `docs/demo.gif` when the tape or CLI surface changes in a user-visible way.
