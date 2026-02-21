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
  - Replay conversion now normalizes compound tokens so `L`/`DL` are always standalone actions (never chained with other actions), and leading Cultists `+TRACK` bumps are backtracked/chained to the prior triggering Cultists main action.
  - Replay-linear conversion (`ConvertSnellmanToConciseForReplay`) currently backtracks/chains leading Cultists `+TRACK` bumps to prior triggering main actions (same as display conversion).
  - For `action BON1` rows with conversion(s) before `build` (e.g. `action BON1. convert ... . build E8`), do not emit `ACTS-<coord>.<coord>`. Emit `ACT-BON-SPD` and keep conversions/build explicit (`ACT-BON-SPD.C... .E8`) so replay order remains BON1 -> conversions -> build and avoids non-idempotent special-action retries (`special action 8 already used this round`).
  - Batch validation on Snellman fixtures `S64-S69` (`server/internal/replay/testdata/snellman_batch*`) currently reports:
    - no chained leech tokens (`.L`, `.DL`, `L.`, `DL.`)
    - no standalone Cultists bump tokens (`+E/+W/+F/+A`) in Cultists column

- Status (current):
  - `bazel --batch test //internal/replay:replay_test --test_output=errors` is failing at `S67_G5` (`cannot afford upgrade to Temple`) with current backtracked Cultists bump behavior.
  - Ledger tests were updated to treat Cultists Snellman row-by-row state as non-authoritative for alignment/state checks (Cultists +TRACK timing can be backfilled in Snellman but applied immediately in replay).
  - In `S67_G5`, first observable Cultists resource drift appears around Snellman lines `81-84`:
    - line 81 (`action ACT5. build G4`) Snellman has `PW=4/5/3`, cult `1/0/2/2`
    - line 84 (`+AIR`) Snellman then moves to `PW=3/6/3`, cult `1/0/2/3`
    - current replay concise backtracks this as `ACT5.G4.+A`, applying the bowl/cult change at line 81 timing.
  - Separate root cause for the later `S67_G5` Temple affordability failure:
    - Snellman line `240` (`+EARTH` after `pass BON2` leech acceptance at lines `238-239`) is not present as a chained bump on the replay concise `PASS-BON-4C` token.
    - Missing this Cultists bump leaves Cultists at Earth `4` instead of `5` at Round 5 income start, which under-awards cult income by `1C` (line `245`), propagating to `3C` (not `4C`) before line `282`.
    - At line `282`, replay executes `C1PW:1C` to reach `4C`, then fails `UP-TE-F3` (`cannot afford upgrade to Temple`); expected was `5C` after conversion.
  - Converter fix for dropped Cultists bumps:
    - `appendCultBonus` now tries secondary target rows when the preferred row already contains the same `+TRACK` token, instead of returning early and dropping the bump.
    - If all candidate rows already include that bump, it appends to the first valid candidate (preserves multiple same-track bumps rather than dedup-dropping them).
    - This unblocks `S67_G5` replay and preserves replay notation tests (`//internal/notation:notation_test`) and replay suite (`//internal/replay:replay_test`).
  - Full server test verification (unfiltered):
    - `bazel --batch test //... --test_output=errors` passed on 2026-02-20.
  - Regression check after BON1 conversion-order fix:
    - `TestSnellmanBatchReplayS64to66_FinalScoresMatch` passes (no S64-S66 final-score regression).
    - S64-S66 ledger subset (`S66_G1`, `S66_G2`, `S64_G4`, `S64_G5`, `S64_G6`, `S65_G4`) has no `special action 8` errors and now passes under the Cultists-row-timing tolerance.
  - S65 Darklings final VP mismatches are fixed:
    - `4pLeague_S65_D1L1_G2`
    - `4pLeague_S65_D1L1_G3`
    - `4pLeague_S65_D1L1_G4`
  - Fix details: for Snellman `action BON1. dig N. build <coord>`, converter now emits `ACT-BON-SPD.DIGN-<coord>.<coord>` and replay supports `ACT-BON-SPD` as a pending free spade marker, so explicit paid digs (including Darklings priest/+2VP semantics) are preserved.

