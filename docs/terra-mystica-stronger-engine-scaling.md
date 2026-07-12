# Scaling The 1v1 AlphaZero Engine

## Objective

The current 1v1 engine proves the full AlphaZero loop: live-engine legal actions, MCTS, self-play JSONL, dataset export, torch training, HTTP inference, arena gating, and bot execution. The next goal is to make that loop strong enough to improve across repeated runs.

Strength comes from four properties:

1. High-volume self-play with measured throughput.
2. Diverse positions that cover factions, round assets, board shapes, and late-game states.
3. A model architecture that uses board structure instead of treating every feature as unrelated scalar input.
4. Evaluation that is reproducible and statistically harder to fool.

The canonical recovery-to-production procedure is
`docs/terra-mystica-neural-production-runbook.md`. In particular, the current
website is not considered neural until its backend runs with
`TM_AZ_REQUIRE_NEURAL=true` and `/api/ai/status` reports `mode=neural`.

## Next Strength Gameplan

The next goal is a stronger promoted checkpoint, not just a bigger pile of rows. The current evidence says the loop is working, but iteration time is dominated by neural MCTS self-play and arena games. Export and h512 training are not the limiting phases yet. Modal compute is useful, but primarily for parallel self-play and arena shards; use GPU only when training larger models or larger exported buffers.

Track round-1 Temple/Sanctuary and Stronghold rates in `docs/terra-mystica-ai-r1-build-rates.md` for each retained model. These rates are an early-game proxy, especially for factions such as Giants and Swarmlings where stronger play should usually build an early Temple or Stronghold.

### 2026-06-29 Local Fast-Lane Result

The retained `/tmp` checkpoints from earlier runs were missing, so the pre-Modal pass rebuilt a local baseline from heuristic MCTS and trained one neural h512 candidate. These are recovery experiments, not a continuation of the former promoted lineage:

- Iter0 heuristic self-play: `/tmp/tm_az_local_fastlane_20260629/iter0/selfplay.jsonl`
  - `168/168` terminal games, `9512` records, `sims=8`, `workers=4`, `recordsPerSecond=3.65`.
  - Exported `9512` samples and trained `/tmp/tm_az_local_fastlane_20260629/iter0/tm_az_h512_iter0.pt`.
- Iter1 neural self-play: `/tmp/tm_az_local_fastlane_20260629/iter1/selfplay.jsonl`
  - `84/84` terminal games, `4853` records, `sims=8`, `workers=4`, HTTP torch evaluator, `recordsPerSecond=15.66`.
  - This is about `4.3x` faster than the heuristic baseline generation path for this run.
  - Combined iter0+iter1 buffer exported `14365` samples and trained `/tmp/tm_az_local_fastlane_20260629/iter1/tm_az_h512_iter1_candidate.pt`.
- Smoke arena: `/tmp/tm_az_local_fastlane_20260629/eval/arena_iter1_vs_iter0_84.json`
  - Candidate scored `46-38`, win rate `54.8%`, 95% CI `[44.1%, 65.4%]`.
  - Did not promote because the lower confidence bound was below the `45%` threshold.
- Ordered arena: `/tmp/tm_az_local_fastlane_20260629/eval/arena_iter1_vs_iter0_168.json`
  - Candidate scored `84-80-4`, win rate `51.2%`, 95% CI `[43.6%, 58.7%]`, `0` truncations.
  - Did not promote. Keep the candidate as an experiment artifact, not as the incumbent.

The run proves the optimized local path is much faster once a neural evaluator exists, but the R1 Temple/Stronghold proxy is still weak. Giants and Swarmlings remain far below the expected mature-engine range, so the next pre-Modal work should bias toward better search targets and targeted early-game data before larger model experiments.

### 1. Freeze A Trustworthy Baseline

Do not rely on old `/tmp` checkpoint paths as durable incumbents. The previously documented promoted h512 paths were missing during the 2026-06-29 local pass, so the next promoted checkpoint should be copied to a durable project artifact location, object store, or Modal volume before it is treated as the incumbent. Keep the previous promoted model as a retained baseline whenever the artifact is actually available so regressions are visible.

Required baseline checks:

- `matrix:base_ordered`, `168` games minimum, `max_plies=500`, `0` truncations expected.
- `training_mix`, at least `200` games for a serious promotion gate, because randomized assets catch failures that a single ordered matrix pass can miss.
- A smaller targeted suite for known weak or historically fragile matchups, especially any matchup involving Mermaids or any future scenario with truncations.
- Promotion should require `-promote_min_games` and `-promote_ci95_lower_bound`, not only raw win rate.

