# Trusted 1v1 Terra Mystica engine: implementation plan

Status: active corrective implementation, 2026-10-15. No milestone below is
complete merely because an equivalent component exists in the current engine.
This is the active corrective plan; it supersedes the earlier plan's completion
claims and experiment ordering. Evidence and rationale are in the
[first-principles review](terra-mystica-1v1-first-principles-review-20260919.md).

## Implementation checkpoint — 2026-10-15

- R0 — complete: the [assigned-faction contract](tm-rules-contract.md) and
  [conformance ledger](terra-mystica-rules-conformance.md) are implemented.
  Rules/state/action version 2 rejects the historical lineage without deleting it.
- R1 — complete: leech fragmentation, atomic power-spade choices, reward refusal, legal
  terrain routes, and caller compatibility checks are implemented with regression
  tests. The websocket API rejects obsolete power-spade requests; subsequent UI
  work migrated the client to atomic version-2 requests (`7e5e821`).
- R2 — complete for the adjudicated contract: source-derived boundary fixtures, additional timing/faction/scoring repairs,
  all-14-faction natural-game invariants, and three detected mutation experiments
  are implemented. The user clarified that replay primarily tests support for
  recorded moves, not equality with the complete legal rules. A large external
  1v1/Fakirs corpus is **not a required acceptance gate**. The bounded follow-up
  uses confirmed user rulings, explicit legal/illegal boundary positions, and
  adversarial reviews. Two working interpretations remain tentative and are
  labeled in the contract and tests. External records remain supplemental;
  do not label internal tests independent complete-game validation.
- R3 — complete for the bounded base-profile acceptance criteria: pooled board context in the single policy scorer, architecture
  compatibility rejection, pending-owner/event/town-partition encoding, exact
  seeded search tie semantics, and true policy-only/random-legal arena modes are
  implemented. Controlled gradients, tiny overfit, semantic encoding pairs, and
  exact search tests pass. See the [representation audit](terra-mystica-representation-audit.md).
  Test-only legal traces now connect the engine to Python across all 14 factions,
  including leech, favor, Halflings, Chaos, town and Darklings boundaries. They
  retain complete action provenance, renamed-player replay equality and complete
  candidate-set encoding checks. Terminal display-name leakage into canonical
  hashes was repaired. This is not exhaustive rules verification.
- R4 — complete for the documented stage-recovery/evaluation contract: replay windows use explicit oldest-to-newest input batches
  with content-hash ordering within each batch, independent of mtimes/absolute
  paths. Relocation and two-example seeded sampling checks cover this contract.
  Immutable cycle-parent identities, transactional stage resume, RNG/cursor
  recovery, held-out diagnostics and development/final evaluation separation are
  implemented and adversarially reviewed. The real CPU interruption/relocation
  drill passes; the recovery package was uploaded to Drive, downloaded with an
  identical SHA-256, and successfully resumed locally. See evidence below.
- R5 — pilot and matched-search follow-up complete; no reliable learning gain.
  Behavioral diagnostics reproduced value/search sensitivity; controlled
  fresh learning-rate validation also failed to demonstrate stronger play.
  The two-seed search-versus-update screen also completed without a replicated
  gain. A bounded policy-target fitting diagnostic is next. R6 onward not started.

Final verification results are recorded in the conformance ledger. Implemented
code is not a substitute for satisfying a milestone's evidence requirements.

The 2026-10-10 R3/R4 increment passed all 15 backend/AI/loop targets uncached;
details are in the representation audit. Replay-order tests deliberately reverse
mtime and directory-name order, relocate batches, distinguish two policy targets,
and verify within-batch truncation, duplicate handling and empty-batch rejection.
This establishes stable replay selection/sampling, not interruption recovery.

## Objective and constraints

Build the strongest demonstrable agent for a specified 1v1 competition profile,
initially on the local Mac. Preserve the AlphaZero core: exact rules, one
policy/value network, search-derived policy targets, terminal WDL values, and
continual latest-network self-play. Keep the residual hex-CNN as the baseline.
Do not use a 55% promotion gate to block learning; deployment selection is separate.

Do not restart expensive training until the R4 recovery/evaluation gates pass. Preserve existing runs
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

Recovery (`tools/run_az_cycles.py`, checkpoint/trajectory/learner code):

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

Implementation contract (2026-10-10): the sequential runner requires explicit
`--mode new|resume`. Recovery retries only an unpublished stage from its pinned
parent and seeds; partial attempts are quarantined, not admitted to replay. This
is stage-level recovery, not live minibatch checkpointing. A completed stage has
a SHA-256 inventory and journal entry; configuration and executable/source
identity are immutable. Resume cannot extend the configured cycle count or
silently change device settings, training settings, or source revision. Runtime
identity trusts Bazel's pinned interpreter/wheel definitions; it is not an audit
against a maliciously modified dependency cache or a physical-chip fingerprint.

Learner checkpoints persist optimizer state, sampling/Torch RNG states, and an
update cursor. `--resume-training` continues the same validated replay stream;
normal cycle transitions retain optimizer state but begin a new seeded stream.
Ten percent of canonical state hashes are permanently excluded from optimizer
sampling for development diagnostics, with phase/round/faction coverage reported.
The same state remains excluded if encountered in a later game or cycle. These
are correlated within-game diagnostics, not independent strength measurements.

Evaluation format 7 calls integrity `mechanical_valid`; the deprecated
`strength_valid` alias means exactly the same thing, not evidence of improvement.
The old 28-case suite is development-only. A separate reserved 256-case suite
requires effect/precision/fixed-sample preregistration before play; both suites'
setup seeds are forbidden in actors and learner replay. The final suite is
operationally single-use: after inspecting its outcomes, retire/version it for
later confirmatory claims. A new output directory does not make exposed cases
untouched again. No final-suite games have been played in this increment.

