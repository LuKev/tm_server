# Base 1v1 representation audit

Scope: the assigned-faction, Snellman-setup base-map profile in
`server/internal/az/position.go`; not arbitrary server/import states. This is an
R3 evidence inventory, not a proof that every conceivable JSON object is legal.

## State inventory

| Authoritative state | Network representation / exclusion |
| --- | --- |
| Map hex coordinate, terrain, building type/owner | Axial masked spatial planes; coordinates come from map keys |
| Building power and resource income/cost | Determined by base faction and building type, not independent reachable state |
| Bridges, ownership, endpoint direction | Six directional planes per player |
| Town membership and marker ownership | Spatial flags; marker tile type has no future rules effect, claimed tile counts are global |
| River membership, map ID/custom map | Base geometry is fixed; terrain planes retain water. Non-base map is rejected by adapter |
| Players: faction, resources/power bowls, VP, shipping, digging, bridges, keys, towns, passed, stronghold ability | Player-relative global scalars/one-hots |
| Structure supply | Derived from board counts and fixed base supply |
| Used specials, claimed town/favor tiles, held bonus card | Global flags/counts, including card-presence flag |
| Faction private state | `factions.StateForSearch`: type, terraform cost, stronghold flag, flight range, shipping |
| Round, phase, setup subphase/index/order/counts, turn/pass order/current actor | Global features; fixed sequence lengths cover base 1v1 setup including Nomads/Chaos |
| Scoring tiles, priests sent, power-action usage, bonus coins and shared tile supply | Global features by round/card/tile/action |
| Cult positions, claimed thresholds, top owner, permanent priests by track/space/player | Global features; duplicate legacy player positions retained too |
| Pending free/cult spades, build allowance, skip destinations | Per-player globals and spatial skip planes |
| Favor selection | Presence, owner, count, selected tile set |
| Halflings continuation | Presence, owner, remaining spades, transformed-hex set |
| Darklings ordination / Cultists track choice | Presence and owner |
| Chaos continuation | Presence, owner, remaining actions |
| Town cult-top choice | Presence, owner, continuation timing, advance amount, maximum selections, candidate track set |
| Free-action window | Presence and owner, distinct from current main actor |
| Mandatory town reservations | Count, union, and separate canonical group membership planes retain the partition |
| Delayed Mermaid opportunities | Union/count summary; exact alternative groups and river anchors are recomputable from board, faction, town markers and timing |
| Leech offers | Ordered per-recipient slots: amount, cost, source player/hex, event association |
| Cultist event state | Per-event owner and created/resolved/accepted/declined counters, linked to offer slots |
| Next leech counter / absolute event IDs | Incidental identities; encode equality/linkage, not numeric counter values |
| Final scoring cache | Derived from board/resources/cults/VP; not a separate nonterminal decision input |
| UI names/options, timers, replay tolerance, confirmation snapshot | Removed or rejected by adapter |
| Fan/expansion pending events and player state | Rejected by adapter; not silently compressed into base features |

## Repairs and bounds

The earlier pending-event sums could identify two differently ordered queues or
different Cultist-event associations. Pending continuation owners were omitted.
Town union planes also discarded the partition of separately reserved rewards
if subsequent construction connected them. The encoder now retains these
distinctions. Event IDs are numbered by first offer occurrence so renaming an
incidental counter does not change features.

Five mandatory-town groups per player are sufficient: there are seventeen
building pieces and even sanctuary towns require at least three buildings.
Groups reserve disjoint buildings. Delayed Mermaid alternatives are not reserved
groups and do not consume these slots.

Reachable base 1v1 actions generate at most one leech offer before a response:
`TriggerPowerLeech` aggregates all adjacent buildings into one offer per opponent;
each base construction action places/upgrades at most one building;
`validateActionDecisionWindow` blocks subsequent actions until the response is
resolved. Halflings continuations and Chaos turns cross that same barrier. Two
slots preserve the existing two-offer import/test cases as well. Longer imported
queues fail explicitly rather than silently losing order; they are not claimed
as supported reachable training states. Cultist bonuses disappear when resolved.

