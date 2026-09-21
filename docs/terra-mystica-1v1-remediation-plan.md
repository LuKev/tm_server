# Trusted 1v1 Terra Mystica engine: implementation plan

Status: active corrective implementation, 2026-09-20. No milestone below is
complete merely because an equivalent component exists in the current engine.
This is the active corrective plan; it supersedes the earlier plan's completion
claims and experiment ordering. Evidence and rationale are in the
[first-principles review](terra-mystica-1v1-first-principles-review-20260919.md).

## Implementation checkpoint — 2026-09-20

- R0 — complete: the [assigned-faction contract](tm-rules-contract.md) and
  [conformance ledger](terra-mystica-rules-conformance.md) are implemented.
  Rules/state/action version 2 rejects the historical lineage without deleting it.
- R1 — complete: leech fragmentation, atomic power-spade choices, reward refusal, legal
  terrain routes, and caller compatibility checks are implemented with regression
  tests. The websocket API explicitly rejects obsolete power-spade requests;
  the frontend requires migration before deployment.
- R2 — complete for the adjudicated contract: source-derived boundary fixtures, additional timing/faction/scoring repairs,
  all-14-faction natural-game invariants, and three detected mutation experiments
  are implemented. The user clarified that replay primarily tests support for
  recorded moves, not equality with the complete legal rules. A large external
  1v1/Fakirs corpus is **not a required acceptance gate**. The bounded follow-up
  uses confirmed user rulings, explicit legal/illegal boundary positions, and
  adversarial reviews. Two working interpretations remain tentative and are
  labeled in the contract and tests. External records remain supplemental;
  do not label internal tests independent complete-game validation.
- R3 onward: not implemented by this rules-focused change. Expensive training
  remains closed until the subsequent representation/search/recovery gates pass.

Final verification results are recorded in the conformance ledger. Implemented
code is not a substitute for satisfying a milestone's evidence requirements.

## Objective and constraints

Build the strongest demonstrable agent for a specified 1v1 competition profile,
initially on the local Mac. Preserve the AlphaZero core: exact rules, one
policy/value network, search-derived policy targets, terminal WDL values, and
continual latest-network self-play. Keep the residual hex-CNN as the baseline.
Do not use a 55% promotion gate to block learning; deployment selection is separate.

Do not restart expensive training on the current rules. Preserve existing runs
as historical artifacts, not admissible new training data or validated strength
benchmarks. No destructive cleanup or blanket rewrite is required.

Use the existing Go engine and PyTorch components initially. All executable
checks and experiments must have Bazel entrypoints. No frontend, deployment,
distributed training, or framework migration is needed for the first trusted loop.

At each milestone: add failing regression tests first where applicable; implement
the smallest coherent change; run the relevant Bazel tests; obtain adversarial
correctness and simplicity reviews; resolve findings; reread this plan and update
its evidence/status. Reviewers must challenge expected results and shared
assumptions, not merely check that tests pass. Keep changes independently reviewable.

## R0 — Freeze the target and evidence boundary

Deliverables:

- A versioned competition profile defining base map, all 14 base factions,
  exact setup variant, turn/pass order, starting VP, scoring/ties, and allowed
  optional timing. Record deviations from official rules explicitly; do not call
  an engine simplification an official rule.
- Initially evaluate assigned-faction play. Reserve faction choice/auction as a
  separate milestone if the intended competition includes it; do not claim full
  tournament strength without it.
- Keep both fixed-simulation and fixed-time evaluation. Until a user-facing move
  limit is chosen, measure a time curve rather than silently optimizing one limit.
- Mark pre-correction checkpoints/replay as incompatible for the new lineage.
  Preserve their hashes and provenance; do not delete them.
- A conformance matrix: rule/phase/faction, authoritative source, fixture,
  independent cross-check, implementation location, status, unresolved question.

Exit: every supported rule family is listed and the new artifact boundary is
explicit. Ambiguous rules are recorded as blockers for the affected coverage,
not resolved through undocumented assumptions.

## R1 — Repair demonstrated rules errors

Primary files: `game/action_power_leech.go`, `game/resources.go`,
`game/power_actions.go`, `az/search_action.go`, `az/enumerate.go`, related tests.

1. Permanently reproduce the free-leech exploit through `LegalActions` and `Apply`.
   Resolve an offer once: decline or accept the rule-permitted amount, capped by
   capacity and VP affordability. Do not requeue the unaccepted remainder as a
   fresh discount. Verify reaction events occur once per original offer.
