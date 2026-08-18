# 1v1 Terra Mystica AlphaZero Plan

Status: implemented through Milestone 5's local scaling foundations; the
ordered large-checkpoint experiment campaign is in progress.

Implementation status (updated 2026-08-17): Milestones 0 through 4 are
complete. The base-rules contract, serial/batch-shaped PUCT, versioned relative
state/action encoder, masked hex-CNN, continual-latest self-play/learner loop,
and paired evaluator are implemented. The debug and main network shapes are
approximately 660k and 4.4M parameters. State records, trajectories, and
checkpoints independently gate rules, state, action, feature-order,
normalization, coordinate, model, and engine schemas. Every completed
milestone passed correctness and simplicity adversarial review plus fresh
uncached Bazel suites. Milestone 5's local scaling foundations are complete:
the large model and accelerator path, measured simulator optimizations,
bounded actors/replay, and a versioned scaling-curve harness are implemented.
Shared local inference is now implemented and measured: independent root
searches feed one bounded MPS inference broker without changing seeded
trajectories. The remaining work is running larger experiments that narrow
strength uncertainty and determine when additional actor or inference-process
parallelism is warranted.

## 1. Objective

Build a strong 1v1 Terra Mystica agent that learns from self-play and uses the
authoritative Go rules engine for every simulated transition. The first target
is the base map, base factions, and the normal two-player setup. Expansion
factions, alternate maps, UI play, and production deployment come only after
the base engine learns reliably and passes correctness and strength gates.