The heuristic baseline is deliberately simple: prefer build-dwelling actions,
then building upgrades, then other legal actions; pass/finish before generic
resource conversions or power burns. Required decisions use legal actions with seeded ties.
It performs no search and is a reproducible weak baseline, not expert play.

R4 acceptance evidence, 2026-10-10:

- All 17 backend/AI/recovery/loop targets passed uncached across the broad run
  and corrected real-drill rerun. `//internal/az:az_test` took 188.3 seconds;
  Python's 29 tests passed in 35.2 seconds, including exact CPU and tolerant MPS
  split/resume tests. `//:az_real_recovery_test` passed in 95.3 seconds.
  The broad run's initial drill failure was a test-output path outside Bazel's
  writable sandbox; it now retains evidence through Bazel undeclared outputs.
- Generic actor/learner/evaluation transaction tests inject failures after the
  command, before directory publication and after publication. The real drill
  kills an actor after partial cycle-2 output, archives a run with a completed
  cycle-1 checkpoint, relocates it, and proves exact trajectory IDs, weights,
  optimizer, RNG/cursor and unchanged repeated resume against the same pinned
  parent. It does not claim to have killed all three real workloads mid-compute.
- Adversarial reviews found and resolved a signal-handler wait deadlock,
  latest-pointer rollback during resume, missing-journal corruption handling,
  and launcher-only source identity. Learner/evaluation/probe reviews found no
  remaining blocking correctness or simplicity issue. Independent new runs may
  have different checkpoint byte hashes because timing provenance differs;
  exact recovery comparisons share the same immutable parent package.
