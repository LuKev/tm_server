# Terra Mystica 1v1 AlphaZero Engine Plan

## Goal

Build a 1v1 Terra Mystica engine that can train from self-play using an AlphaZero-style loop:

1. Encode a live game position.
2. Enumerate legal actions from the current position.
3. Run PUCT/MCTS using a policy/value evaluator.
4. Store `(state, legal_action_mask, visit_policy, chosen_action, outcome)` records.
5. Train a policy/value model from self-play records.
6. Use the trained model inside MCTS for move suggestions or bot play.

## Current Implementation

The implementation intentionally reuses the existing game engine rather than creating a parallel rules model:

- `server/internal/az/actions`: stable action IDs, broad candidate generation, and live-engine legality filtering.
- `server/internal/az/env`: AlphaZero position wrapper, fixed 1v1 scenario sampling, board-aware observation encoding, and outcome scoring.
- `server/internal/az/model`: evaluator interface plus a bootstrap heuristic evaluator.
- `server/internal/az/mcts`: PUCT search with root noise, visit-count policy output, lazy child expansion, and cached legal actions on positions.
- `server/internal/az/selfplay`: JSONL self-play generator.
- `server/internal/az/train`: table-model trainer from self-play JSONL.
- `server/internal/az/arena`: candidate-vs-incumbent evaluation gate.
- `server/internal/az/dataset`: neural-ready sparse dataset and action-vocabulary exporter.
- `server/cmd/az_selfplay`: Bazel-backed self-play CLI.
- `server/cmd/az_train`: Bazel-backed trainer for a lightweight table policy/value model from JSONL records.
- `server/cmd/az_export`: exports samples, action vocabulary, and dataset manifest for a neural trainer.
- `server/cmd/az_eval`: evaluates a candidate table/HTTP evaluator against a baseline table/HTTP/heuristic evaluator.
- `server/cmd/az_loop`: iterative generate/train/arena/promote loop.
- `server/cmd/az_replay_seeds`: exports replay-derived generated snapshots as self-play seed JSONL from a single replay file or replay fixture directory, with optional seed coverage summary output.
- `server/cmd/az_train_torch`: minimal PyTorch policy/value trainer for exported samples.
- `server/cmd/az_infer_torch`: HTTP policy/value inference server for trained PyTorch checkpoints.
- `POST /api/ai/suggest`: MCTS-backed ranked move suggestions from a live game, pasted snapshot, or built-in scenario.
- `POST /api/ai/execute`: confirmation-gated bot execution for a live game, using the same legal action surface and `game.Manager.ExecuteActionWithMeta` path as normal play.

The stronger-engine scaling runbook lives in `docs/terra-mystica-stronger-engine-scaling.md`.

## Important Design Constraint

Every legal move exposed to MCTS is validated by executing a `game.Action` against a cloned state through `game.Manager.ExecuteActionWithMeta`. This is slower than a handwritten action mask, but it prevents self-play from learning illegal Terra Mystica. MCTS does not apply every legal child during expansion anymore; it materializes a child state only when search selects that edge.

The earlier round-6 beam-pruned solver failed the trust test because delayed-payoff actions could be discarded before deeper search saw them. This implementation does not beam-prune legal moves by immediate heuristic value. MCTS keeps all legal children reachable and only biases exploration through policy priors and visit counts.

## Scenarios

The built-in deterministic scenarios are:

- `base_nomads_witches`
- `base_giants_mermaids`
- `base_engineers_auren`
- `base_halflings_alchemists`
- `random_base` samples among those pairings and randomly swaps seats
- `randomized_base` samples base-game faction pairs, seat order, starting dwelling anchors, scoring tiles, and bonus cards
- `snapshots:/path/to/snapshot_seeds.jsonl` samples replay-derived generated snapshots
- `training_mix` mixes deterministic starts with randomized starts for higher-volume training

Each starts with 2 players, round-1 action phase, confirm-actions disabled, and two dwellings per player. Deterministic scenarios use fixed map hexes; randomized scenarios sample anchors and round assets. This avoids auction/setup noise while proving the action/action-mask/search/self-play loop end to end.

## Training Record Shape

`az_selfplay` emits JSONL records with:

- `state`: compact JSON metadata for the position
- `scenario`: scenario name
- `encoding`: numeric observation vector
- `observationSchema`: currently `tm_az_board_v1`
- `observationShape`: `[global feature count, board hex count, per-hex feature count]`
- `featureNames`: emitted on the first ply of each episode and carried into the dataset manifest
- `legalActions`: action IDs legal in that position
- `policy`: MCTS visit-count probabilities by action ID
- `actionId`: action selected for self-play
- `outcome`: final normalized value from that record player's perspective
- `terminal` / `truncated`: whether the game ended naturally or hit the configured ply cap

The table model trains directly against this shape. `az_export` converts the same JSONL into sparse neural samples with `legalActionIndices`, `policyTargets`, a sorted action vocabulary, and a dataset manifest. The PyTorch trainer reads the flat vector size from the manifest and stores the observation schema/shape in the checkpoint; `az_infer_torch` exposes that metadata on `GET /healthz`.

