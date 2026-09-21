# First-principles engine review — 2026-09-19

Reviewed source revision: `1d8fba076cd86dc437c24392934e0eb84c826ef3`.
Scope: rules, representation, search, learning, evaluation, execution, and the
existing plan. Three adversarial reviews separately examined rules, learning,
and search/evaluation. This is a review, not a rules fix or a new training run.

## Conclusion

We have a working experimental pipeline, not yet a validated strong Terra
Mystica engine. “Only scaling remains” is not justified. There is a reproducible
rules exploit, missing legal choices, a policy-head information bottleneck, and
a misleading low-search baseline. Existing checkpoints demonstrate pipeline
execution; their reported wins do not establish strength at correct TM.

The broad AlphaZero design remains appropriate. Correctness, search quality,
representation, and trustworthy measurement must precede more expensive runs.
This review qualifies the milestone-completion and strength conclusions in the
existing plan and local-scaling evidence; those documents remain historical
records rather than independent evidence of correctness.

## What the audit established

### 1. The rules can reward illegal play

**Confirmed by execution:** a three-power leech offer can be accepted as three
one-power fragments for zero VP, instead of costing two VP. Starting at VP 20
and power bowls `(0,12,0)`, full acceptance ends at VP 18 and `(0,9,3)`;
three one-power acceptances end at VP 20 with the identical power state. Every
fragment was returned by the public legal-action interface and applied normally.

The split and requeue are in
[`action_power_leech.go`](../server/internal/game/action_power_leech.go), and
[`leechCandidates`](../server/internal/az/enumerate.go) exposes the choices.
Costs are recalculated independently for each fragment. Source inspection also
shows no cap preventing acceptance from taking VP below zero.

