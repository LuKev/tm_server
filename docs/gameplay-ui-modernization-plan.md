# Gameplay UI modernization plan

Updated 2026-09-23. Status: implemented and verified.

## Product constraint

Preserve the current BGA/Snellman hybrid composition. The user likes where the
game elements are. Improve visual consistency, implementation reliability, and
interaction quality within those locations.

Preserve the panel identities, default positions and relative sizes in
`useGameLayout.ts`, drag/resize/lock/reset behavior, board coordinates, tile
ordering, resource information, faction colors, and gameplay density. Preserve
the existing decision strip and dialog locations. Do not introduce a new action
sidebar, hide opponent information, or replace the layout with CSS Grid.
Small internal padding and sizing corrections are allowed, but relocating panels
or changing the information hierarchy is a separate product decision.

Lobby redesign, framework replacement, a canvas-to-SVG rewrite, new mobile
navigation, and game-rule changes are outside this plan. Shared styling changes
must still be checked for unintended effects on lobby/import and replay.

## Technical direction

- Keep React, TypeScript, Zustand, react-grid-layout, and the canvas renderers.
- Finish client-local Tailwind 4 integration using its Vite plugin. Use it for
  ordinary utilities; shared CSS variables own the visual design. Use scoped
  component CSS for complex tiles and board chrome, with inline styles reserved
  for dynamic geometry and values. Avoid a second large component framework.
- Add a gameplay theme scope so tokens and component rules do not restyle the
  whole application. Preserve the initial light game surface and recognizable
  game art; start with consistent borders, spacing, type, and interaction states.
- Share visual values with canvas through a small theme adapter that reads the
  gameplay tokens. Keep terrain/faction/resource palettes separate from UI
  status colors. Do not hard-code a second copy of the interface palette.
- Extract action controllers only as their flows are touched. Keep game rules
  authoritative on the server; no generic workflow framework or client rules engine.

## Delivery units

### 1. Baseline and production-render harness

Files: `client/e2e/`, existing game-state/mock-WebSocket helpers,
`client/playwright.config.ts`, `server/tools/client_*`, `server/BUILD.bazel`.

Capture current gameplay at 1280x720 and 1920x1080, two and four players, with
setup, a normal turn, leech, town/favor selection, a dialog, and end scoring.
Record panel bounds and relationships independently of cosmetic snapshots.
Include one dragged/resized arrangement and lock/reset behavior. Freeze timers,
animations, fixtures, and fonts for reproducible screenshots.

Add a Bazel-backed production-browser target (proposed name:
`//:client_production_ui_test`). Build and serve the client from an isolated
client-only directory, with a real-backend proxy for action checks. Existing
browser verification runs Vite development mode; keep that suite as well.
Reuse current helpers, repairing any fixture/type defects encountered first.

Exit: reproducible screenshots and geometry assertions; production page loads
without console errors; the harness can assert computed layout styles. A known
broken baseline is evidence, not a requirement to preserve the bug.

### 2. Repair the styling build boundary

Files: `client/package.json`, lockfile, `client/vite.config.ts`,
`client/src/index.css`, and relevant Tailwind/PostCSS configuration.

Use one client-owned Tailwind 4 pipeline. Explicitly prevent parent PostCSS
configuration from applying the old Tailwind plugin. Remove obsolete root
configuration/dependencies only after checking other consumers. Keep deployment
asset base paths and router basename behavior intact.

Audit class compatibility, dynamically assembled color classes, cascade layers,
and Preflight/reset changes. Restoring previously missing utilities can visibly
change spacing and sizing: reconcile those effects against the accepted panel
geometry before proceeding. Do not mix cosmetic redesign into this change.

Exit: development and isolated production builds agree on representative flex,
spacing, color, and sticky-position styles. Layout lock/reset still work. No
unintentional changes to panel placement or other routes.

### 3. Shared visual language and an in-place pilot

Files: new `client/src/styles/game-theme.css`,
`client/src/components/shared/`, `PlayerSummaryBar.tsx`, and `PowerActions.*`.

Create a compact token set: surface/text/border colors, spacing steps, typography,
corner radii, icon sizes, and focus/selection outlines. Starting sizes: 14px body,
12px secondary labels, 16px resource values, 4/8/12/16px spacing, and 4/8px radii.
These are pilot values; verify they fit the current panels before standardizing.
Use tabular resource numbers and consistent icon alignment. Avoid applying a
generic card treatment to every board-game object.

Build only the primitives required by the pilot: GamePanel, ActionButton,
ResourceAmount, and shared selectable-tile state styling. Apply them to the
existing player summary and power-action tiles in their current locations.
Provide screenshots of normal, hovered, focused, selected, disabled, and used
states. Use outlines, labels, and markers so state is not communicated by color
alone. Current turn/round and selected action must remain distinguishable.

Exit: a concrete before/after preview with unchanged panel geometry, readable
resources, consistent controls, and visible keyboard focus. Use this preview
as the design checkpoint before broad rollout, not an abstract mood board.

### 4. Migrate gameplay visuals panel by panel

Order: scoring/passing/favor/town tiles; player boards and conversions; board and
cult-track chrome; dialogs and decision-strip presentation. Keep each family in
a separately reviewable commit. Inspect replay whenever shared pieces change.

Preserve tile shapes, game symbols, ordering, faction distinctions, and visible
information. Standardize border weights, headings, spacing, hit areas, disabled
states, and tooltips. Keep the board drawing and hit-test coordinate systems.
Replace blanket font scaling only where it harms readability; bound text sizes
and use intentional internal overflow without moving panels or hiding controls.
Check resizing across the existing supported range before replacing any formula.