### 2. Shorten Iterations Before Scaling Volume

Use a two-lane loop:

- Fast lane: local or small Modal run, `336-840` ordered-matrix games, `sims=8`, h512 transfer training at conservative LR, then an `84`-game ordered smoke arena. This catches bad data/code/model changes in hours.
- Promotion gate: do not promote from an `84`-game smoke result. Any candidate considered for promotion should get at least a `168`-game arena gate.
- Strength lane: Modal run, `10k-20k` ordered-matrix plus targeted games, optionally `sims=16` for part of the batch, h512 or h768 transfer training, then `336+` arena games split across `matrix:base_ordered` and `training_mix`.

Do not jump straight to very large self-play if the fast lane is flat. The 5k pass already proved volume helps when the data is clean and complete; the next gain should come from better data mix and higher-quality search targets.

### 3. Spend More Compute Where It Buys Strength

Use this priority order:

1. More complete neural MCTS self-play from the current incumbent.
2. Targeted oversampling of weak matchup/scenario buckets found by arena reports.
3. Higher search simulations on a smaller, high-quality batch.
4. Larger model/training experiments only after the current h512 loop still improves with more data.

Current rough sizing from local observations:

- The realistic local h512 neural self-play range is around `12-18` records/sec depending benchmark and scenario mix.
- Full ordered games usually produce roughly `60-100` training records each.
- `10k` games is roughly `0.6M-1.0M` rows. On one local machine that is plausibly an overnight-to-day run; on `16` Modal CPU workers it should become a short multi-hour run plus startup/upload overhead.
- `50k` games is the first scale where Modal compute is clearly justified if the `10k-20k` run promotes cleanly.

### 4. Modal Execution Shape

Use Modal as a sharded job runner, not as a new training architecture:

1. Build or install the Bazel server workspace in a Modal image.
2. Put the incumbent checkpoint and output directory on a Modal volume or object store.
3. For each self-play shard, start `az_infer_torch` locally in the container, then run `az_selfplay` with:
   - `-scenario=matrix:base_ordered` or a targeted comma-separated scenario suite
   - `-compact_records`
   - `-workers=4` unless benchmarking shows the instance prefers fewer workers
   - `-sims=8` for volume batches; `-sims=16` or `32` for smaller quality batches
   - `-batch_size=8`
   - `-global_batch_size=32`
   - `-model_url=http://127.0.0.1:<port>/evaluate`
4. Write one JSONL and one metrics JSON per shard.
5. Merge with `az_buffer`, export with `az_export`, and train with `az_train_torch --init_checkpoint --init_allow_action_mismatch`.
6. Run arena shards the same way, with candidate and incumbent inference servers local to each worker.

For cost control, run one calibration shard first and compute rows/sec, games/hour, truncation rate, and cost/game before launching the full batch.

### 5. Improve Data Quality Per Row

The next self-play buffer should be mixed intentionally:

- Keep all clean complete games from the current promoted lineage.
- Add fresh `matrix:base_ordered` games so every legal ordered base matchup is represented.
- Add targeted weak-matchup games from arena losses, not only from truncations.
- Add replay-derived snapshot seeds for round-2 through round-5 action positions once their coverage summary shows enough faction/round diversity.
- Cap older or weaker data with `az_buffer path@limit` so the buffer does not become dominated by stale bootstrap policy targets.

Preferred next buffer shape:

- `512k` current promoted-lineage rows.
- `600k-1M` fresh ordered-matrix rows.
- `100k-300k` targeted weak-matchup rows.
- Optional `100k-300k` replay-seed continuation rows after seed coverage is audited.

### 6. Training Experiments

Run training experiments as a small grid, not one-off guesses:

- h512 transfer, LR `1e-5`, `5-8` epochs.
- h512 transfer, LR `5e-6`, `5-8` epochs if the first run overfits or regresses.
- h768 or h1024 only after h512 still promotes from the larger buffer.
- Keep `--init_new_action_logit` conservative when action vocab grows.

The current hex model is probably good enough for the next scale step, but it is still a shallow mean/max pooled board encoder. If larger data stops helping, the next architecture work should be a real adjacency-aware board encoder or residual hex encoder before spending much more on raw self-play.

