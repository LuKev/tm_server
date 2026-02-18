# Workspace Notes (Snellman Replay)

- Repo rule: use Bazel (workspace: `/Users/kevin/projects/tm_server/server`). Avoid `go test`.

- Fixture corpus (certification):
  - Snellman ledger fixtures: `server/internal/replay/testdata/snellman_batch/` (S67-S69, G1-G7 = 21 games).
  - Additional batch: `server/internal/replay/testdata/snellman_batch_s64_66/` (S64-S66, G1-G7 = 21 games).
  - Expected totals oracle: `server/internal/replay/testdata/snellman_batch/manifest.json`.
  - Batch test: `server/internal/replay/snellman_batch_replay_test.go` runs `ImportText(...,"snellman") -> StartReplay -> JumpTo(end)` and asserts `state.FinalScoring[*].TotalVP`.
  - Dropout policy: fixtures containing `dropped from the game` are skipped in batch/final-score and ledger resource-matching tests (`snellman_batch_s64_66_replay_test.go`, `snellman_ledger_resources_test.go`).
  - Fetcher: `scripts/fetch_snellman_batch.py --output-dir ... --seasons 64,65,66` writes fixture `.txt` and `manifest.json`.

- Cleanup timing (critical): Snellman logs can contain late reactions after the final `PASS` of a round (leeches, Cultists `+TRACK`, etc.). Cleanup runs at the round boundary (processing the next `RoundStartItem`), not immediately at `AllPlayersPassed()`. See `server/internal/replay/simulator.go`.
  - End-of-log replay fallback: when `JumpTo(end)` lands in `PhaseAction` on round 6, force cleanup/final scoring even if `AllPlayersPassed()` is false (dropped-player logs can omit explicit final `PASS` for the dropped faction).

- Snellman ledger interpretation:
  - Each action row includes the player’s post-action totals (VP/resources/power bowls/cults).
  - Trailing leech markers like `3 1` on an action row indicate later leech/decline rows tied to that action (use these to bind `L`/`DL` to the correct source event).

- Leech semantics (Snellman parity + engine correctness):
  - `PowerLeechOffer.Amount` is the offered amount (adjacent building power sum) and is not capped by current Bowl I/II capacity. Capacity limits the actual gained power at acceptance time (`ResourcePool.AcceptPowerLeech`).
  - Snellman can log `Leech N from X` where the deltas show only `k < N` power gained and `k-1` VP lost.
  - Snellman can also include `Leech ...` rows when Bowl I+II are empty. Replay treats this as an automatic decline/no-op when no offer is pending (see `notation.LogAcceptLeechAction.Execute`).
  - Cultists leech-bonus tracking must be per event, not per source player: offers carry an `EventID` and Cultists pending bonuses are keyed by `EventID` (`server/internal/game/state.go`, `server/internal/game/action_power_leech.go`, `server/internal/game/resources.go`).

- Income-phase ordering (Snellman quirks):
  - Snellman can include actions between income blocks (pre-income interlude) and actions embedded inside income rows (post-income embedded action).
  - Converter tags these as `@<token>` (pre-income) and `^<token>` (post-income). See `server/internal/notation/snellman_to_concise.go`, `server/internal/notation/parser.go`, `server/internal/replay/simulator.go`.

- Converter gotchas (`server/internal/notation/snellman_to_concise.go`):
  - Terrain spelling: accept both `gray` and `grey`.
  - Trim whitespace before `+TRACK` parsing.
  - Preserve `connect rNN` prefixes so `+TW*` segments are not dropped.
  - Preserve intra-row ordering around conversions vs favor/town tokens (some rows rely on “convert before towns” or “convert before/after favor” to avoid priest-cap divergence).

- Status: `bazel test //internal/replay:replay_test` is green, including the 21-game batch final-score check.

- UI gotcha (prod): Tailwind utilities may not apply in production (CSS shipped with unexpanded `@tailwind ...` directives), so layout-critical UI (e.g. the top Player Summary Bar) should use plain CSS/inline styles rather than Tailwind classes for `display:flex/grid`, sizing, and borders.

- Replay "Log" viewer (client):
  - `client/src/components/ReplayLog.tsx` renders `logStrings` as an HTML `<table>` by splitting each line on `|`. It does not display the literal pipes or fixed-width spacing from the concise text format.
  - Cells are fixed width (`6rem`) with `overflow:hidden` + `textOverflow:ellipsis`, so long tokens can be visually truncated even though the underlying `logStrings` line is intact.
  - `ReplayLog` row/cell click lookup now scans all `logLocations` (no early break on `lineIndex`) because server-side leech re-anchoring can produce non-monotonic `lineIndex` ordering by action index.

- Concise log generation:
  - `server/internal/notation/GenerateConciseLog` now keeps `L`/`DL` display tokens but reorders leech cells using `FromPlayerID` anchors so each leech’s previous non-leech token matches its true source action.
  - Regression test added: `server/internal/notation/generator_leech_placement_test.go` (distinct source leeches by same reacting player).

- Player Summary Bar (client):
  - Rendered via `client/src/components/GameBoard/PlayerSummaryBar.tsx` and placed as a `react-grid-layout` tile `i: "summary"` with default `h=2`.
  - To make player cards fill the tile height, the summary bar uses a single-row CSS grid with `gridTemplateRows: "1fr"` and game/replay wrappers avoid Tailwind `flex-1` sizing and use inline flex styles instead.
  - Power bowls use `PowerCircleIcon` (purple circle) instead of the string label `PW`.
