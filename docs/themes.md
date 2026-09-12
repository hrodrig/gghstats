# Built-in theme palettes (gghstats)

The dashboard ships **three first-class themes** — **light**, **dark**, and **midnight** — cycled from the sidebar (stored in `localStorage` under `gghstats-theme`). Each is defined as CSS custom properties prefixed `--brutal-*` in `web/static/app.css`. The default is **neo-brutalist light**.

A fourth option — an operator-supplied stylesheet via `GGHSTATS_CUSTOM_CSS` (see [`contrib/themes/README.md`](../contrib/themes/README.md)) — is **not** a built-in theme. It layers on top and is out of scope for the token contract below.

## Source of truth

- Light + dark blocks: `web/static/app.css` → `body.app-brutalist` and `html[data-bs-theme="dark"] body.app-brutalist`.
- Midnight overrides: `html[data-bs-theme="midnight"] body.app-brutalist` (further down the same file).

## Token reference

A token's meaning is the same across themes; only its value changes.

| Token | Purpose |
|-------|---------|
| `--brutal-text` | Default foreground text. |
| `--brutal-paper` | Page background. |
| `--brutal-surface` | Surfaces that sit on the paper (header, sidebar, search panel). |
| `--brutal-elevated` | Cards, wells, modal/panel bodies. |
| `--brutal-border-ink` | Borders and the hard-shadow offset color. |
| `--brutal-shadow-core` | Hard (stamp) shadow color. |
| `--brutal-accent` | Primary accent (buttons, chart rims, highlights). |
| `--brutal-accent-2` | Secondary accent (info, focus outlines, secondary chart rim). |
| `--brutal-muted` | Muted/secondary text. |
| `--brutal-chart-well` | Chart plot area background. |
| `--brutal-chart-rim` | Chart plot area border. |
| `--brutal-nav-active-bg` / `-fg` | Active sidebar/language button fill and text. |
| `--brutal-search-cta` / `-shadow` | Search/Compare/export CTA fill and its shadow. |
| `--brutal-success` / `--brutal-warning` | Status accents. |

## Palettes

### Light (default — neo-brutalist)

| Token | Hex |
|-------|-----|
| `--brutal-text` | `#1a1a1b` |
| `--brutal-paper` | `#e0e0e0` |
| `--brutal-surface` | `#ecece8` |
| `--brutal-elevated` | `#ffffff` |
| `--brutal-border-ink` | `#121212` |
| `--brutal-shadow-core` | `#121212` |
| `--brutal-accent` | `#e63946` |
| `--brutal-accent-2` | `#1d3557` |
| `--brutal-muted` | `#5c5c58` |
| `--brutal-chart-well` | `#f4f4f0` |
| `--brutal-chart-rim` | `#121212` |
| `--brutal-nav-active-bg` / `-fg` | `#121212` / `#f7f7f2` |
| `--brutal-search-cta` / `-shadow` | `#fc8763` / `#1a0d06` |

Swatches (text · paper · surface · elevated · border · accent · accent-2 · muted):

<div style="line-height:2;font-family:monospace">
<span style="background:#1a1a1b;color:#fff;padding:2px 8px">#1a1a1b</span>
<span style="background:#e0e0e0;color:#121212;padding:2px 8px">#e0e0e0</span>
<span style="background:#ecece8;color:#121212;padding:2px 8px">#ecece8</span>
<span style="background:#ffffff;color:#121212;padding:2px 8px;border:1px solid #999">#ffffff</span>
<span style="background:#121212;color:#fff;padding:2px 8px">#121212</span>
<span style="background:#e63946;color:#fff;padding:2px 8px">#e63946</span>
<span style="background:#1d3557;color:#fff;padding:2px 8px">#1d3557</span>
<span style="background:#5c5c58;color:#fff;padding:2px 8px">#5c5c58</span>
</div>

### Dark

| Token | Hex |
|-------|-----|
| `--brutal-text` | `#e8e8e3` |
| `--brutal-paper` | `#121212` |
| `--brutal-surface` | `#1a1a1b` |
| `--brutal-elevated` | `#1e1e22` |
| `--brutal-border-ink` | `#eaeae5` |
| `--brutal-shadow-core` | `#000000` |
| `--brutal-accent` | `#7cfc00` |
| `--brutal-accent-2` | `#00e5ff` |
| `--brutal-muted` | `#9a9a95` |
| `--brutal-chart-well` | `#2d2d32` |
| `--brutal-chart-rim` | `#7cfc00` |
| `--brutal-nav-active-bg` / `-fg` | `#7cfc00` / `#121212` |
| `--brutal-search-cta` / `-shadow` | `#ff734a` / `#1a0d06` |