- Durable [recovery package](https://drive.google.com/file/d/1O28f6a4BLpmkeCBf5gWKk2dSAY5OK81O/view)
  is 5,832,122 bytes, SHA-256
  `722c05e5c1b9c58d07877c9d0b1add1f2e2451c77c117ca51068156ecb433204`.
  Cloud readback matched byte-for-byte; a fresh extraction resumed successfully
  to model `67482a386d00feafd41aef985917f6a8d55c5acb97c9b542324dd4f4ea3c6911`,
  matching the uninterrupted reference. Local evidence is retained under
  `artifacts/az/r4-recovery-20261010/`. This debug recovery model is not a
  promoted playing-strength checkpoint and will not warm-start R5.

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

First corrected pilot protocol (declared before results, 2026-10-10):

- Fresh main 10×128 network; no historical checkpoint or replay. A separate
  14-game corrected-data bootstrap is permitted only for a bounded optimizer
  screen, not as a warm-start checkpoint. Compare momentum-SGD learning rates
  1e-4, 3e-4 and 1e-3 for 20 steps at batch 32 with identical initialization and
  sampled positions. Prefer the largest tested rate with finite gradients/loss,
  maximum unclipped gradient norm below 100, less than 10% value saturation
  (absolute value output at least 0.95), and no greater
  than 10% increase in held-out diagnostic loss. If none qualifies, stop and
  inspect rather than launching the pilot. This screen does not establish an
  optimal rate or learning strength.
- Measure synthetic main-network learner throughput at batches 32/64/128, plus
  corrected same-initial-network development searches at policy-only/8/32/128
  on a small fixed prefix. These are sanity/throughput screens, not a contest to
  choose the strongest untrained network. Begin the modest pilot at 8 simulations
  unless measured failures make that setting unusable; revisit on trained data.
- Bound the pilot to four cycles × 32 newly generated games, 100 SGD updates per
  cycle at batch 32, and at most 128 recent shards. Sample eligible positions
  uniformly. Use MPS for inference/learning with one Torch CPU thread, bounded
  actor concurrency, and no overlap between actors and learner on this Mac.
  Record actual sampled examples per new position from actor and learner counts.
- Run weak-baseline sanity checks once on two development pairs against uniform
  MCTS, random, heuristic and policy-only. For recurring pilot evaluation use
  `TM_AZ_EVAL_BASELINES=previous,anchor` with `TM_AZ_EVAL_EVERY=2`: evaluate cycles
  2 and 4 on eight development setup pairs with fixed search budgets against
  previous and cycle-1 frozen weights where applicable. The identical previous/
  anchor at cycle 2 is evaluated once. This bounds recurring arena work to 48
  games, rather than letting a six-baseline matrix dominate the 128 training
  games. Cycle 1 is an early trained anchor,
  not an expert baseline. This small screen has wide paired intervals and is
  exploratory: do not infer confirmed improvement from a favorable point estimate.
  Eight pairs give a conservative Hoeffding half-width of about 0.48; modest
  gains need a substantially larger subsequent comparison. Four hundred updates
  without improvement would likewise not establish that learning cannot work.
- Stop for invalid/incomplete games, non-finite training, failed provenance, or
  failed recovery. Continue latest-network learning independently of arena WDL;
  no 55% promotion gate. The final 256-case suite remains untouched. Subsequent
  multi-seed/equal-compute runs are required before closing R5.

Exit: repeatable improvement across runs and held-out opponents with uncertainty
appropriate to the claim, no rules/integrity failures, and finite stable training.
If learning fails, inspect search-target quality, coverage, representation,
gradient/value behavior, and replay reuse before simply adding compute.

Initial screen evidence (2026-10-10, engine `8acfac5`):

- Apple M5, 16 GiB unified memory, MPS/FP32, one Torch CPU thread. Synthetic
  main-network learner throughput at batch 32/64/128 was approximately
  515/520/529 positions per second (128 legal actions, ten measured iterations).
  The small difference does not justify the higher memory footprint for this
  pilot; retain batch 32. These timings exclude actual replay decoding and do
  not establish end-to-end training throughput.
- Fourteen new bootstrap games covered all 14 factions, generated 968 positions
  in 57.0 seconds at eight simulations, and completed naturally. The fixed
  diagnostic split reserved 104 positions and admitted 864 training positions.
  Bootstrap shards are used only for the stability screen, not pilot replay.
- All three 20-step optimizer trials were finite with zero value saturation.
  Maximum unclipped gradient norms were 13.49, 14.82 and 19.73 for learning rates
  1e-4, 3e-4 and 1e-3. Held-out total loss changed from 2.775 to 2.720, 2.674 and
  2.734 respectively. The predeclared largest-stable-rate rule selects **1e-3**;
  this is not a claim that it minimizes loss (it did not), is optimal, or improves
  playing strength. At 1e-3, same-replay training loss rose slightly, 3.986→4.005.
- Same-initial-network policy-only/8/32/128 search checks each completed one
  development setup pair with mechanical integrity. Wall times including command
  startup were 9.9/24.9/66.9/253.1 seconds. Scores against that network's
  policy-only play were 0.5/0.5/1.0/1.0; one pair is not strength evidence and did
  not change the planned eight-simulation pilot budget.
- The untrained eight-simulation network completed four games each against
  uniform MCTS, random, heuristic and policy-only baselines, scoring
  0.50/0.50/0.25/0.50 with mechanical integrity throughout. These are sanity
  checks, not a trained-model strength result.
- Raw screen artifacts: `artifacts/az/r5-pilot-20261010/screens/`. The fresh
  four-cycle pilot launched at 23:25 PDT on 2026-10-10 under
  `artifacts/az/r5-pilot-20261010/run/`, with source pinned to
  `8acfac55596a478fa16b9dd06e8425b5e6cad121`. Actor and learner use MPS, batch 32
  SGD at 1e-3, eight simulations, 16 concurrent actor games, 128-shard window,
  setup/search seed bases 6,000,000/7,000,000, and previous/frozen-anchor evaluation
  every second cycle. Bootstrap shards and probe weights are not admitted.
  First two learner stages completed after 64 fresh games (4,358 positions),
  with 100 updates each and actual learner throughput of 82/76 sampled positions
  per second. Sampled examples per newly generated position were 1.33/1.64;
  this denominator is new actor positions, not the growing replay window.
  Within-cycle held-out diagnostic loss fell 3.104→2.897 and 2.940→2.828;
  these changing mixed-replay samples are not a fixed-set strength curve.
  Completion verified 2026-10-11: runner exited 0 with completed_cycle=4;
  128 self-play games produced 8,946 positions, and all 400 updates completed
  (12,800 sampled examples). All 48 arena games completed naturally, with zero
  invalid actions, duplicate states, tripwires or infrastructure failures.
  The final suite remains untouched.

Pilot strength results (each comparison: eight development pairs, 16 games;
score counts a draw as one half):

| Comparison | W–D–L | Score | Paired 95% interval | Mean VP margin |
| --- | --- | --- | --- | --- |
| Cycle 2 vs cycle 1 | 4–0–12 | 25.0% | 0–73.0% | −9.0 |
| Cycle 4 vs cycle 3 | 9–0–7 | 56.25% | 8.2–100% | +3.0 |
| Cycle 4 vs cycle 1 | 4–1–11 | 28.125% | 0–76.1% | −7.5 |

No demonstrated strength improvement. The anchor comparison is directionally
concerning, but its conservative paired interval includes parity; do not call
this a confirmed regression or pool dependent comparisons. Cycle-4 diagnostic
loss also rose (training 3.635→3.824; held-out 2.800→2.912). In the anchor arena,
cycle 4 made 12 round-one building actions versus 27 for cycle 1, including one
versus six dwelling builds. These are action counts, not final building stocks
or proof of the cause of the score difference.

Next experiments, before increasing model size or launching a long run:

1. Freeze these checkpoints. Expand cycle-4 versus cycle-1 development evaluation
   to all 28 existing setup pairs (56 games total; the first eight pairs overlap
   this pilot, so do not count them as independent new evidence). Inspect paired
   openings, passing, building actions, and policy/search targets in lost games.
2. Revisit search on trained snapshots at 8/32/128 simulations on the same
   development cases, reporting strength and wall time separately. Inspect root
   visit coverage and target concentration; low-budget search is a hypothesis,
   not an established explanation. Keep network and optimizer fixed.
3. Compare downstream learning at 8 versus 32 simulations from matched fresh
   initializations, with at least three independent seeds per setting. First
   match generated games and learner updates, then separately compare equal
   total-time budgets. Use fixed anchors and a common evaluation search budget;
   preregister budgets/cadence before launch. These are new runs, not mutation
   of this completed run's immutable configuration.
4. Only after repeatable development gains, select a candidate and preregister
   an adequately sized untouched final comparison. Keep the final suite unused
   during diagnosis; neither a larger CNN nor a promotion gate is justified yet.

Follow-up protocol (2026-10-11, before comparison results):

Read-only adversarial diagnosis of the pilot: cycle4 made no round-one building
or upgrade in 11/16 anchor-arena games versus 5/16 for the anchor. Both weights
showed Alchemist conversion churn; in seed910000006, candidate round-one
conversions alone had net −15VP/−3coins. Cycle4 self-play passed 20/64 times in
round1 with a legal building option (not proof passing was wrong). One example,
seed6000128/state0104734c52621f104507d70476d7b5db, had ten legal actions including
two building choices, eight actions tied at one root visit, then selected pass.
Among nonforced positions, visited actions averaged 4.76 versus 16.27 legal
actions in cycle4. Single-action targets rose from 0.81% in cycle1 to 8.34%
in cycle4 across different replay populations; this is not a controlled collapse
measurement. Arena traces lack visit vectors/priors/Q; replay has visits but
not priors, so these artifacts cannot establish policy-improvement KL or a
specific value-error cause. Value-owner/sign review found no concrete defect.

- Expand the frozen pilot cycle-4/cycle-1 comparison to all 28 development pairs
  at eight simulations each. Retain the original overlapping eight-pair result
  as a subset, not additional independent games.
- On that same frozen candidate and anchor, compare candidate search 8/32/128
  against anchor search fixed at eight on the first eight development pairs.
  The eight-search result already exists. This is an unequal-compute inference
  experiment, not evidence about the benefit of training with deeper search.
- Train paired fresh model seeds 11/22/33 at 8 versus 32 simulations. Each arm
  gets four cycles of 32 new games and 100 batch-32 SGD updates per cycle,
  main 10×128, learning rate 0.001, momentum 0.9, weight decay 0.0001, window128,
  MPS/FP32, one Torch thread, actor/inference batch16 and arena batch8.
  Within each pair, match initialization, setup/search/sampling seeds; across
  pairs, change all four: setup base=10,000,000+100,000×model seed, search
  base=20,000,000+100,000×model seed, learner base=1,000×model seed.
  No pilot weights or replay enter training. Alternate
  arm execution order by seed. Stop for nonfinite training or integrity failure.
  This matches games and updates, not positions or compute: report both actual
  positions/reuse and elapsed time. Keep latest-network adoption, without gating.
- Primary development comparison: final 32-search-trained versus corresponding
  8-search-trained weights at a common eight-simulation evaluation budget over
  all 28 pairs, separately for each training seed. Secondary: each final model
  versus frozen pilot cycle1 over the first eight pairs. Internal runner arena
  checks use only one pair against each run's own early anchor at cycle4;
  those unequal-search, different-anchor checks are integrity diagnostics only.
- These are exploratory multi-seed screens. Report each seed, uncertainty and
  time; do not pool reused setups/comparisons as independent evidence. Use the
  measured timings to specify the subsequent equal-total-time training budget,
  rather than pretending the matched-game comparison is compute-matched.
  No architecture, reward, replay-weighting or optimizer changes in these arms.
  Keep the confirmatory suite untouched.

Completed follow-up results (2026-10-11):

- Expanded frozen cycle4/cycle1 arena: 26 wins, 3 draws, 27 losses across all
  28 development pairs, score49.107%, paired95% interval23.441–74.773%, mean
  VP margin+0.929. All56 games completed naturally without integrity failures.
  Independent adversarial review verified that the first16 game records match
  the pilot exactly, including decisions/predictions. The shift from28.1% to
  49.1% reflects wider setup coverage, not changed weights or learning progress;
  there are28 pairs total, not36. No demonstrated improvement or regression.
  Round-one building action totals73vs78 also weaken a general early-building
  regression claim, although the specific conversion-churn traces remain valid.
  Artifact: `artifacts/az/r5-comparison-20261011/anchor-28/`.
- Trained candidate search32 versus anchor search8 finished 8–0–8 (50% score,
  paired95% interval1.98–98.02%, mean VP margin−3.625), all16 games valid, wall
  682.14seconds. Same eight setups/checkpoints as the prior8-budget28.125%
  result; this is an exploratory point-estimate increase, not a demonstrated
  search benefit or evidence about learning. Candidate round1building count
  remains12; Alchemist conversions remain frequent. Adversarial review verified
  unchanged arena/search/model code despite the runner-only source revision.
- Trained candidate search128 versus anchor search8 finished 6–0–10 (37.5%,
  paired95% interval0–85.52%, VP margin−5.5), all16 games valid, wall1323.62s.
  Adversarial review verified fixed checkpoints/cases and only the candidate
  budget change. Candidate round1building actions fell to4 (versus12 at32),
  while conversion churn decreased. Point estimates are not monotonic in
  search depth; eight pairs do not establish128 intrinsically worse or identify
  a specific value/prior defect. No case for scaling training to128 from this.
- Seed11 common-eight-simulation head-to-head: 32-trained beat8-trained39–0–17,
  score69.643%, paired95% interval43.977–95.309%, VP margin+4.268; all56 games
  valid. Adversarial review verified final checkpoint identities and budgets.
  This is one directional result, not a confirmed multi-seed learning gain.
  Serious behavioral limitation: the32-trained candidate made no build/upgrade
  actions across these56 games, while the8-trained baseline made only10.
  Relative success against another passive policy is not strong TM play.
  Both arms completed128 games/400 updates; run wall times including internal
  sanity checks were664.34s (8) and1595.26s (32), with7798 versus6957 positions.
- Seed22 common-eight-simulation head-to-head reversed the seed11 direction:
  32-trained scored22–1–33,40.179%, paired95% interval14.513–65.844%, VP margin
  −2.161. All56 games valid. Neither agent built or upgraded during these games;
  the32-trained policy almost exclusively passed after setup. Adversarial review
  checked special-action traces for hidden building effects as well.
- Secondary frozen-pilot-anchor scores, all at eight simulations and eight
  development pairs: seed11 8-trained31.25%,32-trained25%; seed22
  8-trained34.375%,32-trained31.25%. All intervals include50%. These reused
  setups and opponents are not independent replications or promotion tests.
- All six training arms completed:768 games,44,409 new positions,2,400 updates,
  76,800 sampled examples. All24 learner stages reported finite losses and
  throughput. Independent review verified matched initialization and seed streams
  within each pair, different initializations across seeds, and unchanged SGD
  settings. Every committed-stage journal and artifact SHA inventory verified.

  | Seed | Self-play simulations | Positions | Full-run seconds |
  | --- | ---: | ---: | ---: |
  | 11 | 8 | 7,798 | 664.34 |
  | 11 | 32 | 6,957 | 1,595.26 |
  | 22 | 8 | 7,163 | 589.43 |
  | 22 | 32 | 6,712 | 1,365.16 |
  | 33 | 8 | 8,262 | 677.07 |
  | 33 | 32 | 7,517 | 1,529.90 |

  Each arm used128 games,400 updates and12,800 samples. Full-run wall time
  includes runner overhead/internal evaluation, but excludes external arenas.
  The32-search arms cost2.26–2.40× as much; this is not an equal-time experiment.
- Seed33 common-eight-simulation head-to-head:16–5–35, score33.036%, paired95%
  interval7.370–58.701%, VP margin−4.000. All56 games valid. The32-trained agent
  made8 build/upgrade actions versus133 for8-trained (round1:5 versus12).
  Independent review verified checkpoint identities, budgets and special-action
  semantics. Secondary anchor scores were37.5% for8-trained (6–0–10, paired95%
  0–85.52%, margin−4.0625) and25% for32-trained (4–0–12, paired95%0–73.02%,
  margin−10.9375). All16 games in each secondary comparison valid.
- Queue completed successfully at03:55PDT. All12 external reports passed
  mechanical checks:352 naturally completed games, zero invalid actions,
  duplicate-state failures, tripwire hits or infrastructure failures. Six
  internal two-game integrity arenas also passed. No weights promoted.
- Primary result: deeper self-play search did not consistently improve learning
  at this small matched-game budget. Seed directions disagree; all three paired
  intervals include50%. Do not pool repeated development setups as independent
  evidence, or infer that deeper search can never help from this screen. The
  frequent non-building policies make scaling this recipe premature. No settings
  changed in response to results; the final suite remains untouched.

Next bounded diagnosis, before further scaling:

User constraint (2026-10-11): learning must remain purely self-play, from random
initial weights and normal legal game starts. No expert/replay demonstrations,
heuristic-generated training targets, curated training positions, opening
curriculum, building incentives, reward shaping, or special replay balancing to
steer behavior. Policy targets come from self-play MCTS visits; value targets
come from terminal game outcomes. Random-number seeds identify reproducible
independent runs; they do not seed the learner with strategic knowledge.
Heuristic opponents are evaluation-only and never supply training data. Diagnostic
probes below are read-only analysis, not an altered self-play training recipe.

1. Freeze a small development-only set of positions drawn from existing self-play.
   Record legal-action priors, visits, Q, and child values backed up using the
   actual decision owner (including reward/leech decisions). Compare policy-only,
   normal eight-simulation search, and identical-prior/seed search with learned
   nonterminal leaf values set to zero; retain exact terminal outcomes. A change
   in passing implicates learned values in the search behavior, not the cause of
   their errors. If passing persists, a direct online leaf-value explanation is
   weaker; historical value errors could still have trained poor policy priors.
2. Distinguish optimizer drift from replay-distribution effects before choosing
   the next training ablation. Seed22 eligible replay WDL means were−0.070/−0.103
   for8/32, while arena mean predictions were+0.216/−0.315. On317 observed
   nonterminal passes by the32-trained model, mean pre-pass value was−0.321 and
   next-owner value backed to the original owner was+0.282. Passing also changes
   state/resources/rounds, so this is not an isolated owner-flip experiment or a
   confirmed sign bug. Uniform decision labels need not average zero; arena
   outcomes also come from a different policy distribution. Existing value-loss
   deterioration warrants a bounded fixed-replay optimizer-stability check.
3. Retain uniform eligible-position replay sampling. The previously proposed
   game/decision-owner-balanced sampler is withdrawn from the next-run plan.
   Use diagnostics to identify implementation defects or numerical instability;
   any next experiment must preserve the pure self-play targets and objective.
   Building activity and conversion churn are descriptive checks, not targets
   or criteria for selectively admitting training games. Keep evaluation data
   out of replay and the confirmatory suite untouched.
4. Defer larger networks, longer runs and equal-time search allocation until
   these diagnostics identify a defensible intervention. The measured search
   cost ratio can then set a separate equal-total-time comparison; the current
   matched-game result cannot answer that question.

Diagnostic findings and controlled next experiment (2026-10-14):

- New bounded diagnostic tools replay entire content-addressed self-play games,
  verifying every state hash, legal action list, chosen transition and terminal
  state. Probe positions are the first per round admitting pass and ordinary
  build/upgrade in the first content-hash shard of seed22/33 search32 cycle4.
  Reports: `artifacts/az/r5-diagnosis-20261014/`. These selected positions are
  not a representative strength sample or training curriculum.
- On the same six seed22 positions, cycle3→4 normal8 pass visits rose35/48→47/48,
  while zero-nonterminal-value8 stayed25/48 and raw priors barely changed. At
  round3 normal128 pass visits rose22→117; zero128 stayed20. A synthetic test
  independently demonstrates how constant decision-owner value bias can favor
  switching owners even with correct backup signs. No sign-code defect found.
- Values are not the sole cause: policy-only and zero-value8 pass6/6 for seed22;
  seed33 normal8 passes1/6 versus zero8's5/6. Initial random fingerprints match
  the original runs; policy-only initially passed0/6 for both seeds versus6/6
  and5/6 after learning. This establishes learned ranking changes, not that the
  initial moves were good. Construction includes power-spade build variants:
  seed33 round1 has63/239 construction choices with28.02% prior mass, slightly
  above uniform26.36%; a blanket construction-suppression claim is false.
- Fixed-replay, momentum-preserving100-update LR-only continuation uses identical
  sample streams, weights, targets and diagnostic populations. Heldout value MSE:

  | Seed | Before | LR0.001 | LR0.0003 | LR0.0001 |
  | --- | ---: | ---: | ---: | ---: |
  | 22 | 0.5695 | 0.6097 | 0.6001 | 0.5118 |
  | 33 | 0.7831 | 0.7690 | 0.6405 | 0.7103 |

  Lower rates reduce prediction oscillation in both seeds; no rate is proven
  optimal. Heldout states share self-play games and are not a strength test.
  Policy target KL remains about0.39–0.41 nats and barely changes in this stage.
  Train/eval outputs agree; no normalization-mode or target-sign defect found.
  MPS reruns do not reproduce original parameters bitwise despite matching
  code/replay/sampler/runtime; CPU segmented-continuation regression is exact.
  Two identical uninterrupted100-update MPS continuations also diverged:
  maximum parameter difference0.001076 and fixed128-state heldout MSE difference
  0.00348. This establishes numerical non-repeatability under these controls,
  not a replay-order or segmented-resume defect. Exact CPU recovery remains
  separately tested; do not promise bitwise MPS reproducibility.
- With ordinary self-play root noise (epsilon0.25, total concentration10),
  seed22 normal32 assigned113/192 aggregate visits to passes versus42/192 with
  learned values zeroed on the six selected states. This remains an isolated
  value intervention with identical noisy priors, not an improvement recipe;
  self-play normally samples visits whereas displayed diagnostic choices are
  greedy. Exact terminal outcomes remain active in every variant.
- Before new strength results: run fresh seeds22/33 with32 self-play simulations,
  four cycles×32 games×100 batch32 updates, main10×128, SGD momentum0.9,
  weight decay0.0001, changing only learning rate0.001→0.0001 versus their old
  matched32-search arms. Keep original setup/search/learner/model seeds, MPSFP32,
  one thread, actor/inference batch16, wait1ms, concurrent roots, replaywindow128,
  cache8 and uniform eligible-position sampling. No old replay or weights enter
  these new lineages. Root noise and temperature retain ordinary self-play defaults.
  This is an exploratory validation of a conservative rate, not a tuned optimum.
  Seeds/rate were selected after these diagnostics; controls are historical and
  MPS updates are not bitwise repeatable. This is not a confirmatory or perfectly
  paired deterministic causal comparison.
  Actual seed22 cycle1 replay matches all32 historical games exactly except
  source-commit metadata (1,646 decisions). That metadata changes content hashes
  and sorted replay order: only1/32 games retains its old slot, so the same learner
  RNG seed does not give identical minibatches. Uniform sampling and canonical
  diagnostic holdout membership are unchanged. This further limits causal claims.
- Primary: each final new model versus its corresponding old32-trained model,
  both at8 evaluation simulations, all28 development pairs. Secondary: each
  new final versus frozen pilotcycle1, both8 simulations, first8 pairs. Report
  seeds separately, uncertainty and behavioral changes; do not pool shared
  setups. Internal cycle4 anchor one-pair check is integrity-only. No adaptive
  changes mid-run, no promotion gate, no final-suite use. Preserve immutable
  outputs under `artifacts/az/r5-lr-validation-20261014/`; stop on integrity or
  nonfinite failures. More search, larger networks and shaped rewards are not
  part of this experiment.

Fresh learning-rate validation results (2026-10-14, complete):

- Source `602621f6425bcb3ebcdbe38e8afe677fdeb284f8`; immutable queue, manifests,
  checkpoints and reports in `artifacts/az/r5-lr-validation-20261014/`.
  Queue exited0; both four-cycle runs and all four external arenas completed.
  Total256 fresh games,15,016 raw positions,800 updates/25,600 sampled examples;
  all eight learner stages finite. All144 external games and four internal
  integrity-check games completed naturally, without reported integrity failures.
- Primary lower-LR candidate versus same-seed historical32-search final, common8
  evaluation simulations; intervals are the conservative95% paired-case bounds:

  | Seed | W/D/L | Score | 95% interval | Mean VP margin |
  | --- | ---: | ---: | ---: | ---: |
  | 22 | 27/0/29 | 48.21% | 22.55–73.88% | −1.32 |
  | 33 | 24/1/31 | 43.75% | 18.08–69.42% | +0.30 |

  Do not pool seeds/shared setups as independent evidence. Neither establishes
  improvement or equivalence. Secondary versus frozen pilot: both5/0/11,
  score31.25%, interval0–79.27%; margins−5.06/−8.50VP respectively.
- Sparse construction persisted despite changed activity: seed22 candidate built/upgraded zero times
  across72 external games, despite increased non-pass activity. Seed33 primary
  candidate made six construction actions versus baseline four, and zero
  conversions versus baseline1,002; reducing churn did not establish strength.
  Counts include power-build variants, not just ordinary build/upgrade kinds.
- Final-cycle own diagnostic value MSE: seed22 heldout .95075→.92708;
  seed33 training1.03307→.93563 but heldout .96679→.99833. Policy losses were
  essentially flat/slightly worse. These within-run diagnostics are not independent
  game generalization tests. Old/new diagnostic subsets differ with replay order.
- Run wall times1517.36/1646.74s exclude external arenas; matched games/updates
  are not matched compute or positions. Both seeds' first32 self-play games match
  historical controls exactly except provenance (1646/1804 decisions); replay
  ordering caveats above still apply. No old weights/replay or probe outputs were
  adopted. Final suite remained untouched. Adversarial training and evaluation
  reviews found no integrity blocker and rejected claiming a repaired engine.

Conclusion and next bounded experiment:

- Verified search sensitivity: learned near-tied policy rankings and decision-owner
  value estimates materially alter pass visits. Correct backup can amplify
  inconsistent values. These probes do not establish optimal actions or the
  original cause of learning failure. A self-reinforcing weak policy/value loop
  is the leading interpretation, not a universal pass-bias explanation; effects
  differ across seeds. No concrete
  rules, backup-sign, target-sign or train/eval-mode defect was found by these checks.
- Lower LR stabilizes the fixed-replay continuation probes but is not sufficient
  from random initialization over400 updates. Distinguish insufficient optimizer
  work from weak search targets before increasing architecture or scaling the run.
- Authorized three-arm screen (2026-10-14): fresh seeds44 then55, each with three
  four-cycle arms,32 games/cycle: A32 simulations/100 updates; B128 simulations/
  100 updates; C32 simulations/400 updates. Keep main10×128, batch32, SGD.0001,
  momentum.9, weight decay.0001, replaywindow128 and all exploration/runtime
  settings fixed. Same pinned source and four seed streams across each seed's arms.
  The conservative LR is a controlled reference, not an established optimum.
- Freeze B−A and C−A contrasts for both seeds before observing outcomes. Evaluate
  cycle4 at common8 simulations on28 development pairs, plus8-pair frozen-pilot
  checks. Report each seed separately with uncertainty, wall time, positions and
  reuse;384 games/2400 updates per seed, not an equal-compute experiment. This
  screens main effects but cannot identify a search×optimizer interaction.
  Defer the additional eight-cycle horizon arm for simplicity. If no arm gains
  strength, inspect target quality and policy learnability before broader scaling;
  if one does, replicate the frozen contrast before adopting the recipe.
  No heuristic rewards, expert data, curated starts or replay balancing.

Execution protocol frozen before launch:

- Output root `artifacts/az/r5-budget-comparison-20261014/`. Run sequentially on
  the Mac GPU: seed44 A/B/C and its evaluations, then seed55 A/B/C and its
  evaluations. Each arm starts from its seed's random initialization and has an
  isolated replay/checkpoint directory. Other arms' data never enter training.
- Per seed s, model seed=s, setup base=10,000,000+100,000s, search base=
  20,000,000+100,000s, learner base=1,000s. Keep MPS FP32, Torch threads1,
  actor/inference batch16, inference wait1ms, concurrent roots, replay cache8,
  replaywindow128, uniform eligible-position sampling, and ordinary root noise/
  temperature. Pin one full source revision and tool identities for every arm.
- Primary contrasts B4−A4 and C4−A4 each use28 development setup pairs with both
  players at8 simulations. All three final models also face frozen pilotcycle1
  on8 development pairs at common8. Internal evaluation is one anchor pair at
  cycle4 for integrity only. No B−C contrast, adaptive settings or promotion gate.
- Total six runs:768 fresh training games,4,800 updates/153,600 sampled examples;
  ten external arenas/320 games, plus12 internal integrity-check games. Report
  both seed results even if they disagree. A gain is only suggestive when both
  seeds move positively on the same contrast; intervals crossing50% remain
  inconclusive, and anchor/behavior diagnostics are not substitutes for WDL.
  Reused development setups and selected budgets preclude confirmatory claims.
- Stop on nonfinite training, invalid/incomplete games or provenance failures.
  Keep completed artifacts immutable; resume only the unfinished stage under its
  existing manifest. Never rerun the entire new-mode queue over existing outputs,
  duplicate live jobs, or rebuild modified engine sources into running tools.
  Final suite remains untouched. No production engine changes are required.

### Three-arm comparison results — 2026-10-15

Source pinned to `f1cf73f4983857bcc640aec6c1ba571b53b2864c`; evidence under
`artifacts/az/r5-budget-comparison-20261014/`. All six training runs completed
four cycles:768 games,51,419 raw positions,4,800 updates/153,600 sampled examples.
Adversarial audits verified checkpoint hashes/parent lineage, finite learner
stages, matched manifests and natural internal games. A/C first-cycle replay
files were byte-identical in the same sorted order within each seed; subsequent
on-policy replay naturally diverged. All starts were random initialization and
ordinary legal setups, with no external data or special seeding.

Primary results, common8 evaluation simulations,28 paired development setups
(56 games) per row. Score counts draws as half; intervals are paired95%
finite-sample Hoeffding bounds, not independent-game binomial intervals.

| Seed | Contrast | W/D/L | Score | Paired95% interval | Mean VP margin |
| --- | --- | --- | --- | --- | --- |
| 44 | B128/100 vs A32/100 | 30/0/26 | 53.57% | 27.91–79.24% | +1.23 |
| 44 | C32/400 vs A32/100 | 36/1/19 | 65.18% | 39.51–90.84% | +1.73 |
| 55 | B128/100 vs A32/100 | 24/1/31 | 43.75% | 18.08–69.42% | −0.82 |
| 55 | C32/400 vs A32/100 | 26/0/30 | 46.43% | 20.76–72.09% | −2.25 |

Neither intervention's positive seed44 direction replicated in seed55. Every
interval includes50%: this demonstrates neither improvement nor equivalence or
harm. Do not pool shared setups or infer C beats B without a direct contrast.
All224 primary games completed naturally with zero invalid actions, duplicate
states, tripwire hits or infrastructure failures. Final suite remains untouched.

Secondary checks against frozen pilotcycle1, common8 simulations,8 pairs/16 games
per row (not additional independent evidence for the primary contrasts):

| Seed/arm | W/D/L | Score | Paired95% interval | Mean VP margin |
| --- | --- | --- | --- | --- |
| 44 A | 5/0/11 | 31.25% | 0–79.27% | −8.44 |
| 44 B | 5/0/11 | 31.25% | 0–79.27% | −9.50 |
| 44 C | 4/0/12 | 25.00% | 0–73.02% | −4.38 |
| 55 A | 6/1/9 | 40.63% | 0–88.64% | +1.31 |
| 55 B | 7/0/9 | 43.75% | 0–91.77% | +3.56 |
| 55 C | 7/0/9 | 43.75% | 0–91.77% | −2.13 |

All ten external arenas finished:320 natural games with zero failure counters,
plus12 valid internal integrity games (at each arm's training search budget,
not common8). Recovery queue exited0 and published `queue-complete.txt`.
No arm established an advantage over the pilot. Positive mean VP does not
override the WDL objective or wide uncertainty.

Compute accounting (seconds, rounded): each run has128 games; A/B400 updates
and12,800 sampled examples; C1,600 updates and51,200 examples.

| Seed/arm | Raw positions | Committed actor | Committed learner | Full run wall |
| --- | --- | --- | --- | --- |
| 44 A | 7,813 | 1,481.62 | 176.75 | 1,719.93 |
| 44 B | 8,685 | 4,977.69 | 176.66 | unavailable: interrupted |
| 44 C | 8,035 | 1,343.54 | 619.40 | 2,061.91 |
| 55 A | 8,540 | 1,260.39 | 168.33 | 1,596.66 |
| 55 B | 9,367 | 4,592.91 | 173.82 | 5,225.52 |
| 55 C | 8,979 | 1,329.49 | 647.63 | 2,102.88 |

This is equal-game, not equal-position or equal-compute. Committed actor+learner
cost for B/A was3.11x/3.34x across seeds44/55; C/A was1.18x/1.38x. Components
exclude startup/internal arenas; full run wall excludes external arenas. Seed44B
was interrupted by the user pause: unpublished cycle4 replay was quarantined and
retried under its immutable manifest. Its1,259.338 seconds of wasted work are
interruption overhead, not additional training games. Recovery wall1,421.21s
overlaps committed stage compute and is neither full-run cost nor additive to it.
Recovery provenance is in `RECOVERY.md`, `recovery-20261015.sh` and
`recovery-20261015/manifest.json`. Never rerun either new-mode queue over outputs.

Behavioral evidence is seed-sensitive, not a universal no-building failure.
Against A44, B built/upgraded54 times versus0 and C6 versus0; C made427
conversions. Against A55, B constructed253 versus317; C175 versus270 and made584
conversions versus35. Counts include power/bonus builds and implicit faction
build/upgrade specials, not just explicit build flags. More construction is not
a strength metric. Policy losses remained nearly flat, while value-fit changes
varied by seed; neither observation proves one root cause or predicts strength.

Next justified experiment: a small diagnostic-only fit of existing self-play
MCTS visit targets, comparing the current joint policy/value objective with
policy-only fitting. Freeze identical initialization, replay, legal masks and
sample stream and canonical heldout split; predeclare a bounded update budget
and training-KL reduction criterion for fitting. Measure train/heldout policy
KL, target entropy, top-visit agreement and joint-branch value MSE. If only the
policy-only branch fits, investigate objective interference; if neither fits,
investigate optimization/representation/scoring; if both fit, investigate target
quality and action selection. These are hypotheses, not definitive localization.
Discard diagnostic weights: no actor adoption, expert labels, curated starts,
replay balancing or production objective change. Do not launch another large
training queue or enlarge the network before this cheaper test. R5 remains open.

Policy-fitting diagnostic protocol (2026-10-16, before results): use only the
first-cycle A-arm self-play shards from the completed seed44 and55 comparisons,
validated against source `f1cf73f4983857bcc640aec6c1ba571b53b2864c`. For each seed,
uniformly select256 eligible training positions without replacement and128
canonical heldout positions (or all available if fewer). Do not select positions
by action, outcome, phase or faction. Freeze references and identical minibatch
streams; both branches start from the same main10x128 random initialization with
model seed44/55. Use training-subset seed731, heldout-subset seed732 and
minibatch-stream seed732, batch32,400 SGD updates at0.0001,
momentum0.9 and weight decay0.0001. Only the inclusion of value loss differs.
Record initial/100/400-step train/heldout KL, target entropy, top-visit agreement,
value MSE and logit spread, separately for all and multi-action positions.
Also measure policy/value shared-trunk gradient norms and their cosine on the
same first minibatch before/after fitting; this tests scale/interference without
changing updates. A small gradient alone does not establish a causal bug.
The preregistered fitting criterion is at least50% reduction of mean training
KL on multi-action positions; heldout improvement is reported separately and is
not required for this memorization check. Initial KL below1e-6 is uninformative.
Finite gradients, matching sample/init identities and intact replay hashes are
required. This small-subset overfit test deliberately differs from full-window
training and cannot establish playing strength or universal capacity limits.
Use MPS FP32 sequentially, one Torch thread. No weights are saved or adopted;
all targets remain original self-play visit distributions and terminal WDL.
Further probes require explicit recorded hypotheses after these results, not
silent changes to this protocol. Adversarial implementation/result reviews are
required. Output root: `artifacts/az/r5-policy-fit-20261016/`.

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
auxiliary outcome/score predictions, explicit connectivity features, or self-play
distillation. Each needs an isolated hypothesis and equal-compute
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
bazel test //internal/... //cmd/... //:az_recovery_test //:az_real_recovery_test //:az_loop_smoke_test //:az_scaling_curves_smoke_test --test_output=errors --nocache_test_results --test_timeout=900
```

Add new fixtures and recovery/evaluation checks to appropriate Bazel targets.
Run broader backend regressions for shared-rules changes. If a frontend change
becomes necessary, the repository additionally requires `//:client_build_test`.
Passing this command alone is not independent rules verification; apply the
adjudicated-fixture and evidence criteria above.

The initial R0–R2 implementation and clarification follow-up are complete for
the documented competition contract. R3 is complete at its documented confidence
level. R4 passed its documented recovery/evaluation gates. The bounded R5
fresh-lineage screen, pilot and three-seed matched-search follow-up completed
mechanically, without a consistent strength improvement. Bounded policy/value
diagnostics and two-seed fresh learning-rate validation are complete, without
demonstrated improvement. The two-seed search-versus-update screen is complete
without a replicated gain. The diagnostic-only policy-target fit above is next;
it has not been launched. R5 remains open. Scaling is not yet justified.
