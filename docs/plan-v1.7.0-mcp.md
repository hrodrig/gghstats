# Plan detail — MCP server (#97) for v1.7.0

**Status:** Open — design locked for band planning; implement on topic branches.  
**Issue:** [#97](https://github.com/hrodrig/gghstats/issues/97)  
**Band plan:** [plan-v1.7.0.md](plan-v1.7.0.md)  
**Normative after ship:** SPEC.md (new § MCP) + man page + README

**Why this doc exists:** Identity (#86) has `PRODUCTION-SAFETY` depth. MCP needs the same level of operator/security detail before coding — ~8k installs must not get a foot-gun that dumps private traffic over the network.

---

## 1. Problem / value

GitHub keeps ~14 days of traffic. gghstats already stores long history in SQLite. Agents (Cursor, Claude Desktop, automation) cannot use that history without scraping HTML or inventing SQL.

**MCP** = standard tool surface so LLMs call **read-only** queries against the local DB.

---

## 2. Non-negotiables (MCP)

| # | Rule | Why |
|---|------|-----|
| M1 | **Read-only tools only** in 1.7.0 | Agents must not mutate history / pins / settings |
| M2 | **SQLite open read-only** (`mode=ro`, WAL-friendly) | Must not block or corrupt `gghstats serve` writer |
| M3 | **stdio: all logs → stderr** | stdout is JSON-RPC; pollution breaks clients |
| M4 | **SSE: bind loopback by default**; require auth token if bind is non-loopback | Prevent accidental public exposure of traffic history |
| M5 | **Same visibility rules as report/API** | `hidden`, `report_policy`, private inherit — no side channel |
| M6 | **No GitHub network calls from MCP** | MCP reads local DB only; sync stays in `serve` / `fetch` |
| M7 | **Pin one MCP library**; supply-chain review in PR | New dep is the exception carve-out for #97 only |
| M8 | Cap result size (date range + row limits) | Avoid multi-MB tool payloads that hang agents |

---

## 3. Library choice

| Candidate | Pros | Cons |
|-----------|------|------|
| **`github.com/mark3labs/mcp-go`** (issue default) | Mature `ServeStdio` / SSE helpers; many examples | Third-party (still common in Go MCP) |
| **`github.com/modelcontextprotocol/go-sdk`** (official) | First-party protocol alignment | API still evolving; more wiring for SSE |

**LOCKED for 1.7.0 planning:** implement against **`mark3labs/mcp-go`** (matches #97 proposal). Pin a **released tag** in `go.mod` (no `@latest` floating). Re-evaluate official SDK only if mark3labs blocks a protocol requirement — separate follow-up PR, not mid-band rewrite.

Supply-chain: `go get` from module proxy + `go.sum`; no vendored tarball from random mirrors; note version in CHANGELOG.

---

## 4. Process model (important for ~8k)

```
┌─────────────────────┐         ┌──────────────────────┐
│  gghstats serve     │ writes  │  gghstats.db (+WAL)  │
│  (collector/sync)   │────────►│                      │
└─────────────────────┘         └──────────▲───────────┘
                                           │ read-only
                                ┌──────────┴───────────┐
                                │  gghstats mcp        │
                                │  (stdio or sse)      │
                                └──────────────────────┘
```

**Recommended production layout:**

- Keep **`serve`** as the long-running daemon (unchanged).
- Run **`mcp` as a separate process** (desktop client spawns stdio; or systemd/sidecar for SSE).
- Do **not** embed MCP inside `serve` in 1.7.0 (blast radius, auth matrix, port conflicts). Parking: optional “MCP mount under serve” later.

Concurrent access: SQLite WAL allows one writer + readers. MCP must open with read-only URI so it never upgrades to write.

---

## 5. Package / file layout (proposed)

```
cmd/gghstats/
  main.go          # dispatch "mcp" subcommand
  mcp.go           # flags, env, run loop, logging to stderr
internal/mcp/
  server.go        # NewMCPServer + tool registration
  tools.go         # tool handlers
  tools_test.go
  auth.go          # SSE bearer check (constant-time compare)
  db.go            # open read-only Store / sql.DB
  scope.go         # report-visible repo filter (shared rules with report)
```

Reuse existing `internal/store` **read** APIs where possible (`ListRepos`, day series, referrers). Prefer thin wrappers over duplicating SQL. After identity (#86) lands, resolve `owner/name` **or** `repository_id` via identity helpers.

---

## 6. CLI surface

```text
gghstats mcp [flags]

Flags:
  --db PATH                 SQLite path (default: GGHSTATS_DB / platform default)
  --transport stdio|sse     default: stdio
  --addr HOST:PORT          SSE only; default 127.0.0.1:7397
  --token TOKEN             SSE auth; also GGHSTATS_MCP_TOKEN
  --allow-public-bind       SSE only; required to bind non-loopback (still needs token)
  --report-private          include private inherit when true (mirror GGHSTATS_REPORT_PRIVATE)
  --max-days N              hard cap on timeseries window (default 366)
  --max-rows N              hard cap on rows returned per tool (default 5000)
```

Env:

| Env | Role |
|-----|------|
| `GGHSTATS_DB` | DB path (same as rest of CLI) |
| `GGHSTATS_MCP_TOKEN` | SSE bearer token (preferred over flag in prod) |
| `GGHSTATS_REPORT_PRIVATE` | Default for `--report-private` if flag unset |
| `GGHSTATS_MCP_ADDR` | Optional default for `--addr` |

Exit codes: `0` clean shutdown; `1` config/DB/auth bind error; `2` protocol/runtime failure after start (document in man).

Help: `gghstats mcp -h` must not start a server (same pattern as `serve` help).

---

## 7. Transports

### 7.1 stdio (default)

- Client spawns: `gghstats mcp --db …`
- JSON-RPC on stdin/stdout
- `slog` / startup messages → **stderr only**; set default logger to stderr before ServeStdio
- SIGINT/SIGTERM → cancel context → close DB → exit 0

**Claude Desktop / Cursor sketch (docs):**

```json
{
  "mcpServers": {
    "gghstats": {
      "command": "gghstats",
      "args": ["mcp", "--db", "/absolute/path/to/gghstats.db"]
    }
  }
}
```

### 7.2 sse (HTTP + Server-Sent Events)

- For remote/hosted agents against a VPS DB
- Default listen: **`127.0.0.1:7397`** (port pick: document; avoid clash with `serve` 8080)
- Auth: `Authorization: Bearer <token>` **or** `x-api-token: <token>` (align with existing API header name for operator familiarity)
- If `--token` / `GGHSTATS_MCP_TOKEN` **empty**:
  - loopback bind → allow with **startup WARN** (“unauthenticated MCP on loopback”)
  - non-loopback → **refuse to start**
- Non-loopback bind additionally requires `--allow-public-bind` + non-empty token
- TLS: **out of scope** for 1.7.0 — operators put Traefik/Caddy in front (document); raw MCP is HTTP

Prefer path prefix `/mcp` (or library default) — document exact endpoints after library choice in implementation PR.

---

## 8. Tool contracts (MVP)

All tools return **JSON text** (structured object as string content or library structured content if available). Errors: MCP tool error with stable `code` string in message/body.

Visibility: every tool that lists or fetches a repo applies **report scope** (same as HTML/API report surfaces): exclude `hidden=1`, apply `report_policy` + `GGHSTATS_REPORT_PRIVATE` / `--report-private`.

### 8.1 `list_tracked_repos`

| | |
|--|--|
| **Purpose** | Inventory of report-visible repos |
| **Input** | optional `sort` (`clones`\|`views`\|`stars`\|`name`), optional `direction` (`asc`\|`desc`) |
| **Output** | `{ "repos": [ { "name", "repository_id?", "stars", "clones_total?", "views_total?", "github_visibility?", "report_policy?" } ] }` |
| **Limits** | Entire visible set; if > `max-rows`, truncate + `"truncated": true` |

### 8.2 `get_repo_summary`

| | |
|--|--|
| **Purpose** | One-repo aggregate snapshot |
| **Input** | `repo` (string `owner/name`, required) **or** `repository_id` (int, optional once #86) — at least one |
| **Output** | name, id, description, stars/forks, totals views/clones, date extent, top referrer count summary |
| **Errors** | `not_found` (missing or not report-visible — **same message**, no existence leak for hidden) |

### 8.3 `get_traffic_timeseries`

| | |
|--|--|
| **Purpose** | Daily views and/or clones |
| **Input** | `repo` or `repository_id`; `metric` = `views`\|`clones`\|`both` (default `both`); `start_date`, `end_date` (YYYY-MM-DD, required); |
| **Output** | `{ "repo", "repository_id?", "metric", "points": [ { "date", "views_count?", "views_uniques?", "clones_count?", "clones_uniques?" } ] }` |
| **Rules** | Inclusive UTC calendar dates; reject if window > `--max-days`; reject inverted range; empty points OK |
| **Gaps** | Do **not** invent days; omit missing dates (align SYNC-01 / chart gap semantics — document “sparse series”) |

### 8.4 `get_top_referrers`

| | |
|--|--|
| **Purpose** | Historical referrer domains for a repo |
| **Input** | `repo` or `repository_id`; optional `start_date`/`end_date`; optional `limit` (default 20, max 100) |
| **Output** | `{ "repo", "referrers": [ { "referrer", "count", "uniques" } ] }` aggregated across window |
| **Notes** | Snapshot semantics follow store (MAX/replace per SYNC-02 once documented) |

### Deferred tools (not 1.7.0)

- `get_top_paths`, stars timeseries, fleet aggregate, alerts, sync trigger, pin/featured mutate
- Resources / prompts (optional later); MVP = **tools only**

---

## 9. Auth & data exposure matrix

| Mode | Who can call | What they see |
|------|--------------|---------------|
| stdio | Local user who can read DB file + run binary | Report-visible rows only |
| sse + token | Bearer holders | Report-visible rows only |
| sse loopback no token | Local processes | Report-visible; WARN at start |
| sse public no token | **Blocked** | — |
| Hidden / exclude policy repo | Nobody via MCP | Indistinguishable `not_found` |

**Do not** reuse `GGHSTATS_GITHUB_TOKEN` as MCP auth (wrong privilege).  
**May** document that operators can set `GGHSTATS_MCP_TOKEN` equal to `GGHSTATS_API_TOKEN` for convenience — not required.

Constant-time compare for tokens (same hygiene as API token compare from 1.6.8 audit).

---

## 10. Identity interaction (#86)

| When | Behavior |
|------|----------|
| Before identity merge | Tools key by `owner/name` TEXT only |
| After identity A+B | Accept `repository_id`; resolve aliases via `repo_names` / canonical name |
| Rename | `get_*` by old name succeeds if alias exists; output `name` = canonical |
| Name reuse | Never merge; id distinguishes |

MCP PRs may land **before** identity if they use name-only; must rebase to add `repository_id` before tag if #86 is in the same release (preferred).

---

## 11. Observability

stderr logs (English):

- `mcp: transport=stdio db=/path`
- `mcp: transport=sse addr=127.0.0.1:7397 auth=on|off`
- `mcp: tool=get_traffic_timeseries repo=… duration_ms=… rows=…` (debug level; default info without per-tool spam)
- Never log full token; never log raw tool dumps at info

Optional: `--debug` → slog debug on stderr.

No Prometheus in MCP process for 1.7.0 (parking).

---

## 12. PR slicing (#97)

| PR | Content | Depends |
|----|---------|---------|
| **G1** | `internal/mcp` + `gghstats mcp --transport stdio` + 4 tools + RO DB + tests | None (name-keyed OK) |
| **G2** | SSE transport + token / bind guards + tests | G1 |
| **G3** | Wire `repository_id` + alias resolve | G1 + identity B (or C) |
| **G-DOC** | SPEC § MCP, man, README, Claude/Cursor examples, `contrib/gghstats.env.example` notes | G1+G2 |

G1+G2 can merge to develop while identity progresses; G3 before tag if #86 ships same band.

---

## 13. Test plan

### Unit / integration

- [ ] Open missing DB → clear error, exit ≠0
- [ ] RO open: attempt write via MCP connection fails; serve writer still works in parallel test if feasible
- [ ] `list_tracked_repos` omits `hidden=1` and `report_policy=exclude`
- [ ] Private inherit hidden unless `--report-private`
- [ ] `get_repo_summary` hidden → `not_found` (no leak)
- [ ] Timeseries: inverted dates error; > max-days error; sparse dates (no fill)
- [ ] Referrers honor limit cap
- [ ] Token compare: wrong token SSE → 401; right → tools work
- [ ] Non-loopback without token → start error
- [ ] Non-loopback without `--allow-public-bind` → start error
- [ ] Stdio: capture stdout — must be valid MCP framing only (no slog line)

### Manual smoke (dogfood S1)

- [ ] Cursor or Claude Desktop config → `list_tracked_repos`
- [ ] Ask agent for 30-day clone trend on a known repo
- [ ] With `serve` running + sync tick, MCP still answers

### Release gates

- [ ] `make lint` / `make test` / cover floor
- [ ] govulncheck clean for new module
- [ ] Man page + SPEC updated before VERSION cut

---

## 14. Docs deliverables

| Artifact | Content |
|----------|---------|
| SPEC.md | New section: MCP tools, auth, RO DB, transports |
| `contrib/man/man1/gghstats.1` | `mcp` subcommand + flags + env |
| README.md | Short “AI / MCP” subsection + link |
| `docs/api.md` or `docs/mcp.md` | Client configs + tool table (operator-facing) |
| CHANGELOG.md | Added under 1.7.0 |
| `contrib/gghstats.env.example` | `GGHSTATS_MCP_TOKEN`, optional addr |

Prefer new **`docs/mcp.md`** once implemented (keep this plan as pre-ship design).

---

## 15. Security notes for self-hosted (~8k)

1. Default SSE = loopback → safe-ish on VPS if only local agents; still prefer token.
2. Publishing SSE behind Traefik: terminate TLS, forward to loopback, inject/require token.
3. DB file permissions: MCP process user needs **read** on DB + `-wal`/`-shm`; document.
4. Container: do not publish MCP port in Compose examples by default; opt-in profile or comment.
5. stdio over SSH remote: user already has shell = same trust as reading DB.

gghstats-selfhosted follow-up (after app tag): optional compose snippet — **not** required to tag 1.7.0.

---

## 16. Acceptance (close #97)

- [ ] `gghstats mcp` stdio works with a real MCP client
- [ ] SSE mode works with token; public bind guarded
- [ ] Four tools implemented with report-scope + size caps
- [ ] RO DB verified; no write tools
- [ ] SPEC + man + README examples landed
- [ ] Identity fields present if #86 in same release (`repository_id` additive)

---

## 17. Out of scope / parking

| Item | When |
|------|------|
| Write tools (pin, sync, settings) | Never without explicit milestone |
| MCP mounted inside `serve` HTTP | Later optional |
| TLS inside mcp binary | Reverse proxy |
| Streamable HTTP transport (newer MCP) | Evaluate post-1.7 if clients demand |
| Official go-sdk migration | Only if mark3labs insufficient |
| Resources/prompts | Optional 1.7.x |
| Fleet-wide aggregate tool | 1.7.x / demand |
| Multi-DB | Out |

---

## 18. Decision log (LOCKED)

| Decision | Choice |
|----------|--------|
| Library | `mark3labs/mcp-go` pinned tag |
| Process | Separate `gghstats mcp` (not inside serve) |
| Default transport | stdio |
| SSE default bind | `127.0.0.1:7397` |
| Auth | Token required for non-loopback; warn if loopback unauthenticated |
| Tools | 4 read-only listed above |
| Visibility | Report-scope parity with API/HTML |
| GitHub I/O | None from MCP |
| Dep carve-out | Allowed for MCP track only |

---

*Created 2026-11-21 — detailed companion to plan-v1.7.0.md Track B (#97)*