2. Cover offer sizes, empty/full power bowls, zero/low VP, capacity truncation,
   multiple sources, responder order, and Cultists interaction. Test direct game
   actions as well as the AI adapter so invalid requests cannot bypass enumeration.
3. Add explicit destination terrain to applicable power-spade actions and their
   serialization/encoding. Audit intermediate terrain, optional worker top-ups,
   two-spade allocation, optional building, and faction-specific restrictions.
   Do not assume the only defect is the missing field.
4. Verify all callers remain compatible or explicitly reject obsolete actions;
   do not silently reinterpret old serialized actions.
5. Bump the affected rules/action compatibility versions and test rejection of
   incompatible replay/checkpoints before a new learner can admit them.

Exit: independent rulebook-derived expected results pass, including the exact
three-power exploit, and legal action sets contain required intermediate choices
without introducing illegal alternatives. Existing backend tests remain green.

## R2 — Establish independent rules conformance

Audit systematically rather than stopping after the known failures:

- Setup and placement; income and round transitions; pass/bonus scoring.
- Terrain distance, reachability, bridges, shipping, digging, building upgrades.
- Power cycling/burning/conversions, shared actions, leech and response ordering.
- Cult progression, keys, priest availability, favor/town eligibility and choices.
- Connected networks, town formation, final scoring and ties.
- Every supported faction ability and interactions with pending decisions,
  conversions, multi-actions, towns, and passing.

Build independently calculated golden positions at boundaries and ordinary cases.
Keep the exhaustive enumeration oracle, but label it an implementation-agreement
test. Its candidate domains must independently cover every legal parameter; a
shared omitted field must not make both enumerators agree incorrectly.

Add property tests for clone isolation, failed-action atomicity, valid resource
bounds, accounting under documented sources/sinks, deterministic replay, and
natural termination. Minimize failing randomized traces into permanent fixtures.
Use a small set of deliberate fault injections to ensure critical tests detect
wrong costs, omitted choices, scoring errors, and wrong decision owners.

Use available externally checked game records as supplemental positive coverage.
Do not require a large 1v1 or Fakirs corpus. Compare state and score at each
decision where the records support it. Differential-test against
an independent implementation when a compatible implementation and records are
available. If unavailable, use independently adjudicated fixtures and publish
the remaining coverage gap; do not substitute a self-replay for an external check.

Exit: no known failures of the adjudicated competition contract; each prioritized
rule interaction has explicit positive/negative tests and adversarial review;
internal complete-game invariants cover all factions and phases. Tentative
interpretations must stay visible rather than being labeled verified rules.
This establishes a documented confidence level, not a mathematical proof.

## R3 — Repair and prove representation and search

Representation work (`python/schema.py`, `python/model.py`):

- Inventory each search-relevant state field and its encoding. Pending events
  must retain source, owner, order, and identity where strategically relevant.
  Audit action parameters, including compound/Chaos choices, for reachable
  collisions. Remove redundant unsupported schema paths only with evidence.
- Feed pooled board-trunk features into every legal-action scorer, alongside
  global features, action features, and referenced-cell embeddings. Keep one
  shared candidate scorer; do not introduce separate networks by action family.
- Test board-context connectivity with controlled weights/gradients and paired
  positions, not an assumption that random weights always rank moves differently.
- Verify relative player identity, axial neighbors, masks, ragged candidates,
  encode/serialize consistency, and rules/model compatibility rejection.

Search work (`az/mcts.go`, related tests):

- Specify first-play urgency, initial parent visit convention, tie-breaking,
  root noise, temperature schedule, and visit-budget accounting explicitly.
  Eliminate lexical-first behavior at a fresh root. A one-simulation search need
  not be a strong agent; it must not masquerade as policy-only evaluation.
- Add a true policy-only mode and a true random-legal-action mode, separately
  named from uniform-prior MCTS.
- Test exact small trees: terminal outcomes, same-player turns, opponent
  reactions, forced moves, and draws. Verify value signs by decision owner.
- Test invariance under candidate reordering, allowing documented tie randomness;
  compare serial/batched results under controlled scheduling and numeric tolerances.
- Keep complete legal choices. Any compression of forced or equivalent actions
  must preserve decision timing and training/search semantics and be separately
  proved; do not prune actions because they appear strategically weak.

