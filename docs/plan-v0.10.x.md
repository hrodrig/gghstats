# Plan — v0.10.x

**Status:** **v0.10.1** closed the 0.10.x band (2026-07-18) — core **v0.10.0** + milestones + SMTP. Patch **v0.10.2** ships the post-review security slice: trusted proxies (SEC1) and HTTP server timeouts (SEC2). Next: [plan-v0.11.x.md](plan-v0.11.x.md).

**Band goal:** cheaper sync for star-heavy repos, and **opt-in** operator signals (traffic + ops) when rules fire.

Parent: [ROADMAP.md](../ROADMAP.md) · Prior band: [plan-v0.9.x.md](plan-v0.9.x.md)

**Doc roles:**

| Doc | Role |
|-----|------|
| **This plan** | **What we will implement** in the 0.10.x band — scope, order, exit criteria, checklist |
| **[SPEC.md](../SPEC.md) §8** | **What the product must do** for alerts — traffic + ops rules, sinks (incl. Loki / SMTP), message format, secrets |

---

## In scope

| ID | Item | Notes |
|----|------|--------|
| SYNC | **Incremental star history** | **Implemented.** See [SPEC §4.7](../SPEC.md). |
| A2 | **Opt-in alerts** | **Implemented** (0.10.0 delivery). Sinks + `alert test` + traffic + ops. Contract: [SPEC §8](../SPEC.md). |
| A2+ | **Growth milestones** | **Implemented** (stars ladders, fire-once). [SPEC §8.3](../SPEC.md). |
| A2+sink | **Email / SMTP sink** | **Implemented** (groot-style STARTTLS / TLS). [SPEC §8.5](../SPEC.md). |
| PATH | **XDG / default path prep** | **Implemented** (docs soft-land). Binary default still `./data/gghstats.db`; recommended absolute paths in README / env / launchd / systemd. Code default → [v1.0.0](plan-v1.0.0.md). |
| QW | Remaining quick wins | **Implemented.** `getPaginatedCtx` dead `*[]Star` branch removed; access-log level by status (4xx warn / 5xx error). |
| SYNC+ | **UpdateDeltas efficiency** | Incremental / less frequent with star-sync work — not cargo-cult pool bumps. |

## Implementation order (A2)

1. Sinks (`GGHSTATS_ALERT_SINKS`) — Slack + generic webhook + **Loki** deliver in tests → [SPEC §8.5](../SPEC.md)
2. **`gghstats alert test`** — operator smoke-test before real rules → [SPEC §8.8](../SPEC.md)
3. Traffic rules (`GGHSTATS_ALERT_RULES`) — evaluate after sync, fan-out → [SPEC §8.2](../SPEC.md) / [§8.4](../SPEC.md)
4. Ops / sync-health rules — failure **counts**, **levels** (`warn`/`crit`), rate-limit floors → [SPEC §8.7](../SPEC.md)
5. Operator docs (README, env.example, man) aligned with SPEC §8
6. A2+ milestones + SMTP

**Hard rule:** do not ship rule evaluation without a working sink ([SPEC §8.5](../SPEC.md)) and a way to smoke-test delivery ([SPEC §8.8](../SPEC.md)).

## Out of scope (this band)

- Full webhook-driven GitHub sync architecture (→ 0.11.x or 1.1) — distinct from **alert** webhooks/sinks
- GraphQL total rewrite
- SPEC API freeze (→ 1.0)
- Multi-writer SQLite / Postgres
- WhatsApp / native Discord|Teams types / SMTP-in-0.10.0 (see SPEC §8)
- Thin leaderboard (→ [plan-v0.11.x.md](plan-v0.11.x.md) stretch **C?**)

## Exit criteria

1. Star history sync is incremental when star sync is enabled ([SPEC §4.7](../SPEC.md)).
2. **A2:** at least one sink delivers in tests (Slack **or** webhook **or** Loki); traffic + ops rules call sinks; behavior matches [SPEC §8](../SPEC.md). Growth milestones **not** required for **0.10.0**.
3. Default-path prep documented (README / launchd / env example); migration notes if behavior changes.

## Checklist

- [x] Incremental star sync + tests + SPEC §4.7
- [x] **A2 sinks first** — Slack + webhook + **Loki** + tests + env/docs ([SPEC §8.5](../SPEC.md)) — delivery only; rules next
- [x] **`gghstats alert test`** — smoke-test sinks without serve/sync ([SPEC §8.8](../SPEC.md))
- [x] A2 traffic rules after sync + fan-out ([SPEC §8.2](../SPEC.md) / [§8.4](../SPEC.md) / [§8.6](../SPEC.md))
- [x] A2 ops rules — counts / levels / rate-limit ([SPEC §8.7](../SPEC.md))
- [x] Operator docs — README alerts + env table; man / env.example ([SPEC §8](../SPEC.md))
- [x] XDG / default path prep docs (soft-land; no binary default change)
- [x] QW leftovers — `getPaginatedCtx` cleanup; access-log level by status
- [x] Demo/docs note: collector / update-check off in demo (README Features)
- [x] CHANGELOG notes for shipped SYNC work
- [x] `make test` / lint green (current tree)
- [x] *(0.10.1+)* Growth milestones (A2+) — `metric=stars`, `milestones:[…]`, fire-once
- [x] *(0.10.1+)* Email/SMTP sink (A2+sink)

## Parked (do not promote without pain)

- Raising SQLite `MaxOpenConns` / `MaxIdleConns` without measured contention
- Extra SQL INDEX on `(repo, date)` where PRIMARY KEY already exists
- Spec-nits only (explicit HEAD handlers, Accept-negotiation doc spam)
- Growth milestones before A2 sinks + basic rules
- Alert rules without a working sink
- WhatsApp / email-in-0.10.0 as first-class sinks
- Alerting on every single-repo blip (ops rules need count thresholds)
