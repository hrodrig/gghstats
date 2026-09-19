# Plan — v1.6.x

**Status:** **Open** — **1.6.0** / **1.6.1** shipped; remaining slices land as
**1.6.2+** patches when they are docs/ops/tiny UX, or as a **1.7.0** minor when a
slice adds a new CLI surface or non-trivial API/UI contract. Prefer patches until
a slice clearly needs a minor bump (see Versioning).

**Band goal:** After the 1.6.0 dashboard/Settings UX land, close **operator DX and
docs debt** (auth asymmetry notes, config introspection, release checklist
automation), plus a few **high-leverage UX** additions that reuse existing
data/UI — without opening Line B or non-goals.

Parent: [ROADMAP.md](../ROADMAP.md) · Spec: [SPEC.md](../SPEC.md) · Themes:
[themes.md](themes.md).

## Versioning

| Tag | Allowed |
|-----|---------|
| **1.6.0** | Shipped — responsive UX, gated Settings, bg/ru, soft theme starter (#46/#56) |
| **1.6.1** | Shipped — Featured chip contrast (#60) + `docs/themes.md` (#61) |
| **1.6.2+** | DOC / OPS / tiny UX below; no new CLI subcommands unless agreed as patch |
| **1.7.0** | New CLI (`config check` / `doctor`), non-trivial keyboard/shareable-URL
| or badge extensions — bump minor when first of those lands |
| **2.0.0** | Line B — not this band |

SemVer reminder (ROADMAP): patch = small corrections / docs / tiny UX on the
current minor; minor = additive product surface without breaking the 1.x
contract.

## Already shipped (do not re-open)

| ID | Item | Release |
|----|------|---------|
| **UX-46** | Dashboard Settings / soft theme / locales / #56 offcanvas | 1.6.0 |
| **UI-60** | Featured stars/chips readable under Midnight/Dark | 1.6.1 |
| **DOC-61** | Built-in theme palettes + swatches + component contract | 1.6.1 |
| **DEP-sqlite** | `modernc.org/sqlite` 1.59.0 + keep `golang.org/x/net` pin | develop (post-1.6.1) |

## In scope

| ID | Slice | Item | Effort | Notes |
|----|-------|------|--------|--------|
| **DOC-auth** | Docs | Stronger operator note: `GET /{owner}/{repo}/traffic.json` is **public when `GGHSTATS_API_TOKEN` is unset** (option 3); always report-scoped | S | SPEC/CHANGELOG already state it; add line next to `GGHSTATS_API_TOKEN` in `contrib/gghstats.env.example` (+ short README/`docs/api.md` cross-link if thin) |
| **DOC-idx** | Docs | Keep `docs/README.md` index in sync with files on disk (`api`, `themes`, band plans) | S | Include this plan in the docs index |
| **DOC-faq** | Docs | Public FAQ note: aggregate index chart has **no per-repo coverage matrix** (1.5.0 known limitation) | S | README FAQ or `docs/api.md` one-liner |
| **OPS-1d** | Ops | Recheck production dogfood when unique cloners look frozen and `(1d)` is all zeros | S | Sync All + export; compare to prior capture; code change only if store/SQL bug confirmed |
| **DX-check** | CLI | `gghstats config check` — validate env (token present/demo, filter regex, DB path writable, API-only+CORS warn, port) | M | Prefer **1.7.0** if first new CLI |
| **DX-doctor** | CLI | `gghstats doctor` — last sync, DB size, rate_limit peek, repos without traffic, filter vs DB divergence | M | Builds on DX-check; **1.7.0** |
| **META-rel** | Tooling | Pre-release drift checker: VERSION vs README badge vs man `.TH` vs BSD `PORTVERSION` / OpenBSD PKGNAME; print delta (optional `--fix` later) | M | Defends the VERSION-bump checklist; `make` target OK |
| **UX-h2h** | UI | Index row link → `/h2h` with repo A (or B) prefilled | S | Additive query params only |
| **UX-keys** | UI | Keyboard shortcuts: `/` focus search, `?` help, `Esc` close modal, optional `t` theme | S–M | Frontend-only |
| **UX-url** | UI | Shareable index/repo query state (`range`, `metric`, `theme`, selected repos where cheap) | M | `history.replaceState`; no backend |
| **A-uniques** | Alerts | Optional alert rule `metric` for GitHub **uniques** (not only `count`) | M | Parked from [plan-v1.5.0.md](plan-v1.5.0.md); SPEC §8 explicit |
| **DOC** | Docs | SPEC / README / api / man / CHANGELOG / ROADMAP as slices land | — | English |
| **REL** | Release | Patch or minor bump per Versioning; `make release-check` only after user asks | — | develop→main + tag need explicit OK |

### Suggested implementation order

1. **DOC-auth + DOC-idx + DOC-faq** (docs-only; ship as **1.6.2** candidate).
2. **OPS-1d** (operator session; may close without code).
3. **UX-h2h** then **UX-keys** (small UI; patch or early 1.7.0).
4. **META-rel** (tooling; can land anytime; no VERSION bump required until used in release).
5. **DX-check → DX-doctor** (**1.7.0**).
6. **UX-url** / **A-uniques** when capacity allows.
7. **REL** when a coherent set is ready.

Work lands on `develop` via pull requests (repo gitflow).

## Design decisions (recorded)

1. **Auth option 3 stays.** Do not force a token for HTML `traffic.json` when
   unset; document the public-dashboard tradeoff harder instead of changing
   behavior (unless a later security band reopens it).
2. **`(1d)` semantics unchanged** unless OPS-1d proves a store/SQL bug. GitHub
   lag remaining the default explanation when sync `failed=0` but series stall.
3. **CLI DX before OpenAPI / SSE / pprof.** Prefer introspection operators can
   run on the VPS over new HTTP surfaces.
4. **Dependabot bumps must keep `golang.org/x/net` pin.** After any tidy, run
   `go get golang.org/x/net@v0.57.0` + `make check-x-net-pin` (AGENTS.md).
5. **No Line B in 1.6.x.** Webhooks / delta sync wait for **2.0.0**.

## Out of scope

- Line B (webhooks / GraphQL / serious ROADMAP rewrite).
- Index per-repo coverage matrix UI (still parked; FAQ only unless operators demand).
- PWA / Service Worker, SMTP digest, RSS, multi-tenancy, GitHub App/OAuth.
- In-tree SPA; PostgreSQL; replacing neo-brutalist with soft as default.
- Release-traffic correlation, local repo tags, heatmap calendar, sync `--dry-run`,
  sparkline badges — see Parking lot.
- Selfhosted Compose/Helm work (lives in **gghstats-selfhosted**).

## Parking lot (after this band)

1. Release ↔ traffic correlation on repo page.
2. Local repo tags / notes + index filter.
3. Traffic heatmap calendar.
4. Sync `--dry-run` / preview.
5. SVG sparkline badges.
6. Fail-closed upgrade helper CLI (see [plan-v1.5.0.md](plan-v1.5.0.md) parking).
7. Rate-limit-aware sync pause when GitHub `Remaining==0`.
8. OpenAPI stub, SSE sync-done, token-gated pprof.

## Exit criteria

1. DOC-auth + DOC-idx (+ DOC-faq) on `develop`.
2. OPS-1d recorded (pass/fail vs GitHub lag) in CHANGELOG/notes, **or** closed as
   “lag confirmed / bug filed”.
3. At least one of UX-h2h or UX-keys on `develop`, **or** explicitly deferred in
   CHANGELOG with user OK.
4. If DX-check/doctor land: man page + `--help` + tests; VERSION minor bump.
5. META-rel usable locally (`make …`) before the next release that uses it.
6. ROADMAP band row + CHANGELOG `[Unreleased]` aligned.
7. `make release-check` (user asks); merge develop→main + tag (user OK).

## Checklist

- [x] Finalize this plan + docs index / ROADMAP link → `develop` (gitflow)
- [x] DOC-auth (`gghstats.env.example` + thin README/api cross-link)
- [x] DOC-idx (`docs/README.md` lists api/themes/1.x plans + this file)
- [ ] DOC-faq (index coverage matrix known limitation)
- [ ] OPS-1d dogfood recheck note
- [ ] UX-h2h index → H2H prefill
- [ ] UX-keys shortcuts
- [ ] META-rel pre-release drift checker
- [ ] DX-check (→ 1.7.0 if first CLI)
- [ ] DX-doctor
- [ ] UX-url shareable state (optional this band)
- [ ] A-uniques alert metric (optional this band)
- [ ] CHANGELOG / ROADMAP / SPEC touch as slices land
- [ ] `VERSION` bump when releasing; `make release-check` only after user asks
- [ ] Merge develop→main + annotated tag (user OK)
