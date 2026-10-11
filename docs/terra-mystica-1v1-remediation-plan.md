# Trusted 1v1 Terra Mystica engine: implementation plan

Status: active corrective implementation, 2026-10-10. No milestone below is
complete merely because an equivalent component exists in the current engine.
This is the active corrective plan; it supersedes the earlier plan's completion
claims and experiment ordering. Evidence and rationale are in the
[first-principles review](terra-mystica-1v1-first-principles-review-20260919.md).

## Implementation checkpoint — 2026-10-10

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
- R5 — starting the bounded fresh-lineage protocol below. R6 onward not started.

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

Follow-up results so far (2026-10-11):

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
- The sequential queue remains active: trained candidate search32 is running;
  search128, six fresh training arms and common-budget external evaluations
  follow without changing the preregistered settings. Final suite untouched.

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
fresh-lineage screen and pilot completed mechanically, without demonstrated
strength improvement. Next are the frozen-checkpoint diagnosis and controlled
search/learning experiments above; R5 remains open.