### 7. Promotion Gates

Use three gates:

1. Smoke gate: `84` games across selected scenarios, used only to reject obvious failures.
2. Ordered gate: `168` or `336` games on `matrix:base_ordered`, fixed seed, `max_plies=500`, `0` truncations.
3. Serious gate: `200-500` games across `training_mix` plus targeted weak buckets, with confidence lower bound enabled.

Promotion should require:

- raw win rate above incumbent,
- `95%` CI lower bound at or above the configured threshold,
- no material truncation regression,
- no scenario bucket collapse,
- average margin not obviously worse in the candidate's losing buckets.

### 8. Highest-Leverage Code Work

If we want faster iterations before or alongside Modal:

- Add first-class run manifests that capture checkpoint path, scenario mix, seeds, shard counts, command flags, git SHA, and artifact paths.
- Add arena bucket reports by matchup/scenario so targeted oversampling is automatic.
- Add a Modal runner script that launches calibrated self-play shards, merges outputs, exports, trains, and runs arena gates.
- Revisit in-process or ONNX inference only after measuring Modal shard throughput; HTTP is still overhead, but not the only bottleneck.
- Keep subtree reuse disabled for strength runs until state identity checks make it safe.

## Implemented Scaling Surfaces

- `az_selfplay -metrics=/path/metrics.json` writes throughput, branching, phase timing, scenario counts, final round/phase counts, action-type counts, completed/truncated games, worker count, nanosecond timing, and records-per-second.
- `az_selfplay -workers=N -progress` runs games in parallel and writes per-game JSON progress to stderr.
- `az_selfplay -compact_records` omits debug state snapshots from JSONL rows while preserving observation, legal-action, policy, action, and outcome fields.
- `az_selfplay -reuse_tree` keeps the tree API active but currently advances to a fresh root after each real move. A larger self-play run exposed stale pending-decision actions when expanded subtrees were reused without state identity checks.
- `az_selfplay -global_batch_size=N` merges concurrent evaluator batches across self-play workers before calling the wrapped batch evaluator.
- `az_buffer` builds replay buffers from multiple JSONL sources. Repeat `-source`, optionally as `path@limit`, to stream full sources and deterministic-reservoir-sample capped historical pools.
- `training_mix` samples both deterministic base scenarios and randomized scenarios.
- `randomized_base` samples base-game faction pairs, seat order, starting dwelling anchors, scoring tiles, and bonus cards.
- Arena reports now include scenario counts, average plies, search simulations, win-rate standard error, and 95% confidence interval.
- `az_eval -workers=N -progress` runs arena games in parallel and writes per-game JSON progress to stderr.
- `az_eval` reports a structured promotion decision. Use `-promote_min_games` and `-promote_ci95_lower_bound` when a run should require statistical confidence, not only a raw win-rate threshold.
- `az_eval` compares an HTTP candidate against an HTTP baseline, or against the heuristic fallback when no baseline URL is supplied.
- `az_train_torch --architecture=hex` uses observation shape `[global, hexes, per_hex]` to encode hexes with shared weights and pool board embeddings into policy/value heads.
- `az_infer_torch` serves `/evaluate`, `/evaluate_batch`, `/evaluate_binary`, and `/evaluate_batch_binary`; exposes checkpoint schema/shape/architecture on `/healthz`; suppresses access logs by default; and exposes `--torch-threads` / `--torch-interop-threads` for CPU-thread tuning.
- `az_replay_seeds` imports one replay text file or a directory of replay text files and emits generated snapshot seeds. Self-play can sample them with `-scenario=snapshots:/path/to/seeds.jsonl`. Use `-summary` to write seed coverage counts by source, round, phase, player count, root faction, and faction presence.

## Conversion Pruning Policy

AZ should prune free conversions that produce the same strategic position with strictly fewer resources. Do not emit standalone `priest -> worker`, `worker -> coin`, or Alchemists `2C -> 1VP` search actions. Replay/AZ auto-funding may still use `priest -> worker`, `worker -> coin`, and Alchemists `1VP -> 1C`, but only when required to pay the concrete action cost and only up to the shortfall.

Power conversions are different because spending Bowl III power into Bowl I can increase future leech capacity. Keep `power -> coin`, `power -> worker`, and `power -> priest` candidates when they move Bowl III power and therefore allow additional future leech. In AZ self-play this is canonicalized as a pre-action conversion before the main action; the live game engine still supports equivalent post-action free conversions through the normal confirmation window.