The raw Go state/action versions remain unchanged: canonical JSON and legal
production actions have not changed. Feature-name digests and dimensions change,
so checkpoint manifest validation rejects old tensors/weights. This is separate
from the policy architecture compatibility version.

## Actions

The shared candidate representation includes kind/subtype, conversion, both
terrain choices, direction steps, building, cult tracks, bonus/favor/town/faction
choice, both numeric amounts, build/skip/decline flags, and ordered coordinates.
Referenced-cell gathers retain both coordinate roles. Canonical legal-action
enumeration supplies only supported parameters. Public encoding validates them.

Chaos double turns are activation followed by two ordinary decision edges with
reactions between them. Production enumeration does not emit composite
`subactions`. Nonempty legacy composite actions are now rejected by the encoder;
the old subaction-count-only encoding was not an adequate representation of
their contents. Existing empty activation remains supported.

## Evidence and limits

`python/encoding_audit_test.py` pairs records with equal former aggregates and
checks owner distinctions, leech order/source/event association, incidental event
ID renaming, town partition differences, overflow rejection and composite-action
rejection. These are semantic unit fixtures, not claims that every deliberately
mutated pair is independently reachable through a legal opening.

Existing Go contract tests cover legal Mermaid alternatives, town reservation
growth, Halflings reward timing and Chaos reaction sequencing. Existing Python
representation tests cover player-ID renaming, typed action parameters,
serialize/materialize consistency, axial neighbors, masks and version rejection.
The separate policy tests exercise board-context gradients with controlled
weights rather than relying on random output differences.

Remaining proof obligation for full R3 acceptance: connect the pending-event
mutation tests to exported legal-action traces across every supported faction,
especially leech, towns, Halflings and Chaos boundaries. Passing
these focused encoding fixtures alone is not the entire R3 exit criterion.

## Search and policy evidence

The single shared policy scorer now includes masked pooled board-trunk context
alongside globals, typed action features and two cell gathers. Controlled weights
give opposite preferences for identical nonspatial candidates on two boards;
the test checks exact board gradients and zero gradients outside the mask.
A 16-channel, one-block paired-board overfit and the existing real-encoded tiny
overfit pass. These are bounded CPU correctness checks, not strength experiments.
Checkpoint architecture version 2 rejects pre-fix policy heads explicitly.

PUCT uses Q=0 for unvisited edges and `sqrt(max(1, parent edge visits))` for
exploration. Root evaluation is outside the simulation budget. Fresh-root choice
therefore follows priors; exact ties use seeded position/action hashes rather
than candidate order. Zero-temperature selection prefers visits, then priors,
then seeded ties. Tests cover exact forced wins/losses/draws, mixed-owner backup
paths, candidate permutations, and serial/batched/concurrent equivalence.

`az_eval` exposes `--candidate-mode` / `--baseline-mode` as `mcts`, `policy`, or
`random`. Policy-only evaluates the root with zero traversals and no noise;
random samples uniformly from the complete canonical legal set without neural
selection. Uniform-prior MCTS remains a separate option. Format-6 reports record
both modes and effective budgets, rejecting missing modes or contradictory
non-search budgets. Tests verify no traversal, uniform sampling, permutation and
batch consistency, and persisted provenance. Existing development cases are not
claimed to be a new untouched holdout.

Independent adversarial correctness/simplicity reviews found no remaining
actionable defect in this bounded repair. They explicitly did not treat synthetic
record mutations as proof of legal reachability or close the remaining R3 gate.

## Verification — 2026-09-20

From `server/`, all 14 targets passed uncached:

```sh
bazel test //internal/... //cmd/... //:az_loop_smoke_test //:az_scaling_curves_smoke_test --test_output=errors --nocache_test_results --test_timeout=900
```

The AZ suite passed in 182.9s, Python model/learner tests in 31.4s, encoding
audit in 0.4s, and both loop/scaling smoke tests passed. Backend game/replay/
websocket regressions also passed. `git diff --check` is clean. No frontend
changes, new long-running training, or external game-record requests occurred.
