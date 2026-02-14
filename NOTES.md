# Workspace Notes (Snellman Replay Certification)

- Repo rule: use Bazel (`/Users/kevin/projects/tm_server/server` is the Bazel workspace). Avoid `go test`.

- UI (client): `PlayerState.digging` is the digging *upgrade level* (0..2; Fakirs max 1). The UI displays digging as **workers per spade** (`3 - diggingLevel`, clamped 1..3). Temp shipping bonus is detected via `gameState.bonusCards.playerCards[playerId] === BonusCardType.Shipping` and rendered as `X (+1)`. Shipping is hidden for Fakirs/Dwarves; digging is hidden for Darklings. A top-of-screen summary bar mirrors player-board header info (VP, order, resources, shipping/digging) and highlights the current-turn player.

- Replay ingestion (format-driven): `server/internal/replay/manager.go` routes by content format (`auto|concise|snellman|bga`) via `parseReplayLogContent`. Snellman imports are converted to canonical concise logs before replay execution.

- Certification corpus (S67-S69, G1-G7):
  - Fixtures: `server/internal/replay/testdata/snellman_batch/` (raw Snellman ledger text per game).
  - Oracle: `server/internal/replay/testdata/snellman_batch/manifest.json` (expected final VP totals from Snellman).
  - E2E test: `server/internal/replay/snellman_batch_replay_test.go` runs `ImportText(...,"snellman") -> StartReplay -> JumpTo(end)` and asserts `state.FinalScoring[*].TotalVP` matches manifest.

- Cleanup timing (critical): Snellman logs can contain late reactions after the final `PASS` of a round (leeches, Cultists `+TRACK`, etc.). Do not run `ExecuteCleanupPhase()` immediately when `AllPlayersPassed()` becomes true; trigger it at the round boundary (when processing the next `RoundStartItem`), and at end-of-log for round 6 final scoring. See `server/internal/replay/simulator.go`.

- Snellman ledger rows are authoritative for per-step player resources:
  - Each action row includes the player’s *post-action* resources and cult positions.
  - The trailing numeric markers like `3 1` on an action row indicate later leech/decline rows that correspond to that action (example: `cultists ... upgrade F3 to TP ... 3 1` links to later `Decline 3 from cultists` and `Decline 1 from cultists` rows).
  - Use these markers to bind each `L`/`DL` to the correct source action (instead of guessing from pending offers).

- Snellman-to-concise conversion gotchas (`server/internal/notation/snellman_to_concise.go`):
  - Terrain spelling: treat both `gray` and `grey` as `Gy`.
  - `+TRACK` segments can appear with leading whitespace after split (trim parts before `+` checks).
  - Cultists faction ability can show as a leading `+EARTH/+AIR/...` segment on a later unrelated action line; that cult step must backtrack to the prior triggering action, not remain chained to the new action token.
  - Income blocks: only skip rows whose extracted action is `cult_income_for_faction` or `other_income_for_faction` (some logs include real actions between “Round X income” headers).
  - `DIGn-<coord>` tokens: Snellman can interleave conversions between “dig” steps (notably Alchemists spade-trigger power gains). The converter emits `DIGn-<coord>` to preserve intra-row ordering. Execution is implemented by `notation.LogDigTransformAction` and MUST only apply the terraforming step (costs, terrain change, VP/faction spade bonuses) without building; the subsequent build action handles any remaining transform/build.

## Open Divergences (Ledger Resource Mismatches)

- [x] `S68_G2` (fixture: `server/internal/replay/testdata/snellman_batch/4pLeague_S68_D1L1_G2.txt`)
  - First mismatch: Snellman line 306, player `Cultists`
  - Action: `upgrade E6 to SA. +FAV5. convert 2P to 2W. +2TW3. +TW7`
  - Want vs got: priests `P=2`, replay has `P=0` (VP/C/W/PW/cults match at this checkpoint).
  - Note: `+2TW3` means claiming the TW3 town tile twice in the same action (two “priest towns”), plus `+TW7` (shipping town). In concise, `TW3` maps to `TW9VP` (TownTile9Points => +1 priest) and should appear twice.
  - Root cause: `normalizeUpgradeFavorTownOrder` was reordering the compound token list to place town-tile selections ahead of conversions, so the replay tried to gain priests from the two TW3 towns before spending `2P` in the conversion, incorrectly hitting the priest cap.
  - Fix: `normalizeUpgradeFavorTownOrder` now only moves `FAV-*` tokens directly after the triggering upgrade while preserving the original relative ordering of everything else (especially conversions vs `TW*` tokens).
  - Related fix: Snellman action extraction preserves leading `connect rNN` segments so a prefix like `connect r33. +TW4. ...` doesn’t drop the `+TW4` town token.

