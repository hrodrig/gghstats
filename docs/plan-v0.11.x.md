# Plan — v0.11.x (optional)

**Band goal:** cut polling cost with **webhooks + delta-oriented sync**, without abandoning the PAT model.

Parent: [ROADMAP.md](../ROADMAP.md) · Prior: [plan-v0.10.x.md](plan-v0.10.x.md)

## Decision gate

| If… | Then… |
|-----|--------|
| Webhook + delta design fits in a focused cycle | Ship as **0.11.x** before 1.0 |
| Scope balloons (GitHub App, multi-tenant, GraphQL rewrite) | **Skip this band**; land 1.0; finish Line B as **1.1+** |

Do **not** block **1.0.0** on this plan.

## In scope

| ID | Item | Notes |
|----|------|--------|
| B1 | **GitHub webhooks** | Receive events that invalidate or refresh specific repos; secure secret validation. |
| B2 | **Delta sync** | Prefer updating changed repos/metrics vs full filter sweep every tick. |
| B3 | **GraphQL (selective)** | Only where it clearly reduces REST pagination cost; not a wholesale API rewrite. |
| B4 | **Scheduler coexistence** | Keep interval sync as fallback; webhooks reduce, not replace, reliability path. |

## Out of scope (this band)

- GitHub App / OAuth (non-goal unless major rethink).
- Replacing SQLite.
- Freezing API for 1.0 (parallel track: keep additive-only).

## Exit criteria

1. Documented webhook setup (`GGHSTATS_*` secrets, reverse-proxy notes).
2. Measurable reduction in GitHub API calls for typical “few repos changed” days vs full poll.
3. Fallback scheduled sync still works when webhooks are disabled or unreachable.
4. Tests for signature validation and delta path; CHANGELOG + SPEC for any new endpoints.

## Checklist

- [ ] Go / no-go decision recorded in CHANGELOG or ADR note
- [ ] Webhook receiver + auth
- [ ] Delta sync path + metrics
- [ ] Optional GraphQL for one hot path (stars or repo list) — only if justified
- [ ] Docs + `make test` / lint
