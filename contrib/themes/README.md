# Official example theme gallery (gghstats)

The default UI is intentionally **neo-brutalist** (strong borders, hard shadows, monospace). That look is not for everyone: some teams want a **quieter**, **simpler**, or **brand-aligned** dashboard while keeping the same app. Optional **`GGHSTATS_CUSTOM_CSS`** exists so self-hosted operators can layer their own stylesheet **without forking** or rebuilding the binary.

This directory ships **five** optional CSS themes as **copy-paste starting points**. They are **not** loaded automatically: copy one to your server or container volume, set **`GGHSTATS_CUSTOM_CSS`** to that file path, and restart **`gghstats serve`**.

The app loads styles in this order: **Bootstrap** → **`app.css`** (built-in neo-brutalist) → **your file** (via `GET /theme/custom.css`). Override the CSS custom properties prefixed with `--brutal-*` (and optionally `--bs-*`) on `body.app-brutalist` and, for dark mode, `html[data-bs-theme="dark"] body.app-brutalist`. For a **Bootstrap-like** look (not just recolouring), start from [`example-bootstrap-plain.css`](example-bootstrap-plain.css): it resets typography, borders, shadows, and several component rules.

### Screenshot (Bootstrap-plain)

Repository index with [`example-bootstrap-plain.css`](example-bootstrap-plain.css) enabled via `GGHSTATS_CUSTOM_CSS` (light mode):

![gghstats dashboard — Bootstrap-plain optional theme](../../assets/gghstats-main-theme-bootstrap-plain.png)

## Gallery (five variants)

| # | File | Summary |
|---|------|---------|
| 1 | [`example-softer-brutal.css`](example-softer-brutal.css) | Thinner borders and smaller hard shadows; same structure, slightly calmer. |
| 2 | [`example-ocean-accent.css`](example-ocean-accent.css) | Teal / coral accent swap in light and dark. |
| 3 | [`example-mono-paper.css`](example-mono-paper.css) | Cream “paper” light palette and graphite dark; near-monochrome with restrained accents. |
| 4 | [`example-violet-citrus.css`](example-violet-citrus.css) | Violet primary and lime/citrus secondary for a sharper tech contrast. |
| 5 | [`example-bootstrap-plain.css`](example-bootstrap-plain.css) | **Near-vanilla Bootstrap:** system sans, 1px borders, no offset “stamp” shadows, rounded cards/buttons — strips most neo-brutalist chrome (not only colours). |

Pick one file as a base, duplicate it under a new name, and adjust until it matches your brand. Palette-only tweaks (rows 1–4) stay small; the plain-Bootstrap starter (row 5) intentionally overrides more component rules so the UI reads closer to stock Bootstrap on top of the same HTML.

## Safety notes

- Use a **regular file** path (not a directory). Symlinks are resolved by the OS `stat`/`open` stack; only expose paths you trust.
- Maximum served size is **2 MiB** (see `internal/server/custom_theme.go`).

## References

- Built-in tokens: `web/static/app.css`
- Configuration: root `README.md` (environment variables), `.env.example`
