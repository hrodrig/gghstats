# Plan — v1.7.0

**Status:** **Open** — planning; topic PRs → `develop` only.  
**Issues (band):** [#86](https://github.com/hrodrig/gghstats/issues/86) · [#97](https://github.com/hrodrig/gghstats/issues/97) · [#124](https://github.com/hrodrig/gghstats/issues/124)  
**Constraint:** ~8k production installs — **zero silent history loss** on the identity track.

Parent: [ROADMAP.md](../ROADMAP.md) · Spec: [SPEC.md](../SPEC.md) · Prior band: [plan-v1.6.x.md](plan-v1.6.x.md).

Local GSD (not committed): `.planning/PRODUCTION-SAFETY-v1.7.0.md`, `.planning/ROADMAP.md`, `.planning/phases/`.

---

## Band goal

Three parallel tracks, one release band:

| Track | Issue | Goal |
|-------|-------|------|
| **Identity** | [#86](https://github.com/hrodrig/gghstats/issues/86) | Traffic history survives renames/transfers via GitHub `repository_id`; reversible SQLite migrate |
| **MCP** | [#97](https://github.com/hrodrig/gghstats/issues/97) | Read-only MCP server (`stdio` + `sse`) so agents query local history without scraping HTML |
| **MWG web** | [#124](https://github.com/hrodrig/gghstats/issues/124) | Apply [Modern Web Guidance 1.0](https://developer.chrome.com/blog/modern-web-guidance-v1) to dashboard `web/` only (dialog/theme/motion/a11y) |

**Ship rule:** Identity is the **hard gate** for tagging 1.7.0 (data safety for ~8k). MCP MVP + MWG P1–P2 land in the same band (same tag preferred; if calendar slips, **1.7.x** patches immediately after — do not orphan the issues).

---

## Versioning

| Tag | Allowed |
|-----|---------|
| **1.6.9** | Current production pin (Go 1.27.4); rollback target for identity migrate |
| **1.7.0** | Identity (`migrateV9`) + MCP MVP + MWG P1–P2 — **this band** |
| **1.7.x** | Patches: MWG P3–P4 leftovers, MCP tool polish, docs/tiny UX |
| **1.8+ / later** | Optional `migrateV10` PK rebuild / DROP TEXT; MCP write tools if ever |
| **2.0.0** | Line B — out of scope |

DX-check / doctor / keyboard / shareable URL from [plan-v1.6.x.md](plan-v1.6.x.md) may follow as **1.7.x** after the three tracks above are green.

---

## Track A — Stable repo identity (#86)

### Non-negotiables (production)

| # | Rule |
|---|------|
| N1 | Auto `VACUUM INTO` `<db>.pre-v9.bak` via `Store.BackupTo` before v9 schema writes; never overwrite bak |
| N2 | **Expand/contract** — ADD columns + `repo_names`; **keep** `repo TEXT` PKs in 1.7.0 (no DROP) |
| N3 | Additive public contracts only (`repository_id` optional; `name` = canonical) |
| N4 | Refuse start if DB `user_version` > binary max |
| N5 | Bump `user_version=9` only after parity; failed migrate stays at 8 |
| N6 | Never merge history by name — exact `repository_id` only |
| N7 | Fix `syncRepo` name-before-meta **before** treating renames as solved |
| N8 | No merge to `main` / tag without explicit maintainer OK; dogfood ≥48h first |

Full contract: `.planning/PRODUCTION-SAFETY-v1.7.0.md`.

### Scope (#86)

| ID | Slice | Item | Effort | Notes |
|----|-------|------|--------|-------|
| **ID-03** | Schema | `migrateV9`: `github_id`, `id_confirmed`, fact `repo_id`, `repo_names`; provisional negatives; backup + refuse-newer | L | PR **A** |
| **ID-01** | Store | Dual-write name + `repo_id`; dual-read prefer `repo_id`; Store APIs keep `repo string` | L | PR **B** |
| **SYNC-01** | Sync | Metadata/identity **before** capturing `name`; rename / reuse / confirm | L | PR **C** — needs A+B |
| **ID-02** | API/UI | Canonical display; HTML 301; badges alias; additive JSON `repository_id` | M | PR **D** |
| **SYNC-02** | Upsert | replace daily views/clones; MAX for stars/referrers/paths | M | PR **E** — parallel after A |
| **DOC-01** | Docs | SPEC / README / CHANGELOG / man / #86 hygiene; upgrade + rollback | M | PR **F** |

### Identity PR order

| PR | Content | Merge alone? |
|----|---------|--------------|
| **A** | migrateV9 + backup + refuse-newer + parity tests | Yes |
| **B** | Store dual-read/write + identity helpers | Yes |
| **C** | `syncRepo` order + ResolveRepoIdentity | Needs A+B |
| **D** | API/UI alias + 301 | After C |
| **E** | SYNC-02 SPEC + Upsert | Parallel after A |
| **F** | Docs + #86 hygiene + VERSION prep | Last |

### Operator upgrade / rollback

**Before:** note tag (**v1.6.9**); stop writes if possible; optional manual backup; free disk ≥ **2×** DB.

**Upgrade:** deploy **v1.7.0** (Compose: `GGHSTATS_VERSION` + **pull** + **`up -d`**); first Open → `.pre-v9.bak` + migrateV9; logs show backup path + `provisional=M`; spot-check history + one sync.

**Rollback:** stop → restore `.pre-v9.bak` → pin **v1.6.9** → confirm `user_version=8`. Do not re-run 1.7.0 until fixed.

**Caveat:** After v9, older binaries refuse the DB. Rollback = restore bak + older image.

---

## Track B — MCP server (#97)

**Goal:** Agents query accumulated traffic via MCP without custom SQL or HTML scrape.

**Full design (flags, tools JSON, auth matrix, PR G1–G3, tests):** [plan-v1.7.0-mcp.md](plan-v1.7.0-mcp.md).

### Scope (#97)

| ID | Slice | Item | Effort | Notes |
|----|-------|------|--------|-------|
| **MCP-01** | CLI | `gghstats mcp` — **stdio** transport (default); logs → **stderr** only | M | PR **G** |
| **MCP-02** | Transport | **sse** transport + optional Bearer token (`--token` / env) | M | PR **G** or **G2** |
| **MCP-03** | Tools | Read-only: `list_tracked_repos`, `get_repo_summary`, `get_traffic_timeseries`, `get_top_referrers` | M | Respect report_policy / hidden |
| **MCP-04** | Safety | SQLite **read-only** open (`mode=ro`); no write tools in 1.7.0 | S | Must not fight sync writer |
| **MCP-DOC** | Docs | man / README / SPEC / CHANGELOG; Claude Desktop / Cursor config examples | S | With **F** or MCP PR |

### MCP constraints

- **New Go dep allowed** for this track only: official MCP Go SDK **or** `github.com/mark3labs/mcp-go` — pin version; `go mod tidy`; supply-chain review in PR.
- Identity track stays **no new deps**.
- Tools resolve repos via same identity helpers once Track A is in (prefer `repository_id` when present; accept `owner/name`).
- Do **not** expose hidden / report-excluded repos unless operator auth matches existing API rules.
- Out of 1.7.0: write/mutate tools, multi-DB fan-out, remote GitHub calls from MCP.

### MCP acceptance

- [ ] `gghstats mcp` stdio speaks valid MCP; smoke with a desktop client or MCP inspector
- [ ] sse mode serves tools behind token when configured
- [ ] Read-only DB path verified (sync can still write on another connection/process)
- [ ] stdout clean in stdio mode (no slog on stdout)

---

## Track C — Modern Web Guidance (#124)

**Scope root:** `web/templates/`, `web/static/app.css`, `web/static/app.js`, `contrib/themes/` only.  
**Out:** Go server, sync, store, packaging, marketing sites.

Sources: [MWG 1.0](https://developer.chrome.com/blog/modern-web-guidance-v1); local pointer under cursor-plans `Modern Web Guidance 1.0.txt`.

### Scope (#124)

| ID | Slice | Item | Effort | Notes |
|----|-------|------|--------|-------|
| **MWG-P1** | UI | Settings / sync-token modal + theme toggle vs `dialog` / `dark-mode` guides (focus, Esc, light-dismiss, labels) | M | PR **H** |
| **MWG-P2** | UI | `prefers-reduced-motion` in CSS + Chart.js animation options | S | PR **H** or **H2** |
| **MWG-P3** | UI | Modernize scan of `web/static` + templates; file only Baseline-safe follow-ups | S | May spill to 1.7.x |
| **MWG-P4** | UI | Optional `content-visibility` / offscreen chart panels | S | Optional / 1.7.x OK |
| **MWG-DOC** | Docs | Record keep-Bootstrap-modal vs `<dialog>`; motion + `color-scheme` decisions | S | Close #124 |

### MWG constraints

- No wholesale Bootstrap drop without Baseline + a11y proof.
- Keep CSP; never inject unsanitized HTML from custom CSS/themes.
- Identity UI bits (301 / “formerly …”) stay in Track A PR **D** — MWG PRs must not fight those templates; rebase after D if needed.

### MWG acceptance (close #124)

- [ ] Decisions documented (modal strategy, motion, theme/`color-scheme`)
- [ ] P1–P2 landed or explicitly deferred with rationale
- [ ] No regression: keyboard Settings + light/dark/midnight toggle

---

## Cross-track PR map

| PR | Track | Content |
|----|-------|---------|
| A–F | #86 | Identity (see above) |
| G | #97 | MCP stdio (+ sse/tools as one or split G2) |
| H | #124 | MWG P1–P2 (+ P3 if cheap) |
| F | #86+#97+#124 | Docs / VERSION prep — last before cut |

**Parallelism:** G and H can start after A (or even before) on topic branches; rebase onto develop after identity B/C if they touch Store/web. Prefer not to merge H into templates D is actively rewriting without coordination.

---

## Rollout stages (~8k)

| Stage | Who | Gate |
|-------|-----|------|
| S0 | CI + `make release-check` | Green (identity + MCP + web) |
| S1 | Maintainer dogfood ≥48h | History intact; rename test; MCP smoke; Settings/theme OK |
| S2 | Tag **v1.7.0** + GHCR | Explicit OK |
| S3 | gghstats-selfhosted pin + webs | After S2 green |
| S4 | Community | Release notes: identity rollback + MCP how-to + MWG notes |

---

## Design decisions (LOCKED)

### Identity (#86)

1. Expand/contract in 1.7.0 — keep TEXT keys.
2. Provisional negative `github_id` until sync confirm.
3. Merge only by exact `repository_id`.
4. SYNC-02: replace views/clones; MAX snapshots.
5. No new Go deps **on identity PRs**.
6. PK DROP deferred past 1.7.0.

### MCP (#97)

7. Read-only tools + read-only SQLite in 1.7.0.
8. Dual transport: stdio (default) + sse (optional auth).
9. One pinned MCP Go library; review in PR.

### MWG (#124)

10. Incremental PRs; Baseline-first; Bootstrap stays until proven swap.
11. Honor `prefers-reduced-motion`; tighten dialog/theme a11y before perf toys.

---

## Test gates (before merge to develop)

### Migration / identity

- [ ] Empty DB → v9; v8 fixture parity; pins/featured/hidden/policy preserved
- [ ] Interrupted migrate → still v8; bak not overwritten; refuse newer DB
- [ ] Rename continuous; name reuse no merge; allowlist stale → meta first
- [ ] Frozen upstream → no invented days
- [ ] Old name HTML 301; badge 200; JSON additive only

### MCP

- [ ] stdio protocol smoke; sse + token reject without token when required
- [ ] Tools omit hidden / non-report repos per policy
- [ ] No stdout pollution in stdio mode

### MWG

- [ ] Keyboard Settings path; Esc/light-dismiss behavior documented
- [ ] Reduced-motion: charts/CSS do not animate when preference set
- [ ] Theme toggle still sets light/dark/midnight correctly

### Regression

- [ ] `make lint` / `make test` / cover floor
- [ ] Store public signatures unchanged for callers

---

## Checklist

### #86 Identity

- [ ] PR A–F on develop
- [ ] Dogfood ≥48h (history + rename)

### #97 MCP

- [ ] PR G (stdio + tools; sse)
- [ ] Smoke with inspector / desktop client
- [ ] Docs examples

### #124 MWG

- [ ] PR H (P1–P2)
- [ ] Decisions written; #124 closable or leftover issues filed

### Release

- [ ] VERSION cut + release-check (user OK)
- [ ] PR develop→main + annotated tag (user OK)
- [ ] selfhosted + webs pin cascade
- [ ] Close or update #86 / #97 / #124

---

## Out of scope (1.7.0)

- DROP TEXT / rebuild PKs around `repo_id` only
- Multi-token / multi-account sync
- Online GitHub required during Open()/migrate
- Forced big-bang upgrade for all operators
- MCP write tools / mutating DB via agents
- Dropping Bootstrap wholesale
- MWG on marketing sites (`gghstats-com`)
- Line B

---

## Parking (later)

1. `migrateV10` PK rebuild once all `id_confirmed`
2. Multi-token sync
3. MWG P3–P4 leftovers as 1.7.x
4. DX config check / doctor (from 1.6.x plan)
5. Optional health JSON: `schema_version`, `identity_provisional_count`

---

*Updated 2026-11-21 — band = #86 + #97 + #124; ~8k production-safe identity gate*