Exit: tiny-set overfit works, encoding audits have no unresolved reachable
information loss, exact search tests pass, and low-budget behavior is no longer
driven by serialization order. Use debug models and fixtures, not long training.

## R4 — Make training recoverable and evaluation trustworthy

Recovery (`tools/run_az_cycles.sh`, checkpoint/trajectory/learner code):

- Add explicit new-run and resume modes. Persist last completed stage, next
  cycle, immutable checkpoint/replay identities, optimizer and random state,
  logical replay sequence, configuration, and compatibility versions.
- Publish artifacts atomically. Restart must detect partial outputs and avoid
  duplicating games, learner updates, or checkpoint adoption. Keep configuration
  overrides explicit rather than silently changing an existing experiment.
- Use logical sequence for replay age, not filesystem modification time. Validate
  corruption and missing artifacts before resuming. Preserve exact source identity;
  permit compatible cross-revision reuse only through an explicit tested policy.
- Inject interruption during actor output, training/checkpoint publication, and
  evaluation. Compare uninterrupted/resumed CPU runs on a fixed data stream;
  on MPS define numerical tolerances rather than promising bitwise determinism.
- Retain recovery packages durably under the established storage policy; conduct
  a restore test. A path under `/tmp` is not recovery storage.

Evaluation (`az/arena.go`, scaling harness):

- Separate development/tuning setups from a final untouched test suite. Existing
  repeatedly inspected holdouts are development data, not an untouched final test.
- Pair setups and faction assignments with swapped seats. Aggregate uncertainty
  by independent setup/pair, not by correlated positions. Freeze opponent/model
  identity, rules, time/search budgets, and seed selection in each report.
- Include true random, simple documented heuristic, policy-only, uniform MCTS,
  previous checkpoint, and long-term frozen anchors. Add external/expert play
  when available. Report individual matchups, not just one aggregate rating.
- Use WDL as primary strength evidence. Report VP, faction/seat, latency,
  completion, policy entropy, and calibration as diagnostics. Do not reject a
  model solely because mixed-policy Brier worsens or training loss increases.
- Replace opening-only same-replay loss reporting with explicitly labeled
  training diagnostics and separate held-out diagnostics spanning phases and
  factions. Test the split and sampling coverage; no held-out diagnostic position
  may enter optimizer batches. Keep these development diagnostics distinct from
  the final untouched strength suite.
- Before a confirmatory match, specify its minimum worthwhile effect, sample
  size/precision target, and stopping rule. Account for paired outcomes. If
  sequential evaluation is used, use valid sequential bounds; otherwise use a
  fixed sample. An inconclusive interval remains inconclusive.
- Rename or document `strength_valid` as mechanical integrity only.

Exit: restore drills pass; baselines behave as advertised; reports distinguish
mechanical validity from strength evidence; evaluation cannot contaminate training.
Keep evaluation bounded and less frequent than training; a separate service is
not required yet.

## R5 — Demonstrate learning on the corrected game

Start a fresh lineage after R1–R4; do not warm-start the headline experiment from
the flawed-rules checkpoint. Initial experimental defaults, not proven optima:

- Debug: 4 residual blocks × 64 channels. Baseline: corrected 10 × 128 hex-CNN.
  Compare 20 × 256 only after obtaining meaningful learning curves.
- Preserve policy cross-entropy plus terminal WDL value MSE and explicit L2.
  Start with momentum SGD; select learning rate using a bounded stability screen
  on corrected data, including gradients and value saturation. Do not inherit
  the old three-step learning-rate result as an established optimum.
- Initially benchmark learner batches 32/64/128 if memory permits. Choose a
  practical throughput point, then validate learning; inference batch and learner
  batch are separate settings. Leave memory headroom for actors and OS use.
- Choose self-play search from corrected same-network policy-only/8/32/128
  comparisons and downstream learning, at both equal data and equal total compute.
  Eight simulations is a candidate, not the default conclusion.
- Define replay sampling explicitly. Start with uniform positions in a bounded
  recent window; balance faction/setup generation and measure coverage. Treat
  matchup/game-balanced sampling as a recorded ablation rather than an accident.
- Record actual sampled examples per NEW replay position, effective reuse,
  window age, and checkpoint lag. Bound learner work directly rather than tying
  it implicitly to the total accumulated buffer each cycle.
- Adopt latest checkpoints between games. Save immutable snapshots, evaluate on
  a planned cadence, and keep learning independent of champion selection.

