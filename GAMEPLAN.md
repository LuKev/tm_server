# Terra Mystica Replay Validator - Testing Gameplan

## Current Status

**Test File**: `test_data/4pLeague_S66_D1L1_G6.txt`
- Total lines: 413
- Current position: Entry 158 (38% through the file)
- Test file is a 4-player game: Engineers, Darklings, Cultists, Witches

## Current Bug Being Investigated

**Location**: Entry 59 (Round 1, turn 2)
**Faction**: Darklings
**Action**: `upgrade E5 to TE. +FAV11`

**Issue**: Cult track advancement from favor tile not being applied
- Expected: Earth cult at 2 (after +1 from FAV11)
- Actual: Earth cult at 1 (favor tile cult advancement not applied)
- **Symptom**: 127 validation errors accumulated, preventing progress to entry 158
- **Next steps**: Investigate SelectFavorTileAction to ensure cult advancement happens when taking a tile

**Debug Code**: Added temporary debug output in `server/internal/replay/validator.go` to track power bowls and resources.

## Bugs Fixed So Far

### Bug #4: BON1 Bonus Card Spade Action
- **Issue**: BON1 bonus card wasn't properly providing free spade for transformation
- **Fix**: Modified bonus card spade action handling
- **Test**: `TestBonusCardSpade_ProvidesFreeSpade` in `server/internal/game/special_actions_test.go`

### Bug #5: Bonus Cards Returned Too Early
- **Issue**: Bonus cards were being returned to the pool at round transitions instead of when selecting a new card
- **Fix**: Fixed bonus card lifecycle - cards now retained across rounds for income and only returned when selecting new one
- **Tests**:
  - `TestBonusCards_RetainedForNextRoundIncome`
  - `TestBonusCards_ReturnedWhenSelectingNew`
  - Location: `server/internal/game/bonus_cards_test.go`

### Bug #7: ACT6 Split Transform/Build Actions
- **Issue**: Power action ACT6 wasn't properly handling split transform/build (transform one hex, build on another)
- **Fix**: Ensured both hexes use free spades from power action
- **Test**: `TestPowerActionSpade2_SplitTransformAndBuild` in `server/internal/game/power_actions_test.go`
- **Example**: "ACT6. transform F2 to gray. build D4"

### Bug #8: Compound "Convert...Pass" Action Parsing
- **Issue**: Actions like "convert 1PW to 1C. pass BON7" were only parsing the "convert" part, ignoring the "pass" part
- **Impact**: Players didn't pass when they should have, so bonus cards weren't returned/taken, breaking the bonus card pool
- **Fix**: Updated `server/internal/replay/parser.go` to detect compound convert+pass actions and parse the pass action
- **Location**: Line 139 in test file: `convert 1PW to 1C. pass BON7`
- **Result**: Successfully progressed from entry 140 to entry 158

### Bug #9: Power Not Spent in Split Transform/Build Actions
- **Issue**: When using power actions like ACT6 (2 free spades) with split transform/build (e.g., "burn 6. action ACT6. transform F2 to gray. build D4"), the power was burned (moved from bowl 2 to bowl 3) but never spent (moved from bowl 3 to bowl 1)
- **Impact**: Power bowls remained in wrong state (0/0/6 instead of 6/0/0), causing accumulated drift in game state
- **Fix**: Added code in `server/internal/replay/action_converter.go` to spend power from bowl 3 after burning and before marking the action as used
- **Location**: Entry 48 in test file, Lines 319-323 in action_converter.go
- **Result**: Reduced validation errors from 129 to 127

### Other Fixes
- Terrain color parsing improvements
- Income calculation fixes
- Power action handling
- Leech mechanics
- Bonus card management

## Modified Files (Uncommitted)

```
M server/internal/game/actions.go
M server/internal/game/bonus_cards.go
M server/internal/game/power_actions.go
M server/internal/game/state.go
M server/internal/replay/action_converter.go
M server/internal/replay/parser.go
M server/internal/replay/validator.go
```

## Next Steps

1. **Investigate Current Bug**:
   - Run the replay validator to see the debug output for entries 133 and 137
   - Identify what resource mismatch is occurring with Darklings
   - Determine if it's related to power actions (ACT2), priest usage, or cult track advancement

2. **Fix the Bug**:
   - Update game logic in relevant files (`actions.go`, `power_actions.go`, etc.)
   - Add regression test to prevent reintroduction

3. **Continue Validation**:
   - Resume from line 137 and continue through the rest of the 413-line test file
   - Fix any additional bugs discovered
   - Document each bug with clear regression tests

4. **Completion**:
   - Successfully validate entire test file end-to-end
   - Ensure all tests pass: `cd server && bazel test //...`
   - Commit all fixes with comprehensive test coverage

## Running the Replay Validator

```bash
# Run all tests
cd server && bazel test //...

# Run replay validator tests specifically
cd server && bazel test //internal/replay:replay_test

# The validator processes the test file and compares:
# - Expected resources vs actual (VP, coins, workers, priests, power bowls)
# - Expected cult track positions vs actual
# - Action validity and state transitions
```

## Test File Context

- **Game**: 4-player League game (Season 66, Division 1, League 1, Game 6)
- **Players**: GeorgeShortwell (Engineers), Shadow (Darklings), forrestblue (Cultists), 295381644 (Witches)
- **Options**: Various expansions and rule variants enabled
- **Rounds**: 6 rounds total (standard Terra Mystica game)
- **Current Progress**: In Round 2, debugging turns 4-6

## Key Replay Validator Features

- Parses Terra Mystica game logs (snellman.net format)
- Converts log actions to internal game actions
- Validates resource changes match expected values
- Handles leech mechanics (asynchronous power gains)
- Tracks bonus cards, cult positions, power bowls
- Supports power actions, special actions, faction abilities

## Notes

- Each bug found through replay validation gets a regression test
- Tests are in `server/internal/game/*_test.go` files
- Debug output can be temporarily added to `validator.go` for investigation
- Remove debug code before committing fixes
