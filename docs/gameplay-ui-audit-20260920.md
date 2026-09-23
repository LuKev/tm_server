# Gameplay UI audit — 2026-09-20

**2026-09-23 scope correction:** The user wants to preserve the existing
BGA/Snellman hybrid gameplay arrangement. The layout replacement and relocation
proposals below are superseded by
[the modernization plan](gameplay-ui-modernization-plan.md). Keep panel positions
and layout controls; improve the presentation and implementation in place.

Scope: gameplay board, player panels, shared tiles, action flow, dialogs, and
layout. Lobby/import styling is excluded. This is a source and build audit;
no live browser visual, touch, contrast, or performance assessment was performed.
Recommendations below are proposals, not approved implementation decisions.

## Assessment

The framework is not the main problem. The client declares React 19,
TypeScript 5.9, Vite 7, and Zustand 5. The gameplay presentation combines
imperative canvas drawing, utility classes, component CSS, and inline styles
without a shared presentation contract. Canvas itself is not obsolete.

## Findings, in priority order

1. **Repair the styling pipeline before redesigning components.**
   `client/package.json` declares Tailwind 4 while the repository root declares
   Tailwind 3. Root `postcss.config.js` uses the v3 plugin interface and root
   `tailwind.config.js` has an empty content list. `client/src/index.css` uses
   v3 directives; the client's Vite configuration has no Tailwind Vite plugin.
   `PlayerSummaryBar.tsx` explicitly bypasses utilities because production may
   omit them. Gameplay relies on those utilities for spacing, color, layout,
   and the sticky decision strip. This is a confirmed configuration mismatch;
   actual missing selectors must be checked in a freshly served production build.
   The Bazel build wrapper copies only the client into a temporary directory,
   so it does not reproduce discovery of the repository's parent PostCSS config.
   Make configuration self-contained under `client`, choose one Tailwind major,
   and assert computed styles against the production build.

2. **Fix the power-spade client/server contract.**
   `Game.tsx` sends `power_action_claim` with one target and no
   `spadeActionVersion`; `server/internal/websocket/client.go` requires version 2
   for power-spade actions. The current request would be rejected by this server.
   Implement a draft containing all destinations and optional building choices,
   show its cost/result, then submit the complete versioned action atomically.
   Verify through the real WebSocket path, including cancel and invalid choices.

3. **Replace the default editable dashboard with a designed game layout.**
   `useGameLayout.ts` starts unlocked, specifies only lg/md layouts although the
   views expose five breakpoints, and couples panel heights to widths. Player
   board height calculations use different ratios in different callbacks.
   `PlayerBoards.tsx` scales typography with a ResizeObserver, down to 10px.
   This makes readability dependent on panel geometry. Smaller-screen failures
   are a risk, not a browser-verified result of this audit.
   Use CSS Grid for a stable desktop composition; keep customization opt-in.
   Define deliberate tablet/mobile layouts instead of shrinking everything.

4. **Define one visual and interaction system.**
   Player boards mix parchment textures and white/gray cards; scoring and power
   tiles use heavy black borders; summary cards use separate inline styling.
   Yellow marks current round, current player, and hover/selectability; town
   selection uses green. There are no CSS custom-property tokens in source CSS.
   Establish shared spacing, type sizes, surfaces, borders, focus rings, and
   states: available, selected, unavailable, resolving, and used. Keep faction,
   terrain, and resource colors distinct from interface status colors.
   Consolidate Button, Panel, ResourceAmount, TileButton, Tooltip, and Dialog.
   Preserve meaningful board-game shapes and resource symbols.

5. **Make the next decision the center of the interaction.**
   A decision strip already exists; evolve it rather than adding another status
   surface. Show whose decision it is, the required choice, legal targets,
   draft cost/result, and confirm/cancel in one persistent action area.
   `GameBoard.tsx` currently supplies hovered-hex highlighting, not an explicit
   legal/selected-target presentation. Preserve board context during routine
   choices instead of repeatedly covering it with dialogs. Mandatory reactions
   must identify the decision owner even when it is someone else's main turn.

6. **Repair keyboard and assistive-technology access.**
   `shared/Modal.tsx` handles Escape and backdrop clicks but has no dialog role,
   accessible title association, focus containment, or focus restoration.
   The board canvas exposes click/mouse movement without keyboard navigation or
   semantic fallback; cult tracks also use canvas. Add accessible dialog behavior,
   labeled resource/tile controls, and keyboard access to board/cult choices.
   SVG would simplify per-hex focus and event targets, but is not automatically
   accessible; labels, focus order, and keyboard behavior still need designing.

7. **Separate action state from presentation.**
   `Game.tsx` is 3,158 lines and `PlayerBoards.tsx` is 959 lines. Game combines
   action orchestration, faction exceptions, selections, dialogs, and rendering.
   Extract typed action drafts and controllers by interaction, with explicit
   choose/preview/submit/cancel states. Keep authoritative rules on the server.
   Share the board presentation between game and replay without imposing the
   same control layout on both.

## Proposed product direction

Use a quiet, warm-neutral game table with strong text contrast and restrained
panel borders. Let terrain, pieces, and resources carry most color. Standardize
icon weights and number alignment. Avoid turning every game element into a
generic application card or making small text scale with tile dimensions.

Desktop composition: a compact round/turn/opponent summary above a dominant
board; cult tracks and round scoring beside it; the local player's resources
and action controls immediately below. Tile markets remain visible when space
allows and expand for selection. Opponent details and settings are secondary.
On narrow screens, preserve readable board controls and use explicit panels
or tabs for secondary information. Assess pan/zoom with touch before choosing
a board rendering replacement.

Keep React, TypeScript, Vite, and Zustand. Prefer CSS Grid and shared CSS tokens
for layout. Either finish Tailwind 4 integration or deliberately replace its
usage; do not continue the mixed setup. Prototype an SVG board/cult track as
one isolated experiment, comparing fidelity, focus, touch, and measured render
cost before replacing canvas. No evidence currently justifies WebGL or a new
application framework.

## Suggested delivery sequence

1. Fix the style build boundary and power-spade contract. Capture representative
   production-build states before visual changes.
2. Implement shared tokens/primitives and one fixed gameplay layout. Pilot one
   complete action flow, from selection to server response.
3. Migrate player boards, cult tracks, and tile markets to that system. Decide
   canvas versus SVG from the prototype, then add deliberate narrow-screen UX.
4. Add production screenshot coverage and keyboard/touch checks for setup,
   normal turn, leech, spades, town/favor selection, and game end. Include 1v1
   and multiplayer, long names, full resources, errors, and reconnect states.

Existing Playwright tests cover actions and include a 1280x720 decision-strip
position test. No screenshot-baseline assertions or axe integration were found.
Add these alongside the existing functional suite, not instead of it.

## Verification and references

Fresh execution passed:
`cd server && bazel test //:client_build_test --test_output=all --nocache_test_results`.
Vite reported a 550.54 kB JavaScript chunk (159.05 kB gzip) and its size warning;
this is not measured gameplay slowness. No application source was changed and
the browser suite was not run for this audit.

- Tailwind migration requirements: https://tailwindcss.com/docs/upgrade-guide
- Dialog behavior: https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/
- React release families: https://react.dev/versions
