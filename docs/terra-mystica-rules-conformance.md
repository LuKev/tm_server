# Rules conformance evidence ledger

Reviewed 2026-09-20. This is an evidence ledger, not a certification that the
engine implements every legal Terra Mystica game. `Unit` means a local assertion;
`external replay` means a recorded game interpreted through the replay adapter.
Neither label means independent strict per-decision conformance.

## Sources and interpretation

- [Publisher base rulebook](https://www.feuerland-spiele.de/fileadmin/game/Terra_Mystica/Terra_Mystica_rules_EN_Web.pdf): setup pp.3–5; income p.8; actions pp.9–14; towns p.14; cleanup p.15; final scoring p.16; faction appendix.
- [Publisher mini expansion catalog](https://shop.asmodee.ca/en/csgzm7249): new scoring/bonus/town components are an expansion, not original base components.
- [Snellman implementation usage](https://terra.snellman.net/usage/): external log notation and platform conventions, not a replacement for the selected competition profile.
- [Rules FAQ, faction sections 3.1 and 3.6](https://boardgamegeek.com/wiki/page/Terra_Mystica_FAQ): one tunneling/flight destination per action (including the two-spade power action); Giants may only transform to home terrain.

The base scoring initializer now explicitly excludes the Temple/Priest promotion.
It draws rounds 6 through 1, keeping Spades out of the first two draws but in the
remaining pool, as specified by the setup erratum. A scripted physical-draw
trace tests ordering and re-entry without a probabilistic acceptance threshold.
The general web-game initializer retains its existing expanded pool. Winner
selection no longer invents a leftover-resource tiebreak; equal VP is a shared
victory, and player-ID ordering is only stable presentation.

## Rule-family matrix

Paths below are relative to `server/internal/game/` unless prefixed otherwise.
The source column refers to the rulebook sections above. A named test file is
existing coverage to inspect, not a claim every case in that rule family passes
an independent audit. Source-derived golden expectations added in this work are
identified explicitly. Remaining questions are acceptance work, not house rules.

| Rule family | Source | Implementation / fixtures | Independent evidence and remaining gap |
|---|---|---|---|
| Base components and setup | setup | `scoring_tiles.go`, `scoring_tiles_test.go`, `setup_flow_test.go`; `az/setup.go` | New base pool golden check: 256 seeds, 8 possible types, 6 distinct rounds, no late spade scoring. Full setup state still needs external strict comparison. |
| Initial placements / factions | setup, faction appendix | `setup_flow_test.go`, `action_select_faction_test.go`; `az/setup_test.go` | Unit coverage; verify Nomads/Chaos placement timing with canonical external records. |
| Income / priest supply | income | `income.go`, `income_test.go`, `temple_income_test.go` | External ledger includes income effects; all resource-source/order permutations not independently checked. |
| Turn order / pass / bonus selection | actions, cleanup | `turn_order_test.go`, `bonus_card_pass_vp_test.go`, `bonus_card_setup_coins_test.go` | Unit + replay; selected 1v1 profile must distinguish fixed clockwise from variable pass order outside 2p. |
| Terrain and ordinary building | transform action, FAQ 2.1 | `action_transform_build_test.go`, `board/terraform_test.go` | Route-aware golden cases cover up to six steps, no crossing home, canonical minimum, paid long-route cost/scoring, and free-spade top-up restrictions. Complete candidate domain still needs independent enumeration. |
| Reachability / shipping / skipping | transform, shipping, factions | `action_transform_build_test.go`, `action_shipping_digging_vp_test.go`, `faction_integration_test.go` | Unit + replay; end-to-end Fakirs external record missing. |
| Bridges | power actions, Engineers | `board/map_bridge_test.go`, `action_engineers_bridge_test.go` | Unit + replay; crossing/land edge cases need independent positions. |
| Upgrades / favor choices | upgrade action, favor appendix | `action_upgrade_building_test.go`, `action_select_favor_tile_test.go` | Unit + replay; reactions/favor/town simultaneous timing not externally certified. |
| Power cycling / burning / conversion | power rules | `power_test.go`, `action_conversion_test.go` | Unit; final conversion now exhausts ordinary/Alchemist resource boundaries with independent arithmetic. |
| Shared power actions / spades | power actions | `power_actions_test.go`; `az/contract_test.go` | Known intermediate-destination repair handled in R1; oracle agreement alone cannot prove no omitted choices. |
| Leech / reactions | power from structures | `action_power_leech.go`; `az/contract_test.go` | R1 exploit regression and boundary tests; external replay currently accommodates historical order and must not certify strict decision ownership. |
| Cult steps / keys / priests | cult action, towns | `cult_test.go`, `action_select_town_cult_top_test.go` | Unit; new 120-position 1v1 cult scoring golden domain checks both scoring implementations. Pending town key borrowing still needs independently adjudicated timing examples. |
| Town formation / merging / choices | towns | `town_test.go`, `action_select_town_cult_top_test.go` | Unit + replay; simultaneous town eligibility/selection histories need external canonical tests. |
| Round scoring / cult rewards / termination | scoring appendix, cleanup | `scoring_tiles_test.go`, `action_scoring_test.go`, `cleanup_test.go` | Unit + external final-score corpus; strict transition ownership and resource accounting not universally checked. |
| Area connectivity / final ties | final scoring | `final_scoring_test.go` | Unit + external final totals. Removed unsupported resource tiebreak; equal VP now shared victory. Independent connected-network fixtures remain incomplete. |
| Clone isolation / failure atomicity | engine integrity | `state_clone.go`, `manager_state_action_test.go`; `az/contract_test.go` | Implementation properties, not external rules proof. Every mutable faction/reaction field and rejected parameter still needs systematic fault-injection coverage. |
| Complete legal actions / replay determinism | all sections | `az/enumerate.go`, `az/oracle_test.go`, `az/contract_test.go` | Oracle shares game execution; retain as implementation-agreement evidence only. |

## Added reaction and simultaneous-reward fixtures

These are **source-derived local golden positions**, not externally adjudicated
complete games. The designer-referenced [FAQ sections 2.4, 2.10 and 3.4](https://boardgamegeek.com/wiki/page/Terra_Mystica_FAQ)
support response order, leech-before-builder-rewards, freely ordered simultaneous
town/favor/ordination rewards, and Mermaid founding on the owner's turn.
Rulebook pp.14/18 describe town VP, keys, and capped priest rewards separately.

- `game/resources_test.go`: `TestLeechRulebookAcceptanceBoundaries`,
  `TestLeechRejectsFragmentWithoutMutation`,
  `TestLeechCappedAcceptanceResolvesCultistsOnce`,
  `TestLeechDistinctOffersRecomputeCapacityAndAffordability`,
  `TestLeechNoCapacityDoesNotRewardCultists`, and
  `TestLeechResourceAndVPBoundsExhaustive` cover the no-fragment rule,
  affordability/capacity, source-event identity, and 6,552 arithmetic cases.
- `game/manager_state_action_test.go`:
  `TestLeechPrecedesBuilderRewardsIncludingPassedResponder` and
  `TestLeechRespondersFollowCurrentTurnOrder` check strict timing, ownership,
  and agreement with public pending-decision serialization.
- The same file's `TestSimultaneousDarklingsTownAndOrdinationEitherOrder`
  proves that town workers can fund ordination and that ordination-first remains
  legal. `TestChaosFirstFireFavorCreatesImmediateTownKey` checks the first of two
  Chaos favors, immediately borrowable town-key credit, continued favor choice,
  and priest-town VP/key retention when no priest piece remains available.
- `az/conformance_test.go`: `TestLeechPrecedesRewardsInPublicActionSet` and
  `TestSimultaneousTownRewardChoicesInPublicActionSet` verify these alternatives
  through the public AI legal-action interface. They do not use the shared-rule
  enumeration oracle as their source of expected timing.
- `game/special_actions_test.go`: `TestBaseTransformSpecialRejectsInvalidTargetsAtomically`
  rejects river/permanent-terrain transformations and unpaid optional dwellings;
  `TestCultSpecialAllowsCappedAdvancement` and
  `TestAurenCultSpecialAllowsPartialAdvancement` implement the user's explicit
  adjudication allowing cult-bump special actions with partial or zero movement.
  This supersedes the earlier FAQ-based actual-effect restriction and is not
  independent verification of official wording. Public AZ tests
  `TestBaseSpecialActionsCannotTerraformRiver` and
  `TestCultSpecialNoEffectAllowedInPublicActions` check the corresponding
  prohibited terrain and permitted cult choices at the legal-action interface.
  These are local positions, not external complete-game evidence.
- `az/contract_test.go`:
  `TestMandatoryTownDoesNotReopenMainActionWithDelayedTownRemaining` checks that
  resolving a mandatory town reward ends the main action even when an unrelated
  optional Mermaid town remains claimable. This is a local public-interface
  timing regression grounded in FAQ 2.10/3.4, not an external game trace.
- `TestAllBaseFactionsNaturalGameInvariants` in `az/conformance_test.go` covers fourteen seeded
  complete games (seven faction pairs, two seeds) with no resource grants,
  all six rounds, per-action replay hashes, clone isolation, failed-action
  atomicity, resource accounting bounds, home-terrain buildings, physical piece
  limits, and terminal-score agreement. Failures retain their action trace. This is
  **internal invariant coverage**, not independent validation of all rules or
  external full-game coverage; an external corpus is supplemental, not a gate.

Additional spade boundary evidence:

- `game/power_actions_test.go` covers pre-action reachability, one tunnel/flight,
  intermediate destinations, optional reward refusal, and separate free-spade
  round scoring versus paid-priest Darklings scoring. Bonus-card spades reuse
  the same transformation/cost validation without charging shared-action power;
  `TestBonusSpadeUsesCorrectFactionCostsWithoutPowerCharge` checks Giants and
  Darklings payments independently.
- `game/cleanup_test.go`: `TestCultSpadesGiantsRequirePairAndHomeTerrain` and
  `TestCultSpadeIgnoresBonusCardShipping` enforce the distinct cult-reward rules.
  `az/oracle_test.go:TestGiantsCultSpadesPublicActionSet` checks the public action
  set and both consumed spades against explicit expected outcomes.
- `az/conformance_test.go:TestHalflingsStrongholdPublicRules` covers three-space
  allocation, required home completion before splitting, rejection of paid
  top-ups on subsequent hexes,
  optional forfeiture, one dwelling, piece limits, and invalid-action atomicity.
  Each application names a final destination for one hex; revisiting it is not a
  separate legal route. This avoids extra state to track partial directions.

The two websocket golden fixtures retain their original final score expectations.
Their importer matches recorded offer amounts but lets strict acceptance compute
the actual capped gain (S61 line195 offers three and gains two). It reorders
recorded asynchronous responses before builder rewards using lookahead; that
normalization is recorded import behavior, not an original strict decision trace.

## All 14 base factions

Every faction has a corresponding `server/internal/game/factions/<name>_test.go`
and cross-faction integration tests. The table records seats in the 70 stored
Snellman games, not successful strict replays. All are 4-player league records;
they supply interaction examples but do not independently cover the selected
1v1 competition profile. Faction rules come from the publisher faction appendix.

| Faction | Core ability requiring conformance | External seats | Status / highest-value missing evidence |
|---|---|---:|---|
| Alchemists | VP/coin conversion, stronghold/spade power | 4 | Unit + replay; exact conversion affordability and action timing |
| Auren | stronghold favor and cult special action | 2 | Unit + replay; occupied top/key boundary |
| Chaos Magicians | one late initial dwelling, double favor, two actions | 6 | Unit + replay; pending reactions/towns between constituent actions |
| Cultists | response-triggered cult advancement, stronghold VP | 57 | Unit + replay; ledger explicitly excludes Cultists rows, so strict ledger evidence absent |
| Darklings | priest terraform and stronghold conversion | 64 | Unit + replay; priest cap and stronghold conversion timing |
| Dwarves | tunneling, no shipping | 10 | Unit + replay; legal destination/action combinations and final network |
| Engineers | low costs/income, worker bridges, bridge pass scoring | 37 | Unit + replay; bridge geometry and ownership boundaries |
| Fakirs | carpet flight, range, no shipping | 0 | Unit only; complete external game absent from corpus |
| Giants | two-spade transforms, stronghold special action | 1 | New ordinary-action golden rejects all six nonhome destinations without state mutation; home transformation costs two spades. One action spade may be topped up with exactly one paid spade; the lone cult-spade restriction is separate. |
| Halflings | digging costs, spade VP, stronghold three spades | 8 | Strict public-action boundaries cover original-building reachability, split only after home, one final destination per hex, no paid top-up on subsequent hexes, spade VP, optional discard/build forfeiture, home-only dwelling and eight-piece limit. Fixtures are constructed positions, not complete externally verified games. |
| Mermaids | shipping, river town connection | 7 | Unit + replay; delayed/overlapping town identity |
| Nomads | third initial dwelling, sandstorm | 19 | Unit + replay; shipping/bridge adjacency restrictions on sandstorm |
| Swarmlings | town workers, stronghold trading house | 16 | Unit + replay; free upgrade reactions and simultaneous towns |
| Witches | town VP, stronghold dwelling | 49 | Unit + replay; free dwelling reachability exception and reactions |

Corpus counts are derived from `expected_total_vp` keys in the three manifests:
`replay/testdata/snellman_batch` (21 games), `snellman_batch_s60_63` (28),
`snellman_batch_s64_66` (21). The manifests preserve external final scores and
game IDs. They are not newly adjudicated by this audit.

## Limits of existing replay evidence

`replay/manager.go:createInitialState` enables `ReplayMode["__replay__"]`.
`replay/simulator.go:StepForward` executes notation-specific actions directly,
not the AI adapter's complete strict action/decision-owner path. Several log-only
actions have permissive validation. Expansion-specific replay funding also exists
in `notation/types.go` (Firewalkers); it should not be misreported as proof that
base-faction logs necessarily use that particular accommodation.

`replay/snellman_ledger_resources_test.go` skips Cultists ledger rows, allows
out-of-order leech alignment, excludes informational income/scoring rows, and
skips dropout fixtures. These choices can be reasonable for replay viewing but
are not strict per-decision validation. Clearing `ReplayMode` alone does not
remove log-specific execution or skipped assertions.

## Golden-test fault injection

On 2026-09-20, three single-line production mutations were applied one at a time,
tested through Bazel, and restored immediately. Each targeted test failed for
the intended reason:

| Injected fault | Detecting test | Observed counterexample |
|---|---|---|
| Normal final conversion divisor 3 changed to 4 | `TestRulebookFinalResourceBoundaries` | Six total coins produced 1 VP; independent expected value was 2. |
| Cult first-place award 8 changed to 9 | `TestRulebookTwoPlayerCultRanks` | Positions 0/1 awarded player B 9 VP; expected 8. |
| Base scoring pool excludes Town instead of promotional Temple/Priest | `TestBaseGameScoringPool` | Seed 0 selected forbidden promotional tile in round 3. |

The restored resource, cult, setup-pool, and Giants tests passed together under
`bazel test //internal/game:game_test` with their test filter. These experiments
demonstrate sensitivity to those three faults; they do not constitute exhaustive
mutation coverage (decision-owner and all action-omission faults remain to cover).

## Initial R0–R2 local verification — 2026-09-20

From `server/`:

```sh
bazel test //internal/... //cmd/... //:az_loop_smoke_test //:az_scaling_curves_smoke_test --test_output=errors --test_timeout=900
```

All 13 test targets passed, including the exhaustive AZ action/oracle suite
(192.3s), Python encoding/checkpoint/learner tests (29.9s), game, board, factions,
lobby, notation, replay, websocket, both command test targets, and both loop
smokes. The final aggregate check reused successful Bazel results: twelve targets
had just run uncached together; the game suite was rerun uncached (2.7s) after
correcting a stale test that set only one of the two cult-position mirrors.
No production change was needed for that fixture failure.

Correctness and simplicity adversarial reviews found additional defects during
implementation; their resolved cases are listed above. Review also removed an
unnecessary pending-spade map and its two encoding features. `git diff --check`
passes. Frontend files were not changed or built; old power-spade requests are
explicitly incompatible and client migration is required before deployment.
These are local correctness/integration checks, not model-strength experiments.

## Acceptance boundary after user clarification

Implemented follow-up regressions cover all valid current-board Mermaid river
anchors, stale/overlapping opportunity invalidation, own-turn claim timing,
pre-main reward availability, either simultaneous town reward order, and cache
normalization without mutating imported state. Halflings tests reject paid
top-ups after splitting and allow the tentative town/spade ordering. Additional
tests cover intermediate cult terrain and tentative reversal, zero-effect cult
specials and priest placements, Chaos reaction/conversion boundaries, chained
Fakir/Dwarf final networks, and Alchemist same-action power restrictions.

Adversarial review also caught two interacting town defects: growing a town
before selecting its reward could create a duplicate reservation, and the
all-cult town reward ignored keys from other simultaneous towns. Both have
targeted regression tests and fixes preserving the existing state model.

The user clarified that stored games primarily test support for recorded moves,
not equality with the entire legal rules/action set. A large external 1v1/Fakirs
corpus is not a required gate. Supplement the existing replay coverage when useful,
using small, gentle requests only; no Snellman requests were made for the follow-up.

The bounded follow-up instead tests the user-adjudicated interactions through
explicit positive/negative fixtures, complete legal-choice checks, and adversarial
review. Do not call self-replay an independent rules oracle. Two working
interpretations remain tentative: cult-spade reversal and Halflings simultaneous
town/spade reward order. Those choices are implemented and labeled tentative in
the tests and competition contract, not silently treated as verified official rules.

External differential play remains valuable future evidence, not a dependency
that prevents addressing known rules defects. Representation, search, recovery,
and evaluation milestone status is maintained in the corrective plan; these rules checks do not authorize
restarting expensive training or establish model strength.

## Clarification follow-up verification — 2026-09-20

All 13 test targets passed uncached with:

```sh
bazel test //internal/... //cmd/... //:az_loop_smoke_test //:az_scaling_curves_smoke_test --test_output=errors --nocache_test_results --test_timeout=900
```

This includes game, board, factions, replay, websocket, Python learner/encoding,
both training/evaluation smoke targets, and the exhaustive AZ suite (189.5s).
A final nil-cache import guard and its regression were then verified with
`bazel test //internal/az:az_test --test_filter=TestMermaidCurrentBoardTownChoices --test_output=errors --nocache_test_results`
(2.9s). Correctness and simplicity reviews found no remaining actionable blockers
in the completed follow-up. `git diff --check` passes. No frontend changes or
external replay requests were made; the previously documented client power-spade
migration is still required before deployment.