## Current Local Milestone

The local full-game scaling milestone now has two distinct datasets:

- Policy-prior bootstrap: `/tmp/tm_az_scale_100k/loop/iter_0001/selfplay.jsonl`
  - `100000` records, `max_plies=400`, `training_mix`, generated with `-sims=0`.
  - Use this for pipeline scale and broad supervised warm-starts, not as MCTS-improved data.
- Neural MCTS batch: `/tmp/tm_az_scale_next/neural_mcts_s8_selfplay.jsonl`
  - `2065` records from 20 full games, `max_plies=400`, `sims=8`, `batch_size=8`, no truncations.
  - Metrics: all 20 games reached `finalRoundCounts={"6":20}` and `finalPhaseCounts={"end":20}`.
  - Export: `/tmp/tm_az_scale_next/mcts_s8_export`.
  - Candidate checkpoint: `/tmp/tm_az_scale_next/tm_az_policy_value_h256_mcts_s8.pt`.

The next scaling pass added a replay buffer and h512 training. During that pass, broad replay-mode leakage in AZ clone execution was fixed by replacing `ReplayMode["__replay__"]` with the dedicated AZ auto-funding flag `ReplayMode["__az_auto_conversions__"]`. Normal pass actions now advance rounds correctly; do not hide pass actions to force full-game data.

- Initial mixed buffer: `/tmp/tm_az_scale_next/replay_buffer_bootstrap5k_mcts2k.jsonl`
  - `7065` records: `5000` sampled bootstrap rows plus all `2065` sims=8 rows.
  - Export: `/tmp/tm_az_scale_next/replay_buffer_export`.
  - Checkpoint: `/tmp/tm_az_scale_next/tm_az_policy_value_h512_mixed_bootstrap_mcts.pt`.
  - Training: `hidden_size=512`, `epochs=3`, final loss `1.9141`.
- Historical full-game sims=8 batch: `/tmp/tm_az_scale_next/neural_mcts_s8_50g_h512mixed_minpass6.jsonl`
  - Command shape at the time: `-episodes=50 -max_plies=400 -sims=8 -batch_size=8 -min_pass_round=6`.
  - Metrics: `4272` records, `49/50` terminal games, `1` truncation, `finalRoundCounts={"5":1,"6":49}`, `averagePliesPerEpisode=85.44`, `recordsPerSecond=5.58`.
  - This is now a historical artifact, not preferred training data. A follow-up no-pass-hiding smoke after the replay-mode fix reached `8/8` games at `finalPhase=end`, proving pass suppression was unnecessary.
- Historical full-game mixed buffer: `/tmp/tm_az_scale_next/replay_buffer_mixed_fullgames.jsonl`
  - `26337` records: `20000` sampled bootstrap rows, all `2065` original sims=8 rows, and all `4272` full-game sims=8 rows.
  - Export: `/tmp/tm_az_scale_next/replay_buffer_mixed_fullgames_export`.
  - Checkpoint: `/tmp/tm_az_scale_next/tm_az_policy_value_h512_mixed_fullgames.pt`.
  - Training: `hidden_size=512`, `epochs=3`, final loss `1.8456`.

The current clean normal-pass performance pass keeps pass visible and uses four workers:

- Legality fix: passed players no longer receive ordinary main-turn actions when pending resolution temporarily makes them appear current. Pending reactions are still generated by the pending-action path.
- Heuristic smoke after the fix: `8/8` games reached round 6/end with `437` records at `29.4` records/sec using `-workers=4`.
- Clean neural MCTS batch: `/tmp/tm_az_scale_clean/neural_mcts_s8_50g_h512_normalpass_w4.jsonl`
  - Command shape: `-episodes=50 -max_plies=400 -sims=8 -batch_size=8 -workers=4`.
  - Metrics: `5045` records, `50` completed games, `46` terminal games, `4` truncations, `finalRoundCounts={"2":1,"4":1,"5":2,"6":46}`, `finalPhaseCounts={"action":4,"end":46}`, `averagePliesPerEpisode=100.9`, `recordsPerSecond=17.28`.