Experiment order:

1. Tiny deterministic overfit and end-to-end debug smoke with interruption.
2. Bounded corrected-data optimizer stability screen and throughput measurements.
3. Corrected search-budget screen, then one modest end-to-end pilot. On an
   untrained network the screen establishes sanity and throughput, not the best
   strength budget. Revisit search allocation on trained snapshots and compare
   downstream training before locking a production setting.
4. At least three independent training seeds for the selected baseline; use
   comparable generated-data and total-time budgets. A tiny pilot is not a
   substitute for this learning experiment.
5. Evaluate planned snapshots against frozen anchors on development setups;
   perform a preregistered final comparison on untouched setups.

Exit: repeatable improvement across runs and held-out opponents with uncertainty
appropriate to the claim, no rules/integrity failures, and finite stable training.
If learning fails, inspect search-target quality, coverage, representation,
gradient/value behavior, and replay reuse before simply adding compute.

## R6 — Optimize measured bottlenecks, then model capacity

Profile end-to-end wall time and allocations for enumeration, clones, transitions,
hashing, encoding, IPC, inference queue/service, replay decode, learner, and arena.
Include complete-game time, decisions/game, unique leaves, effective batch fill,
memory high-water mark, and checkpoint lag. Benchmark more than one faction/phase.

Apply one change at a time, in measured order:

1. Batched device transfers and compact indexed/materialized replay; keep raw
   canonical audit records. Bound caches by memory as well as item count.
2. Retained search subtrees and checkpoint-scoped inference caches, with explicit
   rules/state/checkpoint invalidation and root-noise handling. Compare at equal
   time; tree reuse intentionally need not preserve the old search trajectory.
3. Remove measured serialization/allocation costs. Add a compact transport only
   if IPC/encoding is material. Delay make/unmake until clone cost justifies it.
4. Tune shared inference batching and actor concurrency on the actual corrected
   model/search workload. Report learner/actor contention on the single Mac GPU.
5. Compare network sizes and search allocation by downstream strength per total
   training hour and strength per move-time budget, not throughput alone.

Optional later ablations: Gumbel low-budget search, randomized playout budgets,
auxiliary outcome/score predictions, explicit connectivity features, expert-data
pretraining, or distillation. Each needs an isolated hypothesis and equal-compute
comparison; none should delay the corrected baseline. Keep terminal WDL as utility.

Exit: each retained optimization has correctness evidence and a repeatable
end-to-end benefit. Capacity increases must improve relevant playing strength.

## R7 — Decide framework, scale, and full tournament scope

Only now consider a native-core/OpenSpiel prototype, a JAX-native simulator,
external GPUs, or distributed actors. A library does not supply verified TM rules.
Prototype one representative full path and compare semantics, total throughput,
memory, implementation complexity, and test reuse before selecting a migration.
Do not rewrite the rules, learner, and search simultaneously.

Scale the measured bottleneck with explicit cost/time limits, durable recovery,
bounded checkpoint lag, and identical evaluation. Add faction selection/bidding
if required by R0 and separately validate its reward/setup semantics. Deployment
and public UI come after engine validation; a deployment champion may be selected
without imposing a promotion gate on learning.

Exit: gains hold against independent strong opponents under the specified
competition conditions. Until then, describe the engine's measured level rather
than claiming superhuman or strongest-possible performance.

## Dependencies, verification, and first implementation batch

Critical path: R0 → R1 → R2 → R3 → R4 → R5 → R6 → R7.
After R0, representation/search and recovery scaffolding can proceed alongside
rules auditing, but expensive training waits for all R1–R4 gates. No calendar
promise is credible until the independent conformance audit bounds the unknowns.

Existing core regression command, from `server/`:

```sh
bazel test //internal/game/... //internal/az:az_test //internal/az:python_test //:az_loop_smoke_test //:az_scaling_curves_smoke_test --test_output=errors --nocache_test_results
```

Add new fixtures and recovery/evaluation checks to appropriate Bazel targets.
Run broader backend regressions for shared-rules changes. If a frontend change
becomes necessary, the repository additionally requires `//:client_build_test`.
Passing this command alone is not independent rules verification; apply the
adjudicated-fixture and evidence criteria above.

The initial R0–R2 implementation and clarification follow-up are complete for
the documented competition contract. Next is R3 representation/search, followed
by R4 recovery/evaluation. No new training campaign until those gates pass.
