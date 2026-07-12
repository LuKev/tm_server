#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
SERVER="$ROOT/server"
RUN_ROOT="$ROOT/artifacts/az/five-cycles-20260712"
INITIAL_MODEL=${INITIAL_MODEL:-/Users/kevin/.codex/worktrees/az-selfplay-cycle-20260711/artifacts/az/models/promoted-h512-selfplay-iter1-20260711/model.pt}
INCUMBENT_PORT=9190
NEXT_PORT=9191
INCUMBENT_PID=""
SUMMARY="$RUN_ROOT/summary.jsonl"

cleanup() {
  [[ -z "$INCUMBENT_PID" ]] || kill "$INCUMBENT_PID" 2>/dev/null || true
  for pid in $(jobs -pr); do
    kill "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

mkdir -p "$RUN_ROOT/incumbent"
cp "$INITIAL_MODEL" "$RUN_ROOT/incumbent/model.pt"
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

"$VOCAB" --checkpoint "$RUN_ROOT/incumbent/model.pt" \
  --output "$RUN_ROOT/incumbent/action_vocab.json"
INCUMBENT_PID=$(start_server "$RUN_ROOT/incumbent/model.pt" "$INCUMBENT_PORT" "$RUN_ROOT/incumbent/infer.log")
INCUMBENT_MODEL="$RUN_ROOT/incumbent/model.pt"
INCUMBENT_VOCAB="$RUN_ROOT/incumbent/action_vocab.json"

for ITER in 2 3 4 5 6; do
  ITER_ROOT="$RUN_ROOT/iter$ITER"
  mkdir -p "$ITER_ROOT"/{selfplay,dataset,candidate,arena,promoted}
  echo "$(date -u +%FT%TZ) iteration=$ITER phase=selfplay" | tee -a "$RUN_ROOT/run.log"

  "$SELFPLAY" \
    -episodes=168 -scenario=matrix:base_ordered -workers=8 -sims=8 \
    -batch_size=8 -global_batch_size=64 -global_batch_delay_ms=2 \
    -max_depth=120 -max_plies=500 -compact_records -reuse_tree \
    -seed=$((20260720 + ITER)) -model_url="http://127.0.0.1:$INCUMBENT_PORT" \
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
    --seed $((20260730 + ITER)) 2>"$ITER_ROOT/candidate/train.log"

  CANDIDATE_PID=$(start_server "$ITER_ROOT/candidate/model.pt" "$NEXT_PORT" "$ITER_ROOT/candidate/infer.log")
  echo "$(date -u +%FT%TZ) iteration=$ITER phase=arena" | tee -a "$RUN_ROOT/run.log"
  "$EVAL" \
    -candidate_url="http://127.0.0.1:$NEXT_PORT" \
    -baseline_url="http://127.0.0.1:$INCUMBENT_PORT" \
    -scenario=matrix:base_ordered -games=168 -workers=8 -sims=8 \
    -batch_size=8 -global_batch_size=64 -global_batch_delay_ms=2 \
    -max_depth=120 -max_plies=500 -seed=$((20260740 + ITER)) \
    -promote_win_rate=0.55 -promote_min_games=168 \
    -promote_ci95_lower_bound=0.45 \
    -output="$ITER_ROOT/arena/arena_168.json" -progress \
    2>"$ITER_ROOT/arena/progress.jsonl"

  jq -c --argjson iteration "$ITER" \
    --slurpfile selfplay "$ITER_ROOT/selfplay/selfplay_metrics.json" '
    {
      iteration: $iteration,
      promoted: .promotion.promoted,
      arena: {
        candidateWins: .result.candidateWins,
        baselineWins: .result.baselineWins,
        draws: .result.draws,
        score: .result.winRate,
        ci95: .result.winRateCi95,
        candidateAverageVP: .result.candidateAverageScore,
        baselineAverageVP: .result.baselineAverageScore,
        averageVPChange: .result.averageScoreDifference
      },
      selfplay: {
        averageFinalVP: $selfplay[0].averageFinalScore,
        averageWinningVP: $selfplay[0].averageWinningScore,
        averageLosingVP: $selfplay[0].averageLosingScore,
        averageMarginVP: $selfplay[0].averageScoreMargin,
        r1ByFaction: $selfplay[0].r1BuildRatesByFaction,
        finalVPByFaction: $selfplay[0].finalScoresByFaction
      }
    }' "$ITER_ROOT/arena/arena_168.json" >>"$SUMMARY"

  if jq -e '.promotion.promoted == true' "$ITER_ROOT/arena/arena_168.json" >/dev/null; then
    cp "$ITER_ROOT/candidate/model.pt" "$ITER_ROOT/promoted/model.pt"
    cp "$ITER_ROOT/dataset/action_vocab.json" "$ITER_ROOT/promoted/action_vocab.json"
    cp "$ITER_ROOT/selfplay/selfplay_metrics.json" "$ITER_ROOT/promoted/selfplay_metrics.json"
    cp "$ITER_ROOT/arena/arena_168.json" "$ITER_ROOT/promoted/arena_168.json"
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
    echo "$(date -u +%FT%TZ) iteration=$ITER promoted=false" | tee -a "$RUN_ROOT/run.log"
  fi
done

echo "$(date -u +%FT%TZ) complete" | tee -a "$RUN_ROOT/run.log"