- Clean mixed buffer: `/tmp/tm_az_scale_clean/replay_buffer_mixed_normalpass.jsonl`
  - `27110` records: `20000` sampled bootstrap rows, all `2065` original no-truncation sims=8 rows, and all `5045` clean normal-pass worker rows.
  - Export: `/tmp/tm_az_scale_clean/replay_buffer_mixed_normalpass_export`.
  - Export result: `27110` samples, `6169` actions.
  - Checkpoint: `/tmp/tm_az_scale_clean/tm_az_policy_value_h512_mixed_normalpass.pt`.
  - Training: `hidden_size=512`, `epochs=3`, final loss `1.7084`.
- Clean h512 arena gate: `/tmp/tm_az_scale_clean/arena_h512_normalpass_vs_h512_historical_50g_w4.json`
  - Candidate score `27-23` (`winRate=0.54`, 95% CI `[0.402, 0.678]`) versus the historical h512 checkpoint.
  - The candidate did not promote because the lower confidence bound stayed below `0.45`.
  - Runtime was `387424ms` with `workers=4`, `averagePlies=86.88`, `averageMargin=0.0565`.

The clean candidate is directionally better than the historical h512 checkpoint in this 50-game gate, but it is not statistically proven. The next quality target is diagnosing the randomized scenarios that still truncate at 400 plies.

## Promotion Result

The next promotion cycle used the clean h512 checkpoint as the baseline and generated a fresh 60-game neural MCTS batch without subtree reuse:

- Self-play: `/tmp/tm_az_promotion_20260604/selfplay_60g.jsonl`
  - `5070` records from `60` games, `sims=8`, `batch_size=8`, `workers=4`, `max_plies=400`.
  - Terminal quality: `58/60` terminal games, `2` truncations, `finalRoundCounts={"2":1,"4":1,"6":58}`.
  - Throughput: `17.98` records/sec.
- Candidate buffer: `/tmp/tm_az_promotion_20260604/replay_buffer_candidate.jsonl`
  - `32180` records: existing clean normal-pass mixed buffer plus the fresh 60-game batch.
  - Export: `/tmp/tm_az_promotion_20260604/export` with `32180` samples and `6715` actions.
- Candidate checkpoint: `/tmp/tm_az_promotion_20260604/tm_az_policy_value_h512_candidate.pt`
  - h512 hex model, `4` epochs, final loss `1.4700`.
- Promotion gate: `/tmp/tm_az_promotion_20260604/arena_100g_candidate_vs_baseline.json`
  - `100` games, `sims=8`, `batch_size=8`, `workers=4`, seed `4401`.
  - Candidate score: `63-35-2`, win rate `0.64`, 95% CI `[0.54592, 0.73408]`.
  - Promotion policy: `minGames=100`, `minWinRate=0.5`, `minCi95LowerBound=0.45`.
  - Result: promoted.

The full-game h512 promotion gate compared the full-game checkpoint against the earlier h512 mixed checkpoint:

```bash
cd server
bazel run //cmd/az_eval:az_eval -- \
  -candidate_url=http://127.0.0.1:9108/evaluate \
  -baseline_url=http://127.0.0.1:9106/evaluate \
  -scenario=training_mix \
  -games=50 \
  -max_plies=400 \
  -sims=8 \
  -batch_size=8 \
  -promote_min_games=50 \
  -promote_win_rate=0.5 \
  -promote_ci95_lower_bound=0.45 \
  -seed=1901 \
  -output=/tmp/tm_az_scale_next/arena_h512_fullgames_vs_h512_mixed_50g_minpass6.json
```

Result: candidate did not promote. It scored `23-27` (`winRate=0.46`, 95% CI `[0.322, 0.598]`) with `averagePlies=88.54`. Treat this as a historical gate from the previous run; rerun the gate with normal pass actions after regenerating data under the no-pass-hiding policy.

The first promotion smoke compared the MCTS-trained candidate against the 100k bootstrap checkpoint:

```bash
cd server
bazel run //cmd/az_eval:az_eval -- \
  -candidate_url=http://127.0.0.1:9105/evaluate \
  -baseline_url=http://127.0.0.1:9104/evaluate \
  -scenario=training_mix \
  -games=8 \
  -max_plies=400 \
  -sims=8 \
  -batch_size=8 \
  -promote_min_games=8 \
  -promote_win_rate=0.5 \
  -seed=1001 \
  -output=/tmp/tm_az_scale_next/arena_h256_mcts_s8_vs_h256_100k.json
```