- [x] `S68_G4` (fixture: `server/internal/replay/testdata/snellman_batch/4pLeague_S68_D1L1_G4.txt`)
  - First mismatch: Snellman line 357, player `Cultists`
  - Action: `build G7`
  - Want vs got: power bowls want `PW=2/4/1`, replay has `PW=1/5/1` (1 power is in bowl1 vs bowl2).
  - Root cause: Cultists “all opponents declined” bonus was being applied even when the leeching player had 0 capacity (no tokens in bowl 1/2). Snellman still logs a `Decline N from cultists` row in that case, but does NOT award the Cultists bonus.
  - Fix: Cultists leech tracking now counts an accept/decline only if the leeching player could actually gain `>0` power at response time (simulated via a clone of their current power bowls). Otherwise it’s treated as a forced no-op for Cultists bonus. Implemented in `server/internal/game/action_power_leech.go` via `potentialGain` + `ResolvedCount`.

- [x] `S68_G7` (fixture: `server/internal/replay/testdata/snellman_batch/4pLeague_S68_D1L1_G7.txt`)
  - First mismatch: Snellman line 335, player `Mermaids`
  - Action: `upgrade C1 to TE. +FAV5. connect r1. +TW2. connect r10. convert 1PW to 1C. +TW4. convert 1PW to 1C`
  - Want vs got: power bowls want `PW=1/0/5`, replay has `PW=2/0/4` (1 power is in bowl1 vs bowl3).
  - Root cause: intra-row compound action token ordering bug. Snellman’s order is:
    - `... +TW2. convert 1PW->1C. +TW4. convert 1PW->1C`
    - If `+TW4` (TownTile6Points, +8 power) is applied before the first conversion, the two `convert 1PW->1C` operations shift the final bowls by exactly `+1` in bowl1 and `-1` in bowl3, producing `2/0/4` instead of `1/0/5`.
  - Fix: `normalizeUpgradeFavorTownOrder` now moves only `FAV-*` tokens directly after the triggering upgrade and preserves the original relative order of towns vs conversions. Also keep `connect rNN` prefixes so `+TW*` segments are not dropped. Regression test: `TestConvertSnellmanToConcise_MermaidsConnectMultipleTownsPreservesOrder`.

- [x] `S69_G3` (fixture: `server/internal/replay/testdata/snellman_batch/4pLeague_S69_D1L1_G3.txt`)
  - Symptom: replay failed while executing a compound `...UP-SA-E6.TW5VP.FAV-A1...` with `no pending town formation for player Cultists`.
  - Root cause: for Temple/Sanctuary upgrades, the engine creates `PendingFavorTileSelection` first and only checks/creates `PendingTownFormation` after the favor tile is selected (`SelectFavorTileAction` calls `CheckAllTownFormations`). If the concise token list selects `TW*` before `FAV-*`, the town selection is attempted before the pending town exists.
  - Fix: `normalizeUpgradeFavorTownOrder` correctly detects `TW*VP` tokens and reorders `UP-(TE|SA)-*` compound rows to execute `FAV-*` immediately after the upgrade and before `TW*VP` tokens.

## Snellman Parsing Gotcha: `connect rNN`

- Snellman can prefix a compound action with a Mermaids “connect rNN” marker and then include real sub-actions like `+TWx` and conversions.
- If `extractSnellmanAction` truncates the string at `convert/build/upgrade`, the `+TWx` is lost and replay state will diverge (typically by missing the town tile power/priests before later conversions).

## Open Divergences (Games That Fail To Replay To End)

- [ ] `S69_G5` (`4pLeague_S69_D1L1_G5.txt`): `LogDigTransformAction` -> `TransformAndBuildAction` fails with `not enough workers: need 2, have 0` (JumpTo fails mid-game).

## Open Divergences (Final Score Mismatches)

- [ ] `S67_G6`: Cultists final total VP mismatch (got 150, want 151).
- [ ] `S69_G6`: Cultists final total VP mismatch (got 121, want 122).
