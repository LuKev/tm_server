#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
SERVER="$ROOT/server"
RUN_ROOT="$ROOT/artifacts/az/paired-anchor-cycles-20260712"
INITIAL_MODEL=${INITIAL_MODEL:-/Users/kevin/.codex/worktrees/az-five-cycles-20260712/artifacts/az/models/promoted-h512-selfplay-iter5-20260712/model.pt}
ANCHOR_MODEL=${ANCHOR_MODEL:-/Users/kevin/.codex/worktrees/az-selfplay-cycle-20260711/artifacts/az/models/promoted-h512-selfplay-iter1-20260711/model.pt}
INCUMBENT_PORT=9390
NEXT_PORT=9391
ANCHOR_PORT=9399
INCUMBENT_PID=""
ANCHOR_PID=""
SUMMARY="$RUN_ROOT/summary.jsonl"

cleanup() {
  [[ -z "$INCUMBENT_PID" ]] || kill "$INCUMBENT_PID" 2>/dev/null || true
  [[ -z "$ANCHOR_PID" ]] || kill "$ANCHOR_PID" 2>/dev/null || true
  for pid in $(jobs -pr); do
    kill "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

mkdir -p "$RUN_ROOT/incumbent" "$RUN_ROOT/anchor"
cp "$INITIAL_MODEL" "$RUN_ROOT/incumbent/model.pt"
cp "$ANCHOR_MODEL" "$RUN_ROOT/anchor/model.pt"
cd "$SERVER"

SELFPLAY="$SERVER/bazel-bin/cmd/az_selfplay/az_selfplay_/az_selfplay"
EXPORT="$SERVER/bazel-bin/cmd/az_export/az_export_/az_export"
TRAIN="$SERVER/bazel-bin/cmd/az_train_torch/az_train_torch"
EVAL="$SERVER/bazel-bin/cmd/az_eval/az_eval_/az_eval"
INFER="$SERVER/bazel-bin/cmd/az_infer_torch/az_infer_torch"
VOCAB="$SERVER/bazel-bin/cmd/az_checkpoint_vocab/az_checkpoint_vocab"

start_server() {
  local model=$1
  local port=$2
  local log=$3
  "$INFER" --checkpoint "$model" --host 127.0.0.1 --port "$port" \
    --torch-threads 2 --torch-interop-threads 1 >"$log" 2>&1 &
  local pid=$!
  for _ in $(seq 1 60); do
    if /usr/bin/curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null 2>&1; then
      echo "$pid"
      return 0
    fi
    sleep 1
  done
  kill "$pid" 2>/dev/null || true
  return 1
}

"$VOCAB" --checkpoint "$RUN_ROOT/incumbent/model.pt" --output "$RUN_ROOT/incumbent/action_vocab.json"
INCUMBENT_PID=$(start_server "$RUN_ROOT/incumbent/model.pt" "$INCUMBENT_PORT" "$RUN_ROOT/incumbent/infer.log")
ANCHOR_PID=$(start_server "$RUN_ROOT/anchor/model.pt" "$ANCHOR_PORT" "$RUN_ROOT/anchor/infer.log")
INCUMBENT_MODEL="$RUN_ROOT/incumbent/model.pt"
INCUMBENT_VOCAB="$RUN_ROOT/incumbent/action_vocab.json"

for ITER in 7 8 9 10 11; do
  ITER_ROOT="$RUN_ROOT/iter$ITER"
  mkdir -p "$ITER_ROOT"/{selfplay,dataset,candidate,parent_gate,anchor_gate,promoted}
  echo "$(date -u +%FT%TZ) iteration=$ITER phase=selfplay" | tee -a "$RUN_ROOT/run.log"

  "$SELFPLAY" \
    -episodes=168 -scenario=matrix:base_ordered -workers=8 -sims=8 \
    -batch_size=8 -global_batch_size=64 -global_batch_delay_ms=2 \
    -max_depth=120 -max_plies=500 -compact_records -reuse_tree \
    -seed=$((20260800 + ITER)) -model_url="http://127.0.0.1:$INCUMBENT_PORT" \
    -output="$ITER_ROOT/selfplay/selfplay.jsonl" \
    -metrics="$ITER_ROOT/selfplay/selfplay_metrics.json" -progress \
    2>"$ITER_ROOT/selfplay/progress.jsonl"

  echo "$(date -u +%FT%TZ) iteration=$ITER phase=export_train" | tee -a "$RUN_ROOT/run.log"
  "$EXPORT" \
    -input="$ITER_ROOT/selfplay/selfplay.jsonl" \
    -samples="$ITER_ROOT/dataset/samples.jsonl" \
    -vocab="$ITER_ROOT/dataset/action_vocab.json" \
    -manifest="$ITER_ROOT/dataset/dataset_manifest.json" \
    -seed_vocab="$INCUMBENT_VOCAB"
  "$TRAIN" \
    --samples "$ITER_ROOT/dataset/samples.jsonl" \
    --manifest "$ITER_ROOT/dataset/dataset_manifest.json" \
    --vocab "$ITER_ROOT/dataset/action_vocab.json" \
    --output "$ITER_ROOT/candidate/model.pt" \
    --epochs 5 --batch_size 128 --hidden_size 512 --lr 0.0001 \
    --architecture hex --init_checkpoint "$INCUMBENT_MODEL" \
    --init_allow_action_mismatch --shuffle_buffer 4096 \
    --seed=$((20260810 + ITER)) 2>"$ITER_ROOT/candidate/train.log"

  CANDIDATE_PID=$(start_server "$ITER_ROOT/candidate/model.pt" "$NEXT_PORT" "$ITER_ROOT/candidate/infer.log")
  echo "$(date -u +%FT%TZ) iteration=$ITER phase=parent_gate" | tee -a "$RUN_ROOT/run.log"
  "$EVAL" \
    -candidate_url="http://127.0.0.1:$NEXT_PORT" \
    -baseline_url="http://127.0.0.1:$INCUMBENT_PORT" \
    -scenario=matrix:base_paired -games=336 -workers=8 -sims=8 \
    -batch_size=8 -global_batch_size=64 -global_batch_delay_ms=2 \
    -max_depth=120 -max_plies=500 -seed=$((20260820 + ITER)) \
    -promote_win_rate=0.55 -promote_min_games=336 -promote_ci95_lower_bound=0.45 \
    -output="$ITER_ROOT/parent_gate/arena_336.json" -progress \
    2>"$ITER_ROOT/parent_gate/progress.jsonl"

  PARENT_PASSED=$(jq -r '.promotion.promoted' "$ITER_ROOT/parent_gate/arena_336.json")
  ANCHOR_PASSED=false
  if [[ "$PARENT_PASSED" == "true" ]]; then
    echo "$(date -u +%FT%TZ) iteration=$ITER phase=anchor_gate" | tee -a "$RUN_ROOT/run.log"
    "$EVAL" \
      -candidate_url="http://127.0.0.1:$NEXT_PORT" \
      -baseline_url="http://127.0.0.1:$ANCHOR_PORT" \
      -scenario=matrix:base_paired -games=672 -workers=8 -sims=8 \
      -batch_size=8 -global_batch_size=64 -global_batch_delay_ms=2 \
      -max_depth=120 -max_plies=500 -seed=$((20260830 + ITER)) \
      -promote_win_rate=0.50 -promote_min_games=672 -promote_ci95_lower_bound=0.50 \
      -output="$ITER_ROOT/anchor_gate/arena_672.json" -progress \
      2>"$ITER_ROOT/anchor_gate/progress.jsonl"
    ANCHOR_PASSED=$(jq -r '.promotion.promoted' "$ITER_ROOT/anchor_gate/arena_672.json")
  fi

  jq -n -c --argjson iteration "$ITER" \
    --argjson parentPassed "$PARENT_PASSED" --argjson anchorPassed "$ANCHOR_PASSED" \
    --slurpfile selfplay "$ITER_ROOT/selfplay/selfplay_metrics.json" \
    --slurpfile parent "$ITER_ROOT/parent_gate/arena_336.json" \
    --arg anchorPath "$ITER_ROOT/anchor_gate/arena_672.json" '
    {
      iteration: $iteration,
      promoted: ($parentPassed and $anchorPassed),
      parentPassed: $parentPassed,
      anchorPassed: $anchorPassed,
      selfplay: {
        averageFinalVP: $selfplay[0].averageFinalScore,
        averageWinningVP: $selfplay[0].averageWinningScore,
        averageLosingVP: $selfplay[0].averageLosingScore,
        averageMarginVP: $selfplay[0].averageScoreMargin,
        r1ByFaction: $selfplay[0].r1BuildRatesByFaction,
        finalVPByFaction: $selfplay[0].finalScoresByFaction
      },
      parentGate: {
        wins: $parent[0].result.candidateWins,
        losses: $parent[0].result.baselineWins,
        draws: $parent[0].result.draws,
        score: $parent[0].result.winRate,
        ci95: $parent[0].result.winRateCi95,
        candidateAverageVP: $parent[0].result.candidateAverageScore,
        baselineAverageVP: $parent[0].result.baselineAverageScore,
        averageVPChange: $parent[0].result.averageScoreDifference
      },
      anchorGate: (if $anchorPassed or ($anchorPath | test("arena_672")) and ($parentPassed) then null else null end)
    }' >"$ITER_ROOT/summary.partial.json"

  if [[ -f "$ITER_ROOT/anchor_gate/arena_672.json" ]]; then
    jq --slurpfile anchor "$ITER_ROOT/anchor_gate/arena_672.json" '
      .anchorGate = {
        wins: $anchor[0].result.candidateWins,
        losses: $anchor[0].result.baselineWins,
        draws: $anchor[0].result.draws,
        score: $anchor[0].result.winRate,
        ci95: $anchor[0].result.winRateCi95,
        candidateAverageVP: $anchor[0].result.candidateAverageScore,
        baselineAverageVP: $anchor[0].result.baselineAverageScore,
        averageVPChange: $anchor[0].result.averageScoreDifference
      }' "$ITER_ROOT/summary.partial.json" >"$ITER_ROOT/summary.json"
  else
    cp "$ITER_ROOT/summary.partial.json" "$ITER_ROOT/summary.json"
  fi
  jq -c . "$ITER_ROOT/summary.json" >>"$SUMMARY"

  if [[ "$PARENT_PASSED" == "true" && "$ANCHOR_PASSED" == "true" ]]; then
    cp "$ITER_ROOT/candidate/model.pt" "$ITER_ROOT/promoted/model.pt"
    cp "$ITER_ROOT/dataset/action_vocab.json" "$ITER_ROOT/promoted/action_vocab.json"
    cp "$ITER_ROOT/selfplay/selfplay_metrics.json" "$ITER_ROOT/promoted/selfplay_metrics.json"
    cp "$ITER_ROOT/parent_gate/arena_336.json" "$ITER_ROOT/promoted/parent_arena_336.json"
    cp "$ITER_ROOT/anchor_gate/arena_672.json" "$ITER_ROOT/promoted/anchor_arena_672.json"
    kill "$INCUMBENT_PID" 2>/dev/null || true
    wait "$INCUMBENT_PID" 2>/dev/null || true
    INCUMBENT_PID="$CANDIDATE_PID"
    INCUMBENT_MODEL="$ITER_ROOT/candidate/model.pt"
    INCUMBENT_VOCAB="$ITER_ROOT/dataset/action_vocab.json"
    old_port=$INCUMBENT_PORT
    INCUMBENT_PORT=$NEXT_PORT
    NEXT_PORT=$old_port
    echo "$(date -u +%FT%TZ) iteration=$ITER promoted=true" | tee -a "$RUN_ROOT/run.log"
  else
    kill "$CANDIDATE_PID" 2>/dev/null || true
    wait "$CANDIDATE_PID" 2>/dev/null || true
    echo "$(date -u +%FT%TZ) iteration=$ITER promoted=false parent=$PARENT_PASSED anchor=$ANCHOR_PASSED" | tee -a "$RUN_ROOT/run.log"
  fi
done

echo "$(date -u +%FT%TZ) complete" | tee -a "$RUN_ROOT/run.log"
