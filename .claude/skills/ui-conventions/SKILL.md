---
name: ui-conventions
description: Established UI/UX decisions for the ramp TUI (layout, keys, theming, destructive prompts, status bar). Read before adding or changing any TUI screen, key binding, footer hint, or visual style so new work matches decisions already made and doesn't relitigate them.
---

# ramp UI conventions

Decisions distilled from the project's PR/commit history. Each was made deliberately — several reversed an earlier approach, so don't reintroduce the old pattern.

## Identity

- The tool presents as **ramp** with the `░▒▓` luminance-ramp wordmark, right-aligned in the preview header and shown in the status bar summary (`N animations · ramp`). The wordmark is the first thing dropped when width runs out. Repo/module intentionally remains `ascii-tui`.

## Layout ("tuxedo" style)

- **Borderless three-column gallery**: library list, preview (with a one-line header: `▸ name` left, wordmark right), detail column (dimensions, length, render options, file size, modified). Bordered panels and hand-drawn border primitives were removed in the tuxedo overhaul — do not add rounded-border panels back.
- Selection is shown as a **full-width background bar** on the list row, not a marker-only highlight.
- The whole screen is painted with the theme background; when embedding frame art, **re-inject the bg SGR code after any embedded reset** so the fill survives.
- Header/detail stay minimal: metadata lives in the detail column, not the header; file paths render in normal text with a dim `path` label (not all-dim).
- Small terminals: gallery collapses to a single full-width panel below 56 cols or 12 rows.
- **Centering gotcha**: `lipgloss.Place(..., Center, ...)` centers each line independently. Pad every row of a menu/panel to one shared width (`fitLine`) so the block centers as a unit — this caused the keybinds staircase bug (PR #15).

## Keys

- Single source of truth is `internal/tui/keys.go` keymap tables; dispatch with `key.Matches`, never raw `key.String()` switches.
- **ctrl+c is the only quit key.** `q` was deliberately freed for rebinding and must never quit or be reserved. Footers do not advertise ctrl+c.
- Player defaults: next/prev on `>`/`<`; left/right arrows **scrub frame-by-frame and pause** (never time-based seeking — too coarse for short GIFs), with hold-to-accelerate 1→2→4→8 frames inferred from OS auto-repeat timing; `,`/`.` frame-step; `+`/`-` speed.
- Speed adjusts by **additive 0.25 steps** (1x → 1.25x → 1.5x), clamped 0.25x–8x. Multiplicative stepping was rejected because it never lands on round values.
- Player keys are user-rebindable via the `k` keybinds screen and `[keys]` in config.toml (`space` spelled out). Reserved: esc, ctrl+c, `?`. Reject collisions with other actions; save on every change; an empty list falls back per-action to defaults.
- `?` opens a centered help overlay from any screen; any key closes it and is swallowed, but non-key messages (ticks, resize, preview loads) must still flow through so playback doesn't freeze underneath.

## Destructive and irreversible actions

- Delete uses a **centered Cancel/Delete menu** (not a status-bar y/n — that was replaced). Cancel is highlighted first so a stray enter never deletes; arrows/j/k/tab move, enter commits, esc cancels, and **every other key is swallowed** while the prompt is up.
- After deleting, clamp the list cursor so a row stays selected.
- Never silently overwrite: rename refuses name collisions; export writes `<name>-2.gif`/`-3` suffixes instead of clobbering.

## Status bar and footers

- Three zones: **mode chip** (NORMAL/ADD/RENAME/DELETE/FILTER; PLAYING/PAUSED; RENDER), key hints, right-aligned summary.
- Key hints right-align via `bubbles/help` so they truncate first as the terminal narrows.
- Transient messages go through `flashStatus` with its generation counter so a newer message is never wiped by an older clear timer. Don't hand-roll clear timers.
- Error bars stay minimal (`esc back` only).

## Theme

- One `theme` struct + `styles` bundle built once via `newStyles` and plumbed into every sub-model. No package-level style vars.
- Four presets (pink, matrix, amber, ocean) cycle with `t` and persist by name in `[theme].name`; an empty name means custom colors from config. New theme fields must keep old shorter configs working.

## Async UI state

- Anything triggered by rapid input (preview on selection change, resize refits) is **debounced (~150ms), cached per entry, and guarded by generation counters** so stale background results are dropped.
- UI-triggered re-renders that change the stored animation (resize refit, filter-bg toggle) **persist back to disk** so the work isn't redone next run.
- Speed-scaled tick delays floor at 10ms so a high multiplier can't spin the loop.

## Inputs and prompts

- The add-gif prompt is a recursive, gif-only fuzzy finder over relative paths (no directory rows, no descend-into-dir step). Walks are capped (depth 6, 500 results, 20k entries) and cached per directory root; hidden entries are skipped unless the query starts with `.`. `~` expands everywhere paths are accepted.

## Verifying UI changes

- Don't trust `capture-pane` spacing under psmux — dump the raw Go `view()` output to check alignment, then drive the built binary for behavior.
- Alignment/layout fixes get a rune-aware regression test (multibyte markers like `▸` break byte-column checks).