- UI gotcha (prod): Tailwind utilities may not apply in production (CSS shipped with unexpanded `@tailwind ...` directives), so layout-critical UI (e.g. the top Player Summary Bar) should use plain CSS/inline styles rather than Tailwind classes for `display:flex/grid`, sizing, and borders.

- Replay "Log" viewer (client):
  - `client/src/components/ReplayLog.tsx` renders `logStrings` as an HTML `<table>` by splitting each line on `|`. It does not display the literal pipes or fixed-width spacing from the concise text format.
  - Cells are fixed width (`6rem`) with `overflow:hidden` + `textOverflow:ellipsis`, so long tokens can be visually truncated even though the underlying `logStrings` line is intact.
  - `ReplayLog` row/cell click lookup now scans all `logLocations` (no early break on `lineIndex`) because server-side leech re-anchoring can produce non-monotonic `lineIndex` ordering by action index.

- Concise log generation:
  - `server/internal/notation/GenerateConciseLog` now keeps `L`/`DL` display tokens but reorders leech cells using `FromPlayerID` anchors so each leech’s previous non-leech token matches its true source action.
  - Regression test added: `server/internal/notation/generator_leech_placement_test.go` (distinct source leeches by same reacting player).
  - Replay import has an opt-in execution reorder for Snellman logs: `ReplayManager.SetSourceAnchoredLeechOrdering(true)` moves source-tagged `L`/`DL` actions directly after their triggering source action during parsing (`server/internal/replay/manager.go`). This is enabled in `server/cmd/server/main.go` for viewer sequencing, but defaults to `false` for strict ledger-parity/test flows.

- Player Summary Bar (client):
  - Rendered via `client/src/components/GameBoard/PlayerSummaryBar.tsx` and placed as a `react-grid-layout` tile `i: "summary"` with default `h=2`.
  - To make player cards fill the tile height, the summary bar uses a single-row CSS grid with `gridTemplateRows: "1fr"` and game/replay wrappers avoid Tailwind `flex-1` sizing and use inline flex styles instead.
  - Power bowls use `PowerCircleIcon` (purple circle) instead of the string label `PW`.

- 2026-02-20 replay update (S60-S63 batch):
  - Added fixture batch `server/internal/replay/testdata/snellman_batch_s60_63/` (28 games) and batch replay test `server/internal/replay/snellman_batch_s60_63_replay_test.go`.
  - Added ACTG conversion fallback for `action ACTG. build <coord>` in `server/internal/notation/snellman_to_concise.go` and regression coverage in `server/internal/notation/snellman_to_concise_test.go`.
  - Fixed Giants terraform charging bug in `server/internal/game/actions.go`: when remaining required spades are `0`, replay must not charge terraform costs (previously charged due Giants-special `GetTerraformCost` behavior).
  - Fixed Dwarves/Fakirs skip-cost dedup scope in `server/internal/game/actions.go`: de-dup skip payment only while `SuppressTurnAdvance` is true (compound/synthetic execution), so standalone later turns pay tunneling/carpet costs again.
  - Fixed `S63_G2` final-VP drift by preserving Snellman negative cult-step tokens in compounds (e.g. `-water. +TW5`):
    - converter now emits `-W` (`server/internal/notation/snellman_to_concise.go`)
    - parser/executor now supports `-F/-W/-E/-A` via `LogCultTrackDecreaseAction` (`server/internal/notation/parser.go`, `server/internal/notation/types.go`)
    - this keeps round-5 Cultists bowl state aligned (`0/2/4` before the line-305 leech) and removes the need for S60-S63 score overrides.
  - Added explicit regressions for cult-town selector semantics (`-<cult>`):
    - parser accepts multi-selector compounds like `-F.-W.-E.TW8VP` (`server/internal/notation/parser_test.go`)
    - conversion preserves up to three selectors for non-Cultists factions (`server/internal/notation/snellman_to_concise_test.go`)
    - replay execution coverage:
      - key-limited near-top scenario where three selectors intentionally leave only one track topping on `TW8VP`
        - test starts at `Keys=0` (town grants to `1`), so there is exactly one key available during cult advancement
      - selector+towntile rows fail if no pending town formation exists
      (`server/internal/notation/types_test.go`)
  - Verification after these changes:
    - `bazel --batch test //internal/replay:replay_test --test_output=errors` passed.
    - `bazel --batch test //internal/game:game_test --test_output=errors` passed.
