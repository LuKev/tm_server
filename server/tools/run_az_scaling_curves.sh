#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -lt 6 || "$#" -gt 8 ]]; then
  echo "usage: $0 MODEL_BENCHMARK EVAL INFERENCE CHECKPOINT OUTPUT_DIR ENGINE_COMMIT [DEVICE] [CASES]" >&2
  exit 2
fi

model_benchmark="$1"
evaluator="$2"
inference="$3"
checkpoint="$4"
output_dir="$5"
engine_commit="$6"
device="${7:-auto}"
cases="${8:-28}"

model_config="${TM_AZ_CURVE_MODEL_CONFIG:-large}"
batch_sizes="${TM_AZ_CURVE_BATCH_SIZES:-1 8 32}"
search_budgets="${TM_AZ_CURVE_SEARCH_BUDGETS:-1 8 32 128}"
baseline_simulations="${TM_AZ_CURVE_BASELINE_SIMULATIONS:-1}"
actions="${TM_AZ_CURVE_ACTIONS:-128}"
warmup="${TM_AZ_CURVE_WARMUP:-5}"
iterations="${TM_AZ_CURVE_ITERATIONS:-20}"
torch_threads="${TM_AZ_TORCH_THREADS:-1}"
arena_games_per_batch="${TM_AZ_ARENA_GAMES_PER_BATCH:-8}"
allow_random="${TM_AZ_CURVE_ALLOW_RANDOM:-0}"

if [[ -z "${checkpoint}" && "${allow_random}" != "1" ]]; then
  echo "a trained checkpoint is required; set TM_AZ_CURVE_ALLOW_RANDOM=1 only for smoke tests" >&2
  exit 1
fi

if [[ -e "${output_dir}" ]]; then
  echo "refusing to overwrite scaling-curve directory: ${output_dir}" >&2
  exit 1
fi
mkdir -p "${output_dir}/model-batches" "${output_dir}/search-reports" "${output_dir}/search-summaries"

for batch_size in ${batch_sizes}; do
  result="${output_dir}/model-batches/batch-${batch_size}.json"
  "${model_benchmark}" \
    --model-config "${model_config}" \
    --checkpoint "${checkpoint}" \
    --device "${device}" \
    --mode inference \
    --batch-size "${batch_size}" \
    --actions "${actions}" \
    --warmup "${warmup}" \
    --iterations "${iterations}" \
    --torch-threads "${torch_threads}" \
    --seed 0 >"${result}"
done

for candidate_simulations in ${search_budgets}; do
  result="${output_dir}/search-summaries/candidate-${candidate_simulations}-baseline-${baseline_simulations}.json"
  "${evaluator}" \
    --inference "${inference}" \
    --candidate-checkpoint "${checkpoint}" \
    --baseline-checkpoint "${checkpoint}" \
    --candidate-model-seed 0 \
    --baseline-model-seed 0 \
    --model-config "${model_config}" \
    --inference-device "${device}" \
    --inference-torch-threads "${torch_threads}" \
    --output "${output_dir}/search-reports" \
    --games-per-batch "${arena_games_per_batch}" \
    --cases "${cases}" \
    --candidate-simulations "${candidate_simulations}" \
    --baseline-simulations "${baseline_simulations}" \
    --engine-commit "${engine_commit}" >"${result}"
done