Result: candidate score `3/8` (`winRate=0.375`), so it did not promote. This is a working gate, not a model-strength failure by itself; 8 games is only a smoke.

## Recommended Run Ladder

Start with tiny smoke runs after code changes:

```bash
cd server
bazel run //cmd/az_selfplay:az_selfplay -- \
  -scenario=training_mix \
  -episodes=2 \
  -max_plies=20 \
  -sims=4 \
  -batch_size=2 \
  -metrics=/tmp/tm_az_smoke_metrics.json \
  -output=/tmp/tm_az_smoke_selfplay.jsonl
```

Then run a small local neural iteration:

```bash
cd server
bazel run //cmd/az_selfplay:az_selfplay -- \
  -scenario=training_mix \
  -episodes=40 \
  -sims=16 \
  -batch_size=4 \
  -max_plies=120 \
  -workers=2 \
  -progress \
  -metrics=/tmp/tm_az_runs/iter_0001/selfplay_metrics.json \
  -output=/tmp/tm_az_runs/iter_0001/selfplay.jsonl
```

Export the self-play records:

```bash
cd server
bazel run //cmd/az_export:az_export -- \
  -input=/tmp/tm_az_runs/iter_0001/selfplay.jsonl \
  -samples=/tmp/tm_az_runs/iter_0001/samples.jsonl \
  -vocab=/tmp/tm_az_runs/iter_0001/action_vocab.json \
  -manifest=/tmp/tm_az_runs/iter_0001/dataset_manifest.json
```

Export and train a neural candidate:

```bash
cd server
bazel run //cmd/az_train_torch:az_train_torch -- \
  --samples=/tmp/tm_az_runs/iter_0001/samples.jsonl \
  --manifest=/tmp/tm_az_runs/iter_0001/dataset_manifest.json \
  --vocab=/tmp/tm_az_runs/iter_0001/action_vocab.json \
  --output=/tmp/tm_az_policy_value.pt \
  --architecture=hex \
  --epochs=5 \
  --hidden_size=256
```

Serve and generate neural self-play:

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
  -scenario=training_mix \
  -episodes=200 \
  -max_plies=400 \
  -sims=32 \
  -batch_size=8 \
  -workers=4 \
  -progress \
  -model_url=http://127.0.0.1:9097/evaluate \
  -metrics=/tmp/tm_az_neural_metrics.json \
  -output=/tmp/tm_az_neural_selfplay.jsonl
```

Build a mixed replay buffer before export/training:

```bash
cd server
bazel run //cmd/az_buffer:az_buffer -- \
  -source=/tmp/tm_az_scale_100k/loop/iter_0001/selfplay.jsonl@20000 \
  -source=/tmp/tm_az_scale_next/neural_mcts_s8_selfplay.jsonl \
  -source=/tmp/tm_az_scale_clean/neural_mcts_s8_50g_h512_normalpass_w4.jsonl \
  -output=/tmp/tm_az_scale_clean/replay_buffer_mixed_normalpass.jsonl \
  -summary=/tmp/tm_az_scale_clean/replay_buffer_mixed_normalpass_summary.json \
  -seed=2701
```

Generate replay-derived midgame seeds:

```bash
cd server
bazel run //cmd/az_replay_seeds:az_replay_seeds -- \
  -input=/path/to/replay.txt \
  -format=snellman \
  -every=20 \
  -max=200 \
  -output=/tmp/tm_az_replay_seeds.jsonl
```

Generate a broader seed set from a replay fixture directory:

```bash
cd server
bazel run //cmd/az_replay_seeds:az_replay_seeds -- \
  -input_dir=internal/replay/testdata/snellman_batch \
  -pattern='*.txt' \
  -format=snellman \
  -every=20 \
  -max=1000 \
  -max_per_replay=50 \
  -phase=Action \
  -output=/tmp/tm_az_replay_seed_batch.jsonl \
  -summary=/tmp/tm_az_replay_seed_batch_summary.json
```

Inspect the summary before training. A useful seed pool should have nonzero coverage across multiple roots, rounds, and phases; otherwise a strong model can overfit to a narrow midgame distribution.

Use filters to build narrower curriculum pools when needed:

- `-phase=Action` or `-phase=Income`
- `-min_round=2 -max_round=5`
- `-player_count=2` or `-player_count=4`
- `-root_faction=Cultists`

The summary includes `skippedByReason` so it is clear when filters are too tight.

Use those seeds as a scenario source:

```bash
cd server
bazel run //cmd/az_selfplay:az_selfplay -- \
  -scenario=snapshots:/tmp/tm_az_replay_seed_batch.jsonl \
  -episodes=100 \
  -max_plies=120 \
  -sims=32 \
  -batch_size=8 \
  -output=/tmp/tm_az_replay_seed_selfplay.jsonl