Replay-derived snapshot seed JSONL may include optional metadata such as source path, replay action index, round, phase, player count, root faction, and factions present. The self-play loader treats those fields as audit metadata; the live position still comes from parsing the `snapshot` text. `az_replay_seeds` can also filter generated seeds by phase, root faction, player count, and round bounds while reporting skipped candidates by reason in the summary file.

## Bot Execution API

`POST /api/ai/execute` accepts:

- `gameId`: required live game ID
- `rootPlayerId`: optional search perspective; defaults to the current decision player
- `actionId`: optional action ID to execute; defaults to the top MCTS result
- `confirm`: must be `true` to mutate the game
- `expectedRevision`: optional optimistic concurrency guard
- `actionRequestId`: optional idempotency key passed to `ExecuteActionWithMeta`
- `search`: optional MCTS config

When `confirm` is omitted or false, the endpoint returns the selected action and ranked search result without mutating the game. When confirmed, it rechecks the selected action against the current live legal action surface, then executes through the normal game manager path.

## Commands

Use Bazel only:

```bash
cd server
bazel test //internal/az/... --test_output=errors
bazel build //cmd/az_selfplay:az_selfplay //cmd/az_train:az_train //cmd/az_loop:az_loop //cmd/az_export:az_export //cmd/az_eval:az_eval //cmd/az_replay_seeds:az_replay_seeds
bazel run //cmd/az_selfplay:az_selfplay -- -list_scenarios
bazel run //cmd/az_selfplay:az_selfplay -- -scenario=training_mix -episodes=1 -max_plies=20 -sims=16 -batch_size=4 -metrics=/tmp/tm_az_metrics.json
```

For any frontend changes:

```bash
cd server
bazel test //:client_build_test --test_output=errors
```

Train and use a table evaluator:

```bash
cd server
bazel run //cmd/az_selfplay:az_selfplay -- -scenario=random_base -episodes=10 -max_plies=120 -sims=32 -output=/tmp/tm_az_selfplay.jsonl
bazel run //cmd/az_train:az_train -- -input=/tmp/tm_az_selfplay.jsonl -output=/tmp/tm_az_model.json
TM_AZ_MODEL_PATH=/tmp/tm_az_model.json bazel run //cmd/server:server
```

Run the iterative table-model self-play loop:

```bash
cd server
bazel run //cmd/az_loop:az_loop -- \
  -work_dir=/tmp/tm_az_runs \
  -iterations=3 \
  -episodes=20 \
  -shards=4 \
  -scenario=training_mix \
  -sims=16 \
  -batch_size=4 \
  -max_plies=120 \
  -arena_games=8 \
  -promote_win_rate=0.55 \
  -promote_min_games=8 \
  -promote_ci95_lower_bound=0.45
```

`az_loop` writes both the legacy top-level `promoted` boolean and a structured `promotion` decision with the gate policy, win rate, confidence interval, and blocking reasons.

Export neural-ready samples:

```bash
cd server
bazel run //cmd/az_export:az_export -- \
  -input=/tmp/tm_az_runs/iter_0001/selfplay.jsonl \
  -samples=/tmp/tm_az_runs/iter_0001/samples.jsonl \
  -vocab=/tmp/tm_az_runs/iter_0001/action_vocab.json \
  -manifest=/tmp/tm_az_runs/iter_0001/dataset_manifest.json
```

Train the baseline PyTorch network when `torch` is installed in the Bazel Python runtime:

```bash
cd server
bazel run //cmd/az_train_torch:az_train_torch -- \
  --samples=/tmp/tm_az_runs/iter_0001/samples.jsonl \
  --manifest=/tmp/tm_az_runs/iter_0001/dataset_manifest.json \
  --vocab=/tmp/tm_az_runs/iter_0001/action_vocab.json \
  --output=/tmp/tm_az_policy_value.pt \
  --architecture=hex
```

Serve the PyTorch checkpoint and use it from Go MCTS:

```bash
cd server
bazel run //cmd/az_infer_torch:az_infer_torch -- \
  --checkpoint=/tmp/tm_az_policy_value.pt \
  --host=127.0.0.1 \
  --port=9097
```

In another shell:

```bash
cd server
bazel run //cmd/az_selfplay:az_selfplay -- \
  -scenario=random_base \
  -episodes=10 \
  -max_plies=120 \
  -sims=16 \
  -batch_size=8 \
  -model_url=http://127.0.0.1:9097/evaluate \
  -output=/tmp/tm_az_neural_selfplay.jsonl
```

The live server API can also use the HTTP evaluator:

```bash
cd server
TM_AZ_MODEL_URL=http://127.0.0.1:9097/evaluate bazel run //cmd/server:server
```

`az_loop` can use a served neural model as the incumbent before any promoted Go table model exists:

```bash
cd server
bazel run //cmd/az_loop:az_loop -- \
  -work_dir=/tmp/tm_az_runs \
  -iterations=1 \
  -episodes=20 \
  -scenario=random_base \
  -base_model_url=http://127.0.0.1:9097/evaluate
```

## Open Scaling Work

- Add setup/auction self-play once the legal action surface is cheap enough for those phases.
- Replace HTTP inference with an in-process runtime or ONNX path if inference latency becomes material.
- Expand replay-derived seed coverage across more factions, round phases, and endgame states.
