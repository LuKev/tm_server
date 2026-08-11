#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -lt 6 || "$#" -gt 12 ]]; then
  echo "usage: $0 SELFPLAY INFERENCE LEARNER EVAL RUN_DIR ENGINE_COMMIT [CYCLES] [GAMES] [SIMULATIONS] [TRAIN_STEPS] [EVAL_CASES] [MODEL_CONFIG]" >&2
  exit 2
fi

selfplay="$1"
inference="$2"
learner="$3"
evaluator="$4"
run_dir="$5"
engine_commit="$6"
cycles="${7:-1}"
games="${8:-8}"
simulations="${9:-8}"
train_steps="${10:-100}"
eval_cases="${11:-8}"
model_config="${12:-large}"
inference_device="${TM_AZ_INFERENCE_DEVICE:-cpu}"
learner_device="${TM_AZ_LEARNER_DEVICE:-${inference_device}}"
torch_threads="${TM_AZ_TORCH_THREADS:-1}"
actor_games_per_batch="${TM_AZ_ACTOR_GAMES_PER_BATCH:-8}"
replay_cache_games="${TM_AZ_REPLAY_CACHE_GAMES:-8}"
checkpoint_dir="${run_dir}/checkpoints"
replay_root="${run_dir}/replay"
evaluation_dir="${run_dir}/evaluation"
mkdir -p "${checkpoint_dir}" "${replay_root}" "${evaluation_dir}"

replay_args=()
for ((cycle = 1; cycle <= cycles; cycle++)); do
  cycle_replay="${replay_root}/cycle-${cycle}"
  actor_checkpoint=(--checkpoint "")
  learner_parent=(--initial-checkpoint "")
  previous_reference="${checkpoint_dir}/previous.json"
  if [[ -f "${checkpoint_dir}/latest.json" ]]; then
    actor_checkpoint=(--checkpoint-latest "${checkpoint_dir}/latest.json")
    learner_parent=(--initial-latest "${checkpoint_dir}/latest.json")
    cp "${checkpoint_dir}/latest.json" "${previous_reference}"
  fi

  "${selfplay}" \
    --inference "${inference}" \
    "${actor_checkpoint[@]}" \
    --model-config "${model_config}" \
    --inference-device "${inference_device}" \
    --inference-torch-threads "${torch_threads}" \
    --output "${cycle_replay}" \
    --games "${games}" \
    --games-per-batch "${actor_games_per_batch}" \
    --simulations "${simulations}" \
    --seed "$((1000000 + cycle * games))" \
    --model-seed 0 \
    --engine-commit "${engine_commit}"
  replay_args+=(--input "${cycle_replay}")

  "${learner}" \
    "${replay_args[@]}" \
    "${learner_parent[@]}" \
    --model-config "${model_config}" \
    --device "${learner_device}" \
    --torch-threads "${torch_threads}" \
    --output-dir "${checkpoint_dir}" \
    --engine-commit "${engine_commit}" \
    --steps "${train_steps}" \
    --replay-cache-games "${replay_cache_games}" \
    --model-seed 0 \
    --seed "${cycle}"

  "${evaluator}" \
    --inference "${inference}" \
    --candidate-latest "${checkpoint_dir}/latest.json" \
    --model-config "${model_config}" \
    --inference-device "${inference_device}" \
    --inference-torch-threads "${torch_threads}" \
    --baseline-uniform \
    --output "${evaluation_dir}" \
    --cases "${eval_cases}" \
    --simulations "${simulations}" \
    --engine-commit "${engine_commit}"
  if [[ -f "${previous_reference}" ]]; then
    "${evaluator}" \
      --inference "${inference}" \
      --candidate-latest "${checkpoint_dir}/latest.json" \
      --baseline-latest "${previous_reference}" \
      --model-config "${model_config}" \
      --inference-device "${inference_device}" \
      --inference-torch-threads "${torch_threads}" \
      --output "${evaluation_dir}" \
      --cases "${eval_cases}" \
      --simulations "${simulations}" \
      --engine-commit "${engine_commit}"
  fi
done