The design follows the core algorithm in Silver et al., [*Mastering Chess and
Shogi by Self-Play with a General Reinforcement Learning
Algorithm*](https://arxiv.org/abs/1712.01815):

1. A single neural network evaluates a position as `(policy, value) = f_theta(s)`.
2. PUCT Monte Carlo tree search uses that policy and value to produce an
   improved root visit distribution `pi`.
3. Both players select moves from `pi` during self-play.
4. The terminal rules produce `z = -1, 0, +1` for loss, draw, or win.
5. Training minimizes `(z - v)^2 - pi^T log(p) + c ||theta||^2`.
6. New self-play uses the latest network. Strength evaluation runs alongside
   training; it does not gate whether the learner may continue.

Terra Mystica is materially harder to represent than chess or Go: its hex map
is irregular, its action space is typed and hierarchical, and one player may
make several consecutive decisions while reactions can temporarily belong to
the opponent. A prior Terra Mystica study identifies the same broad
representation and action-space challenges, but this plan does not adopt that
implementation: [Perez, *Mastering Terra Mystica: Applying Self-Play to
Multi-agent Cooperative Board Games*](https://arxiv.org/abs/2102.10540).

## 2. Non-negotiable design rules

- The live rules engine is the only source of legality, transitions,
  termination, and scoring. Do not build a second Terra Mystica rules model.
- The agent must choose actual legal decisions. Do not grant resources, bypass
  costs, silently auto-convert, or use replay tolerances during search.
- The primary reward is the game result. Final VP, VP margin, opening choices,
  and faction statistics are evaluation metrics or auxiliary predictions, not
  shaped rewards in the first system.
- A tree edge does not imply that the player changed. Backups are keyed by the
  identity of the decision owner, not by blindly negating value every ply.
- Illegal actions are masked by the rules engine. The network is never expected
  to learn legality from examples.
- No map symmetry augmentation is allowed unless the transformation is proven
  to preserve terrain, setup, turn order, and action semantics.
- UX-only actions such as confirmation and undo are not game decisions. Search
  uses an AI rules profile with explicit game choices and no UI convenience
  automation.
- Training artifacts are immutable and versioned by engine commit, ruleset,
  state schema, action schema, model ID, and random seed.

## 3. Target architecture

```text
authoritative Go rules engine
        |
        v
pure 1v1 search adapter -----> legal complete actions + canonical state hash
        |                                      |
        v                                      v
self-play actors <---- batched PUCT MCTS ---- neural inference service
        |                                      ^
        v                                      |
immutable trajectory shards -----> learner/checkpoints
        |                                      |
        +------------> evaluator <-------------+
                             |
                  Elo, VP, matchup and quality reports
```

Keep the search adapter, MCTS, and actor orchestration in Go so they can call
the current engine directly. Use PyTorch for the network and learner. A small,
versioned binary protocol should carry encoded states and ragged legal-action
features to a long-lived inference process. HTTP/JSON may exist as a debugging
adapter, but it is not the scaled data path.

## 4. Milestone 0: freeze the game contract

Before writing MCTS, define a minimal search-facing interface:

```go
type Position interface {
    Clone() Position
    DecisionPlayer() PlayerID
    LegalActions() []SearchAction
    Apply(SearchAction) error
    IsTerminal() bool
    Outcome(PlayerID) float32
    CanonicalHash() Hash128
}
```

The adapter should operate on `game.GameState`, use existing action validation
and execution, use `CloneForUndo` initially, and use `CalculateFinalScoring` at
the terminal state. It must not depend on `game.Manager`, WebSockets, player
names, game IDs, timers, confirmations, or replay modes.

### Legal-action oracle

Implement two enumerators:

- A deliberately slow test oracle that exhausts every bounded parameter domain
  for every base-game action constructor, retaining candidates whose normal
  `Validate` call succeeds.
- An optimized production enumerator with cheap structural gates.

On sampled and replay-derived states, the two action sets must match exactly.
This makes optimization measurable without trusting handcrafted pruning.

Each `SearchAction` has a stable semantic encoding defined by the rules, not by
which actions happened to appear in a dataset. It contains an action kind plus
typed arguments such as source hex, target hex, building type, cult track,
tile/card/faction ID, response, and bounded amount. Serialization must
round-trip to the corresponding `game.Action`.

### Correctness exit gate

- Every emitted action validates and applies on an identical clone.
- The optimized action set equals the slow oracle across every base action
  kind, setup phase, pending-decision kind, faction, and round represented in
  the test corpus.
- Seeded games are deterministic and naturally reach final scoring. A ply cap
  is a failing tripwire, not a draw label for training data.
- Canonical hashes are identical for semantically identical states and change
  for every search-relevant field, including pending decisions.
- Snapshot or action-log replay reproduces every sampled state and terminal
  score.
- Bazel tests and fuzz/property tests run this contract continuously.

## 5. Milestone 1: implement and prove PUCT MCTS

Each node stores legal actions and `(P, N, W, Q)` per edge. Selection uses a
PUCT score of the form:

```text
Q(s,a) + c_puct * P(s,a) * sqrt(sum_b N(s,b)) / (1 + N(s,a))
```

At a new leaf, the network supplies priors and a value. At a terminal state,
the rules engine supplies the exact result. The root policy target is derived
from visit counts.

Critical TM rule: store backed-up utility by player identity. If the same
player makes another free action, setup choice, or follow-up decision, the
value sign does not change. If control passes to the opponent, the zero-sum
value changes perspective. Test this first on small synthetic games containing
same-player edges and reaction edges, then on constructed TM states.

Start with a uniform-policy, zero-value evaluator. Add:

- deterministic seeds and reproducible search traces;
- root-only Dirichlet exploration noise scaled to observed legal branching;
- visit-proportional self-play selection and greedy evaluation selection;
- an evaluation cache by canonical state hash;
- virtual loss and leaf batching only after the serial implementation passes;
- subtree reuse only when the new root hash exactly matches the applied child.

Do not initially share visit statistics through a transposition DAG. Equivalent
conversion orders can share neural evaluations through the cache without
introducing graph-backup bugs.

### Search exit gate

- Toy-game tests recover known optimal policies and values.
- Serial and batched searches agree within deterministic tolerances.
- More simulations improve or preserve strength for the same network over a
  statistically meaningful paired evaluation.
- Search never mutates its parent state, emits an invalid action, or backs up a
  value from the wrong player's perspective.

## 6. Milestone 2: design the state and policy representation

### State encoder

Encode the position from `DecisionPlayer()`'s perspective. For the base-map
baseline, place the hex map in a fixed masked axial grid and provide spatial
feature planes for terrain, building type, relative owner, town membership,
bridge endpoints, and valid-map cells.

Encode the remaining state as a fixed global vector: both relative players'
factions, VP, resources and power bowls, shipping/digging, structure supply,
cult positions and keys, passed state, bonus/favor/town tiles, abilities,
round-scoring tiles, available shared pieces, claimed power actions, turn/pass
order, setup subphase, pending decisions, map, setup mode, and ruleset version.
Project that vector into conditioning channels that are broadcast over the
spatial grid and into FiLM scale/bias values for the residual trunk.

The position must be Markov-complete. Do not add eight historical boards merely
because AlphaZero used history for chess repetition rules. Add compact action
history only if an ablation shows useful information absent from the current
state.

### Variable legal-action policy

Avoid a dataset-built global vocabulary. Score the complete legal action list:

1. Encode each candidate from its action kind and typed arguments.
2. Gather referenced board/global embeddings for its hexes, tile, faction, or
   track.
3. Produce one logit per legal action with a shared action-scoring head.
4. Apply softmax only over the legal candidate list.

This is still the paper's `p(a|s)`, but it cannot lose actions merely because a
small replay buffer never observed them. The network API must support ragged
action lists and preserve a stable action schema across checkpoints.

### Network and loss

Begin with a residual CNN, a legal-action policy scorer, and a scalar `tanh`
value head. This follows AlphaZero's board-plane inductive bias and makes
batched leaf inference substantially cheaper than a Transformer. Train exactly
the AlphaZero policy/value objective first. Optional auxiliary heads for final
VP, VP margin, round, or resource totals may be added later behind ablations;
they must not change `z`.

The concrete v0 shape is:

```text
spatial planes -----------> masked axial grid ----------------------+
                                                                    |
relative player/global ---> broadcast planes + FiLM conditioning --+--> residual hex-CNN trunk
                                                                               |
                                                                               +--> global pooling + MLP --> tanh(v)
                                                                               |
legal action features + gathered source/target cells --------------------------+--> shared policy MLP --> K legal logits
```

- Hex convolution reads the six axial neighbors; invalid grid cells are masked
  after every block. A conventional 3x3 convolution is an acceptable first
  implementation if its mask and coordinate convention are tested, but a
  six-neighbor kernel is the intended baseline.
- Bridge ownership/endpoints are spatial planes. Long-range effects are learned
  through stacked residual blocks and global conditioning.
- Player and shared features are categorical embeddings or normalized bounded
  scalars. Player identity is relative: decision owner and opponent.
- Each legal action embedding contains the action kind and typed non-spatial
  arguments, then gathers trunk features at any referenced source/target hexes.
- The policy MLP scores each complete legal action independently while sharing
  weights across action kinds. A masked softmax over the `K` candidates yields
  `p(a|s)`.
- The value head combines masked global average pooling with the encoded global
  vector, then uses a two-layer MLP to emit one scalar in `[-1, 1]`.
- Batches have a fixed spatial tensor and global vector; only legal actions are
  ragged/padded. The checkpoint manifest fixes every feature, coordinate,
  embedding, and action schema version.

Use two initial sizes:

| Model | Trunk | Channels | Expected purpose |
|---|---:|---:|---|
| Debug | 4 residual blocks | 64 | Pipeline correctness and fast ablations; roughly 0.5-1M parameters |
| Main v0 | 10 residual blocks | 128 | First serious self-play run; roughly 3-5M parameters |
| Scale candidate | 20 residual blocks | 256 | Only after the main model is demonstrably capacity-bound; roughly 20-30M parameters |

Parameter counts are estimates until the exact feature/action encoders are
implemented. The graph/Transformer design is an experimental later alternative
for variable custom maps, not the baseline. Increase CNN width/depth, change the
trunk family, or add auxiliary VP/margin heads only when held-out loss, search
Elo, inference throughput, and simulation-scaling curves identify model
capacity or representation as the bottleneck.

### Representation exit gate

- Encode/decode and action-schema golden tests cover every base action kind.
- Permuting internal player IDs while preserving actor/opponent roles leaves
  encoded semantics unchanged.
- A tiny dataset can be intentionally overfit to near-zero policy error and
  accurate value targets.
- Checkpoint loading rejects schema or ruleset mismatches loudly.

## 7. Milestone 3: build the self-play and learner loop

For every decision, persist the canonical pre-action state, complete legal
action list, root visit counts, chosen action, decision player, and model ID.
When the game ends, attach the result from each recorded player's perspective,
plus final VP and VP margin for reporting.

Store trajectories as immutable compressed shards. Keep the raw canonical
record separate from materialized tensors so a new encoder can re-materialize
old compatible games. Each shard needs a manifest and content hash.

The learner samples a recent sliding window, balanced enough that common
factions or high-branching action types do not crowd out the rest. It publishes
a new checkpoint atomically after training steps. Actors adopt the latest
checkpoint between games, never halfway through a game.

Follow AlphaZero's continual-latest-network loop. Do not block training on a
55% candidate-versus-champion promotion match; that was the AlphaGo Zero
procedure, whereas AlphaZero continuously updated one network. Keep immutable
snapshots so regressions remain visible and deployment can still select a
stable champion.

### Loop exit gate

- A random initialization can produce valid complete games and train without
  NaNs, dropped actions, truncations, or schema drift.
- Policy cross-entropy to MCTS targets falls on a fixed diagnostic batch.
- Value calibration improves on held-out self-play games.
- A trained network plus search beats its random/uniform ancestor in paired
  games with both seats and identical setup seeds.

### Implemented evidence

- Go actors play authoritative games against a long-lived Python inference
  process. Independent roots share bounded neural batches, and completed games
  publish immutable, hash-chained trajectory shards.
- The learner validates model, checkpoint, engine, seat, result, and schema
  provenance; trains the exact policy/value objective with non-finite guards;
  and atomically publishes immutable checkpoints plus a latest pointer.
- The scheduler covers all 168 legal ordered distinct-home base-faction
  matchups. The two-cycle smoke test proves that actors adopt the latest
  checkpoint between game batches and that the evaluator retains both the
  uniform and previous-checkpoint anchors.
- A balanced 168-game complete replay run covered every ordered matchup without
  truncation. A continuation pass reduced fixed-diagnostic policy loss from
  1.8049 to 1.6336 and value loss from 0.000655 to 0.000033. On the independent
  paired holdout in Milestone 4, weighted Brier score improved from the uniform
  baseline's 0.25 to 0.1769.

## 8. Milestone 4: evaluate strength without corrupting training

Run evaluation as an independent service. Every comparison uses paired setup
seeds and swaps seats/factions. Report uncertainty rather than calling a winner
from a partial run.

Required metrics:

- Elo or another explicitly defined rating across immutable checkpoints;
- win/draw/loss with confidence intervals against recent and long-term anchors;
- average final VP, winning VP, losing VP, and VP margin;
- results by ordered faction matchup and first-player seat;
- setup and opening action distributions, including round-1 building types;
- value calibration/Brier score and policy entropy by phase;
- invalid actions, natural completions, duplicate states, and tripwire hits.

Keep a fixed holdout suite of legal base-map setups that self-play actors never
sample for training diagnostics. Historical human games may later be used as
an external behavior/position benchmark, but tabula-rasa training should not
silently mix them into the replay window.

### Evaluation exit evidence

The format-3 paired arena report completed all 16 games across eight reserved
setup seeds, with each candidate playing both seats against the exact uniform
anchor. It recorded a 9-0-7 W/D/L, point score 0.5625, and smoothed Elo +41.06,
with 16 natural completions and zero invalid actions, duplicate states, or
tripwire hits. The report includes final/winning/losing VP, margin, ordered
matchup and seat splits, per-agent semantic setup/opening distributions,
round-1 building types, and Brier/entropy by phase.

This is evidence that the end-to-end system learns and can beat its uniform
ancestor at the observed point estimate, not yet evidence of reliable playing
strength. The paired 95% interval is wide at [0.082, 1.0], average VP margin is
-0.4375, and setup-phase Brier is worse than uniform even though rounds 1-6
improve. Milestone 5 must increase the evaluation sample and diagnose these
metrics while scaling compute. The development report used a working-tree
engine label; regenerate it against an immutable commit before archiving it as
durable experimental evidence.

## 9. Milestone 5: scale in the right order

Current status (2026-08-10): the local implementation passed correctness and
simplicity review. Self-play now reports search/evaluator work, inference queue
and service time, median/tail latency, batch/cache/trajectory metrics, and
phase-round-faction splits. Bazel benchmarks cover naturally reached setup,
rounds 1/3/6, a pending reaction, naturally terminal scoring, and distinct
batched roots. The first local probe found legal enumeration to be the dominant
simulator cost: roughly 10 ms/4.8 MB in an early Witches state, 15 ms/6.9 MB in
a natural round-3 Engineers state, and 51 ms/32 MB in a natural round-6 Fakirs
state. That evidence gates the next optimization to skip-action enumeration;
it does not justify copy-on-write, make/unmake, or distributed infrastructure.
The resulting range/resource prefilter reduced that same round-6 Fakirs case
to roughly 10.6 ms/6.1 MB and 194 candidates, from 1,013 candidates, while the
independent exhaustive oracle and full engine/AZ contract suites remained
green. This is deliberately a candidate-domain reduction, not a second rules
implementation: the authoritative skip validator still decides legality.
The next measured cost was the roughly 6.1 ms/0.75 MB canonical record and
hash. Each immutable position now computes that record once, shares it safely
with clones, and invalidates it on `Apply`; the neural inference transport
reuses the same bytes already hashed by MCTS. A cached hash is about 69 ns with
zero allocation, while an 8-root/8-simulation search probe improved from about
1.33 s to 1.18 s. The cold canonicalization benchmark remains explicit so this
cache cannot hide future encoder-state growth.

Actor volume is now independent of its memory/batching bound. `--games` is the
total workload, while `--games-per-batch` limits concurrent roots and
trajectories retained at once; each completed batch is immediately published
as immutable content-addressed shards. The measured local default is 32
concurrent games feeding a single bounded inference broker with a 32-position,
1 ms flush policy. The cycle runner can override these independently with
`TM_AZ_ACTOR_GAMES_PER_BATCH`, `TM_AZ_INFERENCE_MAX_BATCH`,
`TM_AZ_INFERENCE_MAX_WAIT`, and `TM_AZ_ROOT_SEARCH_MODE`. Metrics record both
configured limits and observed broker fill/queueing. The final summary retains
counters rather than every shard reference, and inference percentiles are
explicitly recent-window metrics over at most 4,096 observations; whole-run
counts and durations remain exact. The two-cycle smoke uses a one-game bound,
verifies all four shards, and still verifies latest-checkpoint adoption plus
the uniform/recent-anchor reports.

The scheduling choice was measured on the full 20-block, 256-channel large
hex-CNN on Apple MPS with eight simulations per decision. Every point processed
the same 32 setup/search seeds. The lockstep reference with eight concurrent
games and no shared broker reached 432 games/hour; shared inference reached 894
games/hour with 8 concurrent roots, 1,041 with 16, and 1,079 with 32. All four variants
produced the exact same 32 content-addressed trajectory files. This is a
deterministic random-model throughput experiment, not playing-strength
evidence. Lockstep remains available as the reference mode; the 32-root default
is justified by the 2.50x local throughput improvement and should be
re-profiled when model shape, simulation count, or hardware changes. Exact
configuration and output metrics are retained in the
[local scaling evidence](terra-mystica-1v1-alphazero-local-scaling-evidence.md).

Replay loading is likewise bounded. The learner validates immutable shards one
at a time, then retains only path/matchup/ply metadata for the selected window.
An LRU capped by `--replay-cache-games` (runner override
`TM_AZ_REPLAY_CACHE_GAMES`, default eight) holds decompressed raw games, and
only sampled plies are encoded into network features. Allocation-free dry
traversals still validate every encoder-consumed state/action value before a
shard enters the window, so corrupt unsampled plies cannot fail training
nondeterministically. Training batches and the fixed diagnostic set remain
separate explicit bounds. Checkpoint provenance records replay games/plies,
cache loads/hits, validation time, and learner examples/second.
The continual runner exposes the replay horizon as
`TM_AZ_REPLAY_WINDOW_SHARDS` (default 10,000) and records it in the immutable
run manifest, so the learner-intensity experiment can vary replay horizon
without silently changing cache residency or code.

The model/accelerator step also passed correctness and simplicity review. The
large preset is an AlphaZero-style 20-block, 256-channel residual hex-CNN with
29,114,114 parameters, retaining the exact versioned 38-plane spatial input and
ragged legal-action head. Checkpoints self-describe this shape; explicit model,
device, and thread mismatches fail the inference handshake. CPU, CUDA, and MPS
share one FP32 learner/inference implementation. On the development Apple GPU,
large-model inference measured about 111, 255, and 286 positions/second at
batches 1, 8, and 32, versus 80 and 108 positions/second on one CPU thread at
batches 1 and 8. A lifecycle test trains the large model for one accelerator
step, publishes it immutably, recovers its shape automatically, and performs
canonical ragged inference. Benchmark, inference, and checkpoint artifacts
record device, dtype, effective thread count, model shape, and seed/provenance.
Training-throughput probes must also name and execute the selected learner
optimizer; an AdamW timing is not evidence for the primary SGD path.
The continual runner defaults to this `large` profile; its bounded smoke test
selects `debug` explicitly so functional CI does not masquerade as a
production-scale training run.

The fixed `tm-v0-paired-v2` holdout now contains 28 reserved setups, covers all
14 base factions, and includes both faction orders for every selected matchup.
Full-suite and ordered-prefix identities are derived from the exact cases and
rejected when their contents do not match the name. Evaluation format 5 records
separate candidate and baseline simulation budgets, the bounded games per
batch, and infrastructure failures, so a search curve changes one agent's
compute without silently changing both sides or losing a failed attempt.
`//:run_az_scaling_curves` produces large-model batch measurements and immutable
paired arena reports for the default 1/8/32/128 candidate budgets against the
same checkpoint at a fixed one-simulation budget. The production runner
requires that checkpoint, verifies that its embedded shape matches the requested
model, and records its model ID and SHA in every batch artifact; random mode is
explicitly smoke-only. Model identity and search-role identity are separate in
format 5, so this comparison isolates search compute instead of changing the
network or both agents' budgets. Its Bazel smoke test executes an actual model
probe and complete paired arena case. That smoke proves the path, not playing
strength.

The ordered scaling experiment sequence is:

1. Run a two-cycle large-model accelerator pilot. Prove actor-to-learner latest
   adoption, collect complete per-stage JSON, and choose learner work in
   examples per available replay ply rather than a fixed step count.
2. Sweep inference and training batch sizes against that exact checkpoint.
   Choose the smallest batch near the measured throughput knee; do not infer a
   production batch size from synthetic random weights or a different shape.
3. Sweep bounded local actor concurrency at a fixed network and search budget.
   Measure games/hour, GPU batch fill, queue/service latency, decisions/game,
   and simulator work so concurrency is not selected from GPU utilization
   alone.
4. Run the same-checkpoint paired search curve at candidate budgets
   1/8/32/128 against a fixed one-simulation anchor on the reserved holdout.
   Retain score, final VP, margin, Brier calibration, openings, and paired
   uncertainty; the comparison changes search compute, not model identity.
5. Sweep learner intensity (examples per replay ply) and replay-window size.
   Reject settings that improve sampled training loss while worsening held-out
   Brier, causing value saturation, or destabilizing policy entropy.
6. Sweep self-play simulation budgets with data volume and learner intensity
   fixed. Select the lowest budget whose downstream checkpoint strength and
   calibration are not materially worse, rather than assuming the paper's 800
   simulations transfers directly to Terra Mystica.
7. Run balanced data-scale checkpoints at 168, 672, and 2,688 games, preserving
   the 168 ordered legal base-faction matchups and separating setup seeds from
   search/model RNG seeds. Compare learning curves per generated game and per
   wall-clock hour.
8. Evaluate surviving configurations on the full 28-case paired holdout and
   larger reserved extensions, then against uniform, the previous checkpoint,
   and the best frozen checkpoint. A promotion conclusion needs both seats,
   absolute VP, calibration, natural completion, and uncertainty.
9. Decide the next architecture from the measurements. Add one shared GPU
   inference service plus more CPU actors only if local actors cannot fill the
   selected batch without unacceptable queue latency. Add distributed storage
   or actors only if generation, rather than learner or evaluation, is the
   measured bottleneck.

The first 8-game/two-cycle large-MPS pilot completed the mechanics but was
intentionally too small for strength claims. It exposed two immediate controls:
25 learner steps meant 800 sampled examples after only 318 first-cycle plies,
and the resulting checkpoint saturated its value output and roughly doubled
decisions/game in the next self-play cycle. Learner work is therefore now
expressible as examples per replay ply. The same pilot also showed serial arena
evaluation dominating wall time, so paired games now advance in lockstep and
batch neural/search requests without changing each game's seed or action
sequence. Both changes require deterministic-equivalence and end-to-end smoke
gates before the pilot is repeated.

A repeat at only 0.25 requested examples per replay ply still saturated the
large value head after three AdamW steps at `1e-3`, and the next self-play
cycle rose from 39.75 to 88.75 decisions/game. A same-initialization/same-replay
diagnostic localized this to optimizer scale: AdamW through `1e-4` saturated,
while `3e-5` reduced the diagnostic value loss from 1.146 to 0.958. The primary
large-model learner now follows the paper's optimizer family—momentum SGD with
L2 weight decay—while AdamW remains an explicit ablation. A follow-up SGD sweep
selected `1e-4` as the conservative pilot default: it improved the same
three-step diagnostic from total/value 3.534/1.146 to 3.333/0.941 without
saturation. Optimizer, learning rate, applicable momentum, and weight decay are
checkpoint and run-manifest provenance.
Optimizer buffers are part of every learner checkpoint and must be restored on
the next cycle; otherwise cycle boundaries reset SGD momentum or AdamW moments
and change the algorithm. An optimizer/config change requires an explicit
recorded reset, and uninterrupted versus checkpoint-resumed training must be
deterministically equivalent on a fixed data stream.

The mechanically corrected two-cycle SGD pilot kept the large value head finite
and restored the optimizer state across cycles. At eight simulations, the two
actor cycles generated 39.75 and 44.63 decisions/game at roughly 429 and 390
games/hour;
three and then six SGD steps moved diagnostic value loss from 1.146 to 0.941
and from 0.996 to 1.198. The latter regression means the next learner-intensity
experiment still needs held-out calibration rather than choosing from training
loss. All 24 tiny-pilot arena games completed naturally with zero invalid,
duplicate, tripwire, or infrastructure failures, but the two-case reports are
mechanics evidence only.
Its manifest was later found to contain a mistyped, nonexistent engine commit;
rerun it from a verified revision before treating its checkpoint or metrics as
durable evidence.

Exploratory checkpoint-bound large-model probes put the local MPS inference
knee at batch 32: batches 1/4/8/16/32/64 measured about
110/221/255/273/287/290 positions/second. Batch 64 adds only about 1.2%
throughput while doubling batch latency, so 32 is the provisional inference
throughput setting. Momentum-SGD training batches 1/2/4/8/16/32 measured about
32/50/69/85/95/102 examples/second, but 32 was the largest point and is not yet
an established training knee. A fixed eight-game actor sweep at eight
simulations preserved exactly 68.625 decisions/game at every concurrency.
Bounds 1/2/4/8 produced roughly 210/229/231/248 games/hour with mean inference
batches 1.00/1.73/2.90/4.66 and p95 latencies 13.6/20.5/38.6/59.3 ms. Retain
eight concurrent games only as the fastest observed exploratory point; the
small 7.4% advantage over four needs a counterbalanced repeat before becoming
a default. The checkpoint used by these probes was later found to contain a
mistyped, nonexistent engine revision in its upstream training provenance, so
these numbers cannot support strength or durable scaling conclusions. Repeat
the pilot and selected throughput points from a verifiable Git revision before
the search curve. Distribution remains unjustified.

That replacement run is now complete and recorded in the
[local scaling evidence](terra-mystica-1v1-alphazero-local-scaling-evidence.md).
Its real engine revision matches all 16 shards and two checkpoints; seven
stages exited cleanly and all 12 pilot arena games completed naturally with no
integrity failures. The final checkpoint is therefore admissible for controlled
throughput and same-model search experiments, though the two-case arenas still
make no strength claim. Clean checkpoint-bound probes establish batch 32 as the
smallest local near-ceiling point: 99.0% of batch-64 inference throughput and
98.8% of batch-128 SGD throughput. A forward/reverse fixed-game actor sweep
produced the same semantic trajectories at every bound and averaged
208/230/231/247 games/hour for 1/2/4/8 concurrent games, with order spreads at
or below 1.05%. This older lockstep result is retained as historical evidence.
It was superseded on 2026-08-17 by the independent-root shared-inference
experiment above: use 32 concurrent games locally, while retaining lockstep as
the reference mode. Distributed inference remains gated on evidence that one
local broker is saturated.

The full same-checkpoint search curve is also complete in that evidence report.
Across 56 games per point, candidate budgets 1/8/32/128 against a fixed
one-simulation anchor produced observed score 0.500/0.625/0.732/0.696, VP
margin 0/+4.43/+8.50/+7.38, and wall time 347/753/2,108/6,068 seconds. Every
game completed naturally and every integrity counter remained zero. The paired
intervals are wide and do not establish statistical superiority, but 128 is
observationally dominated by 32 at almost three times its wall time. Treat 32
as the measured local search-strength peak, reject 128 for this checkpoint, and
carry eight forward as the efficient self-play candidate pending the separate
downstream self-play-budget experiment.

The first fixed-replay learner-intensity screen is also complete. Requested
examples-per-replay-ply ratios 0.10/0.25/0.50/1.00 and newest-only replay
windows of eight and four games all worsened paired held-out Brier calibration
against their common parent, including the one-step window-four arm. The small
score and VP samples were mixed and have wide uncertainty, so no learner
setting was selected. In accordance with step 7's balanced-data requirement,
the next experiment first grows the fixed replay from 16 games to all 168
ordered legal base-faction matchups at eight simulations, then repeats the
learner screen rather than tuning further on an under-diverse sample.

Every campaign run owns a new directory and immutable configuration manifest;
the continual runner refuses a nonempty directory rather than mixing shards or
checkpoint lineage. Each actor, learner, and arena stage atomically retains its
stdout, stderr, and exit status even when the stage fails. Learner admission
also requires every trajectory's engine commit to equal the requested training
commit. These are experiment-integrity constraints, not deployment machinery.

### First optimize the simulator

Profile legal enumeration, cloning, action application, hashing, and terminal
scoring before buying actor capacity. The correctness baseline may deep-clone
each state. Optimize in this order:

1. remove avoidable allocations and cache immutable map/rules data;
2. cache legal actions and neural results by canonical hash within a search;
3. batch leaf evaluations across many concurrent games;
4. replace deep copy with copy-on-write or make/unmake only after equivalence
   tests prove the optimized transition matches the baseline.

### Then scale actor/learner throughput

Use this capacity equation rather than guessing cluster size:

```text
required neural evaluations / second
  = games / second
  * decisions / game
  * simulations / decision
  * leaf evaluations / simulation
```

Measure each term by phase and faction. Scale only the current bottleneck.

| Stage | Shape | Purpose | Advance when |
|---|---|---|---|
| A | One process, serial MCTS, uniform network | Correctness | Complete deterministic games and all contract gates pass |
| B | One workstation, one GPU, many Go actors | Learnability | Search beats uniform; GPU batches are stable; strength rises with compute |
| C | One GPU inference server plus distributed CPU actors | Throughput | GPU utilization and batch fill are high; storage and learner keep up |
| D | Multiple inference replicas and one learner | Sustained training | Checkpoint freshness, data lag, and faction coverage stay within targets |
| E | Multiple learners only if demonstrated necessary | Large experiments | Learner, not actors/inference, is the measured bottleneck |

At each stage record games/hour, decisions/game, simulations/second,
evaluations/second, median and tail inference latency, batch size, cache hit
rate, bytes/game, CPU/GPU utilization, learner examples/second, replay age, and
cost per thousand complete games.

The paper's 800 simulations per move is a reference point, not an initial TM
constant. Establish scaling curves at several budgets and spend compute where
additional simulations still produce measurable Elo gain.

## 10. Implementation sequence

1. Search adapter, slow legal oracle, canonical actions, and hash.
2. Random-play full-game harness and correctness/property tests.
3. Serial PUCT with uniform evaluator and identity-correct backups.
4. Versioned masked axial-grid/global encoder and legal-action encoder.
5. PyTorch policy/value network, tiny-set overfit test, and checkpoint schema.
6. Batched inference protocol and local self-play actor pool.
7. Immutable trajectory format, learner, and continual checkpoint publishing.
8. Paired evaluator, rating history, VP/faction reports, and dashboards.
9. Profiling-driven simulator optimization and cross-game inference batching.
10. Distributed actors, durable object storage, recovery, and only then a
    user-facing engine service.

## 11. Definition of a credible first engine

The first engine is ready for scaling only when it can play every supported
base 1v1 setup to natural completion, enumerate the same actions as the slow
oracle, reproduce games from seeds, train the exact policy/value objective,
beat a uniform-search ancestor, improve with a larger search budget, and report
strength with paired confidence intervals and absolute VP metrics. A working
HTTP endpoint or a single favorable arena result is not sufficient.