Exit: no clipped required controls, no new overlap, no lost information, and
consistent styling at both desktop viewports and in a resized layout. Inspect
small screens for regressions; a new mobile composition is not required here.

### 5. Action and accessibility repairs in existing locations

Files: `Game.tsx`, `utils/pendingDecision.ts`, `shared/Modal.tsx`,
`GameBoard/HexGridCanvas.tsx`, `CultTracks/CultTracks.tsx`, and focused tests.

First fix the existing power-spade request mismatch in a separate functional
commit: the server requires `spadeActionVersion: 2` and atomic destinations.
Use a typed draft for targets and optional building, keeping the existing entry
points and dialog. Verify one/two spades, second destination, cancel, invalid
targets, and server rejection through the real WebSocket path. This repair can
follow unit 1 independently of the visual work; it should not wait for unit 4.

Give dialogs accessible names, initial focus, focus containment/restoration,
and correct Escape behavior. Preserve action confirmation preferences. Add
keyboard/pointer equivalents for board and cult choices, with visible focused
targets and semantic controls synchronized to the canvas. Do not replace canvas
as a prerequisite. Add accessible descriptions for icon-only tile rewards.

Extract the touched action draft/controller from `Game.tsx`; reuse pending-decision
decoding. Make submitting, server success, failure, and cancel explicit so a
rejected action does not look successful or erase a useful selection. Improve
instructions and owner/status text in the existing decision strip.

Exit: representative flows work by mouse and keyboard; dialog focus returns
correctly; failed requests preserve useful context; no duplicate submissions or
changes to action rules. Finish remaining accessibility repairs by component.

### 6. Final integration and cleanup

Review setup, ordinary actions, leech, spades, town/favor rewards, passing, game
end, spectator, replay, and reconnect/error states. Exercise multiple factions,
non-base map geometry, long names, 1v1 and multiplayer. Use deterministic visual
fixtures for appearance and real-server scenarios for behavior.

Remove superseded style declarations and temporary compatibility workarounds
only after their consumers migrate. Update NOTES.md to replace the old warning
about unreliable Tailwind once production-browser evidence supports that change.

Exit: the same recognizable game arrangement, cohesive visuals, unchanged layout
controls, and passing production-build, browser-action, and visual checks.

## Verification and review policy

For implementation, run from `server/`:

```sh
bazel test //:client_build_test --test_output=errors --nocache_test_results
bazel test //:client_playwright_test --test_output=errors --nocache_test_results --test_timeout=3600
bazel test //:client_production_ui_test --test_output=errors --nocache_test_results --test_timeout=3600
```

The browser targets use separate ports and can run together. The longer timeout
allows the recorded full-game click scenarios to finish.
Current shell wrappers read client source from the workspace rather
than declared Bazel inputs; use `--nocache_test_results` when validating fresh
client edits unless that input tracking is repaired. Visually inspect snapshot
changes; do not bulk-accept baselines to make failures pass. Include screenshot
pairs and relevant interaction results with each visual delivery unit.

Implementation and verification results (2026-09-23):

- Client-local Tailwind integration, scoped theme, shared primitives, panel styling,
  keyboard board/cult access, and native dialog focus behavior are implemented.
  Default layout definitions in `useGameLayout.ts` are unchanged.
- Both complete browser targets passed: 49 checks each, including all three
  recorded games with exact final-score matches. The two preexisting opt-in
  observer/video replay tests remain skipped; no new skips were introduced.
- Four reviewed Chromium/macOS visual baselines pass in both modes. Checks cover
  desktop geometry, drag/resize/lock/reset, tile states, setup/rewards, replay,
  narrow custom-map spectators, long names, dialogs, and reconnect recovery.
- Real-server single/double power-spade rejection, corrected resubmission, and
  atomic acceptance pass. Water favor uses a named native button above its artwork,
  verified with real pointer clicks rather than injected DOM clicks.
- Browser action scripts were regenerated from the existing server exporter.
  Independent S69/S60/S61 fixed-score server assertions pass. Only the test exporter
  changed to ignore historical zero-burn no-ops; runtime game rules and server
  expected scores were not changed.
- After the complete runs, a keyboard fallback for smaller replacement maps was
  added and its expanded regression passed in both modes. A fresh production
  build also passed after that final change.

## Evidence map

| Requirement | Evidence |
| --- | --- |
| Existing desktop arrangement and layout controls | `gameplay-visual.spec.ts`: four baselines, panel bounds, actual drag/resize, lock/reset; `useGameLayout.ts` unchanged |
| Isolated production CSS and build | `client_production_ui_test` copies the client only; computed flex/sticky assertions; `client_build_test` passes |
| Shared component states | `gameplay-modernization.spec.ts`: normal/hover/focus/selected/used captures, Yetis cost display, accessible rewards |
| Mouse and keyboard spades | Draft contract tests plus real-server single/double-spade acceptance, rejection and corrected resubmission |
| Dialog and reconnect | Focus containment, Escape/restoration/reopen, opaque end-scoring capture, disconnected submission recovery |
| Replay and constrained widths | Two replay widths and controls; custom negative-coordinate map, long names, spectator controls and narrow non-overlap |
| Full gameplay regression | Server S69/S60/S61 fixed-score assertions pass; all three recorded games complete with exact scores in development and production |

Complete-run logs and captures are retained locally in
`artifacts/ui-modernization/final-production/` and `final-development/`; the
additional keyboard/build checks are in `final-checks/`. Earlier pilot captures
are in `before/` and `pilot/`, and the reviewed focused captures in `final-visual/`. The screenshot baselines under
`client/e2e/gameplay-visual.spec.ts-snapshots/` are the durable regression inputs.

Tailwind integration reference:
https://tailwindcss.com/docs/installation/using-vite
