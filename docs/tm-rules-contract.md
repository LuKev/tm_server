# Assigned-faction 1v1 rules contract

Profile: `tm-base-1v1-assigned-v2`. Rules, state, and action schema versions: 2.
This document defines the supported competition, not a claim that all of its
rules have already been independently verified. See the conformance evidence
and corrective plan before admitting a training run.

## Game and setup

- Exactly two players on the original base map, no fan/Fire & Ice factions,
  alternate maps, expansion final scoring, auctions, or faction drafting.
- All fourteen base factions are supported, with distinct home terrains:
  Chaos Magicians, Giants, Fakirs, Nomads, Halflings, Cultists, Alchemists,
  Darklings, Mermaids, Swarmlings, Witches, Auren, Engineers, and Dwarves.
- Assigned factions and initial seat order are inputs. Both start at 20 VP,
  using the original starting resources/cult positions of their faction. This
  intentionally does not use later variable-starting-VP tables.
- `SetupModeSnellman` is the existing engine name for standard interactive
  dwelling/bonus selection; it is not permission to use tolerant replay rules.
- Initial dwellings follow forward then reverse order; Nomads place their third
  afterwards; Chaos Magicians place their single dwelling last. Bonus cards are
  chosen in reverse initial order. Starting structures are free and do not
  trigger construction scoring or leech. Income follows completed setup.
- Draw five distinct bonus cards from the nine base cards and six distinct
  round-scoring tiles from the eight base tiles. Exclude the promotional
  Temple/Priest scoring tile. Spade-scoring cannot occupy rounds five or six.
  Draw scoring tiles from round six backwards, withholding Spades only for the
  first two draws and returning it to the eligible pool for rounds four to one.
  Use the five base town-tile types and their standard counts; no promotional tiles.
- Setup randomness is seed-controlled and public. Faction assignment is not a
  learned decision in this profile. Seed alone is not a portable cross-version
  game identity; retain the generated setup and version as well.

## Play and results

- Play six rounds with income, actions, pass scoring, and cleanup according to
  the supported rules; no cult round bonus after the final round.
- The first player to pass starts the next round. The engine stores full pass
  order: for exactly two players this is equivalent to the original clockwise
  rule. Do not extrapolate that equivalence to multiplayer.
- Leech offers resolve once. Accept the maximum permitted by the offer, charging
  capacity and VP affordability, or decline. No arbitrary fragments or repeated
  one-power discounts. Original offer identity governs triggered reactions.
- Power-spade actions expose legal target terrain, including intermediate
  transformation; costs, leftover allocation, and building rights follow the
  original single-action rules, not an automatic home-terrain shortcut.
- Final utility is win/draw/loss from total final VP. Equal VP is a draw, not a
  remaining-resource tiebreak. Preserve VP breakdown and margin for reporting.
- No time-based forfeits, resignation labels, replay funding, or artificial
  terminal draws. A search/game ply limit is an infrastructure failure, not a
  training outcome. Evaluation may use fixed simulations or a declared time
  budget; neither changes the game's rules.

## Decision timing and action compatibility

Leech resolves before the builder's favor/town rewards, including for responders
who have passed. Simultaneous favor, town, and Darklings ordination choices can
be ordered by their owner. Mandatory reactions cannot be bypassed.

The AI profile explicitly retains an after-action window for free conversions
and eligible Mermaid town choices. `FinishTurn` closes that window; it is not
passing for the round. A delayed Mermaid town before the main action does not
consume the main action, including when the town requires a cult-top choice.
Source-derived timing fixtures exercise these paths, but do not replace strict
external complete-game evidence.

### User-adjudicated clarifications

The 2026-09-20 rules discussion is part of this competition contract:

- Simultaneously founded towns provide their keys immediately, before reward
  selection. The owner chooses which town's reward to resolve first. Available
  keys must be used when an all-track cult reward can reach eligible top spaces;
  the player chooses among tracks if there are insufficient keys for every top.
- Mermaid river towns are choices on the current board, not reserved historical
  opportunities. Expose every valid river anchor, including overlapping
  alternatives. Joining an existing town invalidates a former separate-town
  opportunity. Claims are legal before/after the owner's main action, never
  after passing or during an opponent's turn/leech response.
- Chaos double actions are two consecutive turns, with reactions/rewards and
  free conversions between independently selected main actions.
- Leftover spades on subsequent hexes cannot be topped up with workers, for
  either shared power actions or Halflings' stronghold. Unused stronghold spades
  may be forfeited while still building on a transformed home-terrain hex.
- Cult-reward spades may be split across intermediate terrains, and use only
  permanent shipping. Final Fakir/Dwarf networks chain all legal flight/tunnel
  connections without paying or requiring resources.
- Alchemists cannot convert newly earned spade power to fund a dwelling during
  that same action. A priest town reward remains selectable at the piece limit;
  only the unavailable priest is lost.
- Permanent cult-priest placements are legal even with no advancement.
  **By explicit user adjudication, cult-bump special actions also remain legal
  with no advancement.** This supersedes the earlier implementation's reading
  of FAQ 2.12; it is not presented as independently verified official wording.

Two working interpretations are implemented but remain **tentative**: cult
spades may reverse on the same hex (including Halflings VP), and simultaneous
Halflings town rewards/stronghold spades may resolve in either order. Tests are
explicitly labeled tentative so later adjudication can change these expectations.

Two-spade power actions are atomic: validate both destinations against the
pre-action board, transform both, then optionally build one dwelling. Building
is associated with the first target; reorder targets to build on the other
eligible target. Unused rewards may be declined. A second transformation cannot
gain reachability from the dwelling just built.

Websocket power-spade requests now require `spadeActionVersion: 2` and support
explicit first/second terrains and targets. Older requests fail explicitly;
they must not silently lose their second spade. **The existing frontend has not
been migrated and must be updated before deploying this backend change.**
Replay import normalizes legacy notation, but its tolerant execution is not
strict gameplay conformance evidence.

## Artifact admission and historical quarantine

`RulesVersion`, `StateSchemaVersion`, and `ActionSchemaVersion` in Go and their Python counterparts are
2. Canonical records, trajectory manifests, learner admission, and checkpoint
manifests must agree. Tests explicitly reject version-1 rules/states/actions. Existing
checkpoint files and replay shards remain untouched, but are inadmissible in
the new lineage. Do not relabel their manifests to bypass rejection.

A fresh lineage starts only after the corrective plan's rules, representation,
search, recovery, and evaluation gates pass. This rules-only implementation does
not reopen expensive training. Model/search issues from the review are separate
unfinished work.

## Sources and executable checks

Use the [publisher base rulebook](https://www.feuerland-spiele.de/fileadmin/game/Terra_Mystica/Terra_Mystica_rules_EN_Web.pdf)
and identified official clarifications for expected results. Distinguish original
rules from [later starting-VP recommendations](https://capstone-games.com/wp-content/uploads/2020/09/TM_new-starting-VP_US_Letter.pdf).
Do not infer rules from the implementation being tested.

`az.NewBaseGame` explicitly sets setup/order/final-scoring settings and uses the
base-only scoring initializer. `az.NewPosition` rejects out-of-profile state,
including promotional scoring tiles. `TestAssignedCompetitionProfile` checks
the setup contract; trajectory and Python checkpoint tests check the artifact
boundary. These mechanical checks complement, but cannot replace, the
rule-by-rule independent conformance audit.