Swatches:

<div style="line-height:2;font-family:monospace">
<span style="background:#e8e8e3;color:#121212;padding:2px 8px">#e8e8e3</span>
<span style="background:#121212;color:#e8e8e3;padding:2px 8px;border:1px solid #555">#121212</span>
<span style="background:#1a1a1b;color:#e8e8e3;padding:2px 8px;border:1px solid #555">#1a1a1b</span>
<span style="background:#1e1e22;color:#e8e8e3;padding:2px 8px;border:1px solid #555">#1e1e22</span>
<span style="background:#eaeae5;color:#121212;padding:2px 8px">#eaeae5</span>
<span style="background:#7cfc00;color:#121212;padding:2px 8px">#7cfc00</span>
<span style="background:#00e5ff;color:#121212;padding:2px 8px">#00e5ff</span>
<span style="background:#9a9a95;color:#121212;padding:2px 8px">#9a9a95</span>
</div>

### Midnight

| Token | Hex |
|-------|-----|
| `--brutal-text` | `#f3f7fb` |
| `--brutal-paper` | `#030508` |
| `--brutal-surface` | `#080d14` |
| `--brutal-elevated` | `#0d1520` |
| `--brutal-border-ink` | `#d8e5f2` |
| `--brutal-shadow-core` | `#000000` |
| `--brutal-accent` | `#65a4ff` |
| `--brutal-accent-2` | `#70d8e8` |
| `--brutal-muted` | `#9aaabd` |
| `--brutal-chart-well` | `#0d1520` |
| `--brutal-chart-rim` | `#30445e` |
| `--brutal-nav-active-bg` / `-fg` | `#12305f` / `#ffffff` |
| `--brutal-search-cta` / `-shadow` | `#4f8df7` / `#000000` |

Swatches:

<div style="line-height:2;font-family:monospace">
<span style="background:#f3f7fb;color:#030508;padding:2px 8px">#f3f7fb</span>
<span style="background:#030508;color:#f3f7fb;padding:2px 8px;border:1px solid #555">#030508</span>
<span style="background:#080d14;color:#f3f7fb;padding:2px 8px;border:1px solid #555">#080d14</span>
<span style="background:#0d1520;color:#f3f7fb;padding:2px 8px;border:1px solid #555">#0d1520</span>
<span style="background:#d8e5f2;color:#030508;padding:2px 8px">#d8e5f2</span>
<span style="background:#65a4ff;color:#030508;padding:2px 8px">#65a4ff</span>
<span style="background:#70d8e8;color:#030508;padding:2px 8px">#70d8e8</span>
<span style="background:#9aaabd;color:#030508;padding:2px 8px">#9aaabd</span>
</div>

## Component contract

Themes are token-based: components must reference `--brutal-*` variables, **not** raw Bootstrap color utilities, so they stay correct under all three themes.

- ✅ **Cards, buttons, panes, nav, badges, muted text** → use `--brutal-*` (e.g. `--brutal-elevated`, `--brutal-border-ink`, `--brutal-text`, `--brutal-muted`) or a dedicated app class styled from those tokens.
- ❌ **Unthemed Bootstrap subtle/body-secondary utilities** (`bg-body-secondary` + `text-body`, `bg-secondary-subtle` + `text-secondary`) do **not** flip with `data-bs-theme` and can render light-text-on-light-pill under Midnight/Dark. See [#60](https://github.com/hrodrig/gghstats/issues/60) (Featured stars badge).
- Featured-card chips (stars, fork badge, empty-state command) use the dedicated classes `app-featured-stars` / `app-featured-fork-badge` / `app-featured-empty-cmd` in `web/static/app.css`, styled from `--brutal-chart-well` + `--brutal-border-ink` + `--brutal-text`. New Featured/KPI chips must follow the same pattern.

When adding or changing a token, update this doc **and** the matching blocks in `web/static/app.css` in the same commit.