The published rules prohibit arbitrary partial acceptance to save VP: reductions
are for power capacity or avoiding a negative score. See “The price of Power”
in the [publisher's rulebook](https://www.feuerland-spiele.de/fileadmin/game/Terra_Mystica/Terra_Mystica_rules_EN_Web.pdf),
also available as a [rulebook mirror](https://cdn.1j1ju.com/medias/9c/2c/c8-terra-mystica-rulebook.pdf).

**Confirmed by source inspection, not a separate execution repro:** power-spade
actions force transformation to home terrain and buy missing spades. Their
action schema has no target-terrain choice. This excludes stopping at an
intermediate terrain, which the rulebook explicitly permits. See
[`power_actions.go`](../server/internal/game/power_actions.go), especially
`requiredSpadesForTransform` and `executeTransformWithFreeSpades`.

These are examples, not a complete list of rules errors. The existing oracle
shares action conversion and rules execution with the engine. Agreement catches
enumeration discrepancies, but cannot establish that the shared rules are right.
Synthetic action-kind coverage also does not prove reachable, full-game fidelity.

**Required:** independently derived rules fixtures, adjudicated complete games,
legal-set completeness tests, and per-decision differential comparison against
an independent implementation where available. Compare resources, reactions,
terrain, towns, cults, and scoring—not just final scores. Exclude incompatible
replay after rules corrections; retain old artifacts only as historical evidence.

### 2. A larger CNN cannot fix the current policy connection

In [`model.py`](../server/internal/az/python/model.py), the policy receives action
features, up to two referenced cell embeddings, and global scalar features.
Absent cell references are zeroed. Pooled board features reach the value head
but not the policy head.

Consequently, with global/action features held fixed, the relative logits of
nonspatial actions cannot respond to a different board arrangement. This affects
choices such as shipping, cult advancement, conversion, and passing. Other legal
actions can change the softmax denominator; that does not restore the missing
relative preference between these choices.

**Required:** feed pooled spatial context to every candidate-action score,
alongside global state and relevant local embeddings. Test sensitivity to board
changes. Audit whether observations preserve all pending-event identities,
ordering, action parameters, and ownership needed for a Markov state. Encoding
collisions in unreachable schema cases are warnings, not proof of a live bug.

Keep the residual hex-CNN initially. A corrected 10-block × 128-channel model is
a reasonable comparison point, a small model is useful for debugging, and the
existing 20 × 256 model is a candidate—not an established optimum. Compare
strength at equal wall-clock budgets, not parameter counts.

### 3. The search evidence needs to be rerun

[`selectPUCTEdge`](../server/internal/az/mcts.go) multiplies the prior by
`sqrt(totalVisits)`. At a fresh node this is zero, all scores tie, and the first
lexically sorted action wins. With one simulation, the selected move is therefore
the first action, not the network's preferred action. The previously used
one-simulation comparator is not a meaningful policy-only baseline.

At eight simulations this also consumes a substantial fraction of search on an
ordering artifact. Most positions in the retained runs had more legal actions
than simulations. Low-budget search can work, but this implementation and budget
have not demonstrated adequate policy improvement.

**Required:** explicit first-play/initial-visit handling and tie behavior;
action-permutation tests; exact small-tree tests including consecutive decisions
and opponent reactions; a genuine policy-only comparator. Re-run search curves
with independent seeds and fixed-time as well as fixed-simulation comparisons.
The previous claim that 32 simulations is a measured strength peak is too strong.

Search currently starts new trees across moves. Retained subtrees and
checkpoint-scoped inference caches are promising after state identity and backup
semantics are proved. Test low-budget alternatives such as Gumbel search as
ablations, not assumptions. [DeepMind's mctx](https://github.com/google-deepmind/mctx)
provides JAX-native implementations and describes the conditions behind its
policy-improvement claims; it is not a drop-in accelerator for the current Go core.

### 4. Successful training is not established learning strength

The retained two-cycle run contains 336 games and 16,307 new positions. It used
184 SGD steps and 5,888 sampled examples, batch size 32, a roughly 29.1-million-
parameter network, and eight self-play simulations. Sampled examples are not
necessarily distinct positions.

The second checkpoint beat the first 9–7. The stored paired 95% confidence
interval is approximately 0.082–1.0: this is inconclusive. Its 10–6 result against
`uniform` is also small-sample evidence, and that opponent is uniform-prior,
zero-value MCTS—not a true random-action player. Neither result validates play
under correct rules.

The learner's before/after diagnostics use opening positions from the same replay
being trained on. They are neither held-out validation nor a representative
sample of full games. Arena value calibration on mixed-policy game continuations
is useful diagnostically, but should not independently veto a model as weaker.
Thousands of correlated plies are not thousands of independent strength trials.

**Required:** distinct training, tuning, and untouched evaluation seeds; paired
seat/faction matchups; true random, heuristic, policy-only, archived-network, and
eventually expert/external opponents. Report WDL with appropriate uncertainty,
VP margins, faction/seat breakdowns, and time per move. Use enough independent
games for the effect size being claimed, often hundreds or more. Mechanical
`strength_valid` checks do not certify rules correctness or statistical strength.

### 5. Efficiency work must preserve the experiment

The bounded cycle runner refuses a nonempty output directory and has no input
for resuming an external checkpoint/replay lineage. The learner itself can
restore optimizer state; the missing capability is orchestration-level recovery.
Replay age is tied to filesystem modification time, which is fragile on restore.

The pipeline also repeatedly serializes/encodes positions, decodes raw replay,
and makes per-example device transfers. Legal enumeration validates candidate
actions through full state transitions. These are profiling candidates, not
proof that a C++ rewrite is necessary. Current concurrency is not a fully
independent asynchronous actor system, and evaluation still delays subsequent
cycles even though it is not a promotion gate.

**Required:** recoverable local runs with checkpoint, optimizer, replay sequence,
configuration, compatibility fingerprints, and counters. Test an interruption
and restoration. Profile rules, search, encoding, IPC, GPU inference, replay I/O,
learner, and evaluation separately before choosing the next optimization.
Use compact indexed replay and batched transfers if measurements justify them;
retain canonical audit records separately. Track samples per newly generated
position and replay age, not just nominal batch size.

## Ground-up design and acceptance gates

1. **Define the actual competition.** Specify map, factions, setup, faction
   selection or bidding, starting VP, tie treatment, and seconds per move.
   Current self-play fixes factions before setup. That cannot train drafting.
   Provisionally target the existing base-map, 14-faction profile, but distinguish
   strong play within that profile from the strongest possible tournament engine.

2. **Establish a trusted simulator.** Resolve the confirmed rules failures, then
   systematically audit every phase and faction against external rules evidence.
   Add golden transitions, full-game records, invariants, legal-set tests, and
   exact small scenarios. Gate: independent conformance evidence, not merely
   agreement between two wrappers around the same transition function.

3. **Prove observations and search.** Ensure no strategic choice or state
   information is lost. Preserve decision-owner backup semantics. Connect global
   board context to the policy, fix ordering effects, and test forced choices,
   reactions, terminal scoring, and action permutations. Gate: exact toy-game
   solutions and invariant checks pass; neural preferences influence low-budget
   search as intended.

4. **Make experiments recoverable and evaluation trustworthy.** Implement and
   test resumption before spending days training. Establish the opponent suite,
   held-out setups, timing protocol, and baseline confidence intervals. Gate:
   interrupted runs recover compatible state; measured strength claims survive
   independent paired evaluation.

5. **Demonstrate repeatable learning.** Start with a corrected moderate CNN and
   enough search to improve decisions. Train latest-network self-play continuously,
   with visit-distribution policy targets and terminal WDL value targets. Compare
   multiple seeds and checkpoints over meaningful training horizons. Gate:
   repeatable held-out improvement over policy-only, baseline, and earlier agents,
   rather than a single 16-game result or decreasing training loss.

6. **Optimize playing strength per unit of compute.** Jointly vary network size,
   search allocation, replay reuse, and learner throughput. Profile tree reuse,
   compact state, batched inference and transfers, and encoded replay. Consider
   auxiliary predictions or expert data as explicit ablations if the objective is
   strongest play rather than strict tabula-rasa replication. Gate: improvements
   survive equal-time comparisons and do not change rules semantics.

7. **Scale the proven system.** Only then select longer Mac runs, external GPUs,
   distributed actors, or a native-core rewrite. Maintain continual training;
   an optional deployment champion is separate from the learning loop. Preserve
   immutable checkpoints, manifests, evaluation results, and tested recovery.

The [AlphaZero paper](https://arxiv.org/html/1712.01815v1) supports the central
policy/value-network-plus-search design and continual use of the latest network,
not a 55% promotion barrier. Its compute budget is not evidence that a similarly
sized network is efficient on a Mac. AlphaZero supplies the learning structure;
it does not supply TM rules, a sufficient encoding, or statistical validation.

[KataGo's research](https://arxiv.org/html/1902.10565v5) is relevant to efficient
training through global context, auxiliary predictions, and search-budget
allocation. Its reported improvements in Go are not speedup guarantees for TM.

## Framework decision

Retain Go rules/search plus PyTorch initially if correctness and profiling support
it. Build an OpenSpiel adapter or isolated native prototype if it helps obtain
independent tests or measure a concrete bottleneck. OpenSpiel still requires a
correct TM implementation; moving a flawed transition function to C++ does not
solve conformance. Its [AlphaZero documentation](https://openspiel.readthedocs.io/en/latest/alpha_zero.html)
explicitly describes illustrative implementations rather than a turnkey
superhuman engine, and includes C++/LibTorch options. Evaluate the particular
backend, not the library name.

Do not simultaneously rewrite the rules, replace the learner, change the network,
and alter the search algorithm. That would destroy attribution and multiply
verification work. Pure JAX approaches are a larger architectural commitment
because the recurrent simulator must fit their compiled execution model.

## What remains useful and what was verified

Potentially reusable foundations include typed candidate actions, masked hex
convolutions, legal-action masking, decision-owner-aware value backups,
checkpoint hashing, optimizer restoration, and paired evaluation plumbing.
Retain them subject to targeted tests; neither blanket trust nor blanket deletion
is warranted.

Fresh verification from `server/`:

```sh
bazel test //internal/az:az_test //internal/az:python_test //internal/game/... //:az_loop_smoke_test --test_output=errors --nocache_test_results
```

All six selected suites passed uncached. A separate temporary adversarial leech
test then reproduced the exploit through `LegalActions` and `Apply`; it was
removed after the diagnostic. No implementation changes were made in this review.

There is no defensible date or model size that guarantees “strongest possible.”
On the current Mac, optimizing strength per second is essential; an unconstrained
training budget could justify a different model or later distillation. The next
commitment should be correctness and measurement gates, not an expensive run of
the existing configuration.