```

Evaluate a retained model directly:

```bash
cd server
bazel run //cmd/az_eval:az_eval -- \
  -candidate_url=http://127.0.0.1:9098/evaluate \
  -baseline_url=http://127.0.0.1:9097/evaluate \
  -scenario=training_mix \
  -games=100 \
  -sims=32 \
  -batch_size=4 \
  -workers=4 \
  -progress \
  -max_plies=400 \
  -promote_win_rate=0.55 \
  -promote_min_games=50 \
  -promote_ci95_lower_bound=0.50 \
  -output=/tmp/tm_az_eval_iter_0005.json
```

## Promotion Discipline

Do not promote only because a candidate wins one tiny match. Use:

- `training_mix` or an explicit comma-separated scenario suite.
- Use `84` arena games for local smoke checks. Use at least `168` arena games for promotion candidates, and `200+` for serious promotion when runtime allows.
- Fixed seeds recorded in `report.json`.
- A promotion threshold that considers confidence interval width. A candidate with `winRate=0.56` over 8 games is not meaningfully proven.
- `-promote_min_games` for minimum sample size and `-promote_ci95_lower_bound` for lower-bound confidence gating.
- `ratings.json` as a long-lived trend signal, not as a substitute for arena confidence on a promotion decision.

## Metrics To Watch

- `recordsPerSecond`: primary self-play throughput.
- `averageBranchingFactor`: catches action-surface explosions.
- `legalMillis / searchMillis / applyMillis`: shows whether legality, MCTS, or model inference is the bottleneck.
- `truncatedGames`: high truncation means value targets are mostly heuristic margins, not real terminal outcomes.
- `scenarioCounts`: confirms the run did not collapse to one scenario bucket.

Current local throughput observations:

- Export/training are usable at 27k rows: export produced `27110` samples / `6169` actions; h512 training took `212` batches per epoch.
- MCTS generation dominates. The historical single-worker 50-game full-game sims=8 batch produced `5.58` records/sec. The clean four-worker 50-game batch produced `17.28` records/sec.
- HTTP evaluator overhead is still material. Four-worker arena reduced the 50-game h512 gate from about `12.5` minutes to `6.46` minutes, but search/evaluator time still dominates wall-clock runtime.
- The first opt-in micro-optimization benchmark used 12 games, `training_mix`, `max_plies=400`, `sims=8`, `batch_size=8`, `workers=4`, seed `3101`, and the clean h512 checkpoint.
  - Baseline: `1122` records, `98.895s`, `11.35` records/sec, output `27MB`.
  - `-compact_records -reuse_tree -global_batch_size=32`: `1123` records, `98.735s`, `11.37` records/sec, output `15MB`.
  - `workers=8`, compact/tree/global batch: `1123` records, `103.434s`, `10.86` records/sec.
  - Torch server with `--torch-threads=1 --torch-interop-threads=1`: `1122` records, `98.612s`, `11.38` records/sec.
- The second performance pass added binary float32 evaluator transport and MCTS batch-leaf deduplication.
  - Binary transport only: `1122` records, `96.925s`, `11.58` records/sec.
  - Binary transport plus MCTS leaf dedupe: `1122` records, `91.878s`, `12.21` records/sec.
  - Binary transport plus leaf dedupe plus compact/tree/global-batch knobs: `1123` records, `90.141s`, `12.46` records/sec, output `15MB`.
  - Final root-binary rerun measured `12.36` records/sec, so treat `12.46` as the best observed local run and `12.2-12.4` as the realistic current range on this benchmark.
- Compact records are useful for storage/I/O. The measured throughput gain came primarily from deduplicating repeated MCTS leaf evaluations inside a batch, with a smaller contribution from binary evaluator transport. The remaining larger speed path is still likely in-process/ONNX inference or a deeper search/evaluator redesign.

## Remaining Work

- Add ONNX or in-process inference if HTTP latency dominates.
- Add scenario-level truncation reports so long randomized games can be separated from model weakness.
- Add setup/auction self-play once the action surface is cheap enough for those phases.
