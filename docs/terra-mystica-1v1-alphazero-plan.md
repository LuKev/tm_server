# 1v1 Terra Mystica AlphaZero Plan

Status: clean-slate design after removal of the previous AlphaZero implementation.

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

## 9. Milestone 5: scale in the right order

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
