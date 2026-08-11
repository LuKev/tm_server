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
replay_window_shards="${TM_AZ_REPLAY_WINDOW_SHARDS:-10000}"
arena_games_per_batch="${TM_AZ_ARENA_GAMES_PER_BATCH:-8}"
examples_per_replay_ply="${TM_AZ_EXAMPLES_PER_REPLAY_PLY:-}"
optimizer="${TM_AZ_OPTIMIZER:-sgd}"
learning_rate="${TM_AZ_LEARNING_RATE:-1e-4}"
weight_decay="${TM_AZ_WEIGHT_DECAY:-1e-4}"
momentum="${TM_AZ_MOMENTUM:-}"
optimizer_args=(--optimizer "${optimizer}")
if [[ "${optimizer}" == "sgd" ]]; then
  momentum="${momentum:-0.9}"
  optimizer_args+=(--momentum "${momentum}")
elif [[ "${optimizer}" == "adamw" ]]; then
  if [[ -n "${momentum}" ]]; then
    echo "TM_AZ_MOMENTUM is only valid with TM_AZ_OPTIMIZER=sgd" >&2
    exit 1
  fi
  momentum="not_applicable"
else
  echo "TM_AZ_OPTIMIZER must be sgd or adamw" >&2
  exit 1
fi
if [[ -d "${run_dir}" && -n "$(find "${run_dir}" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
  echo "refusing to reuse nonempty run directory: ${run_dir}" >&2
  exit 1
fi

checkpoint_dir="${run_dir}/checkpoints"
replay_root="${run_dir}/replay"
evaluation_dir="${run_dir}/evaluation"
metrics_root="${run_dir}/metrics"
mkdir -p "${checkpoint_dir}" "${replay_root}" "${evaluation_dir}" "${metrics_root}"

manifest="${run_dir}/run-manifest.txt"
if [[ -e "${manifest}" ]]; then
  echo "refusing to overwrite run manifest: ${manifest}" >&2
  exit 1
fi
{
  printf 'format_version=1\n'
  printf 'engine_commit=%s\n' "${engine_commit}"
  printf 'cycles=%s\n' "${cycles}"
  printf 'games_per_cycle=%s\n' "${games}"
  printf 'simulations=%s\n' "${simulations}"
  printf 'train_steps=%s\n' "${train_steps}"
  printf 'examples_per_replay_ply=%s\n' "${examples_per_replay_ply}"
  printf 'evaluation_cases=%s\n' "${eval_cases}"
  printf 'model_config=%s\n' "${model_config}"
  printf 'inference_device=%s\n' "${inference_device}"
  printf 'learner_device=%s\n' "${learner_device}"
  printf 'torch_threads=%s\n' "${torch_threads}"
  printf 'actor_games_per_batch=%s\n' "${actor_games_per_batch}"
  printf 'arena_games_per_batch=%s\n' "${arena_games_per_batch}"
  printf 'replay_cache_games=%s\n' "${replay_cache_games}"
  printf 'replay_window_shards=%s\n' "${replay_window_shards}"
  printf 'optimizer=%s\n' "${optimizer}"
  printf 'learning_rate=%s\n' "${learning_rate}"
  printf 'weight_decay=%s\n' "${weight_decay}"
  printf 'momentum=%s\n' "${momentum}"
} >"${manifest}.tmp"
mv "${manifest}.tmp" "${manifest}"
chmod 0444 "${manifest}"

run_stage() {
  local stage_dir="$1"
  local stage="$2"
  shift 2
  local stdout_path="${stage_dir}/${stage}.stdout.json"
  local stderr_path="${stage_dir}/${stage}.stderr.log"
  local status_path="${stage_dir}/${stage}.status.json"
  local exit_code
  set +e
  "$@" >"${stdout_path}.tmp" 2>"${stderr_path}.tmp"
  exit_code=$?
  set -e
  cat "${stdout_path}.tmp"
  cat "${stderr_path}.tmp" >&2
  mv "${stdout_path}.tmp" "${stdout_path}"
  mv "${stderr_path}.tmp" "${stderr_path}"
  printf '{"format_version":1,"stage":"%s","exit_code":%d,"stdout":"%s","stderr":"%s"}\n' \
    "${stage}" "${exit_code}" "$(basename "${stdout_path}")" "$(basename "${stderr_path}")" >"${status_path}.tmp"
  mv "${status_path}.tmp" "${status_path}"
  chmod 0444 "${stdout_path}" "${stderr_path}" "${status_path}"
  return "${exit_code}"
}

replay_args=()
for ((cycle = 1; cycle <= cycles; cycle++)); do
  cycle_replay="${replay_root}/cycle-${cycle}"
  cycle_metrics="${metrics_root}/cycle-${cycle}"
  mkdir -p "${cycle_metrics}"
  actor_checkpoint=(--checkpoint "")
  learner_parent=(--initial-checkpoint "")
  previous_reference="${checkpoint_dir}/previous.json"
  if [[ -f "${checkpoint_dir}/latest.json" ]]; then
    actor_checkpoint=(--checkpoint-latest "${checkpoint_dir}/latest.json")
    learner_parent=(--initial-latest "${checkpoint_dir}/latest.json")
    cp "${checkpoint_dir}/latest.json" "${previous_reference}"
  fi

  run_stage "${cycle_metrics}" selfplay "${selfplay}" \
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

  learner_work=(--steps "${train_steps}")
  if [[ -n "${examples_per_replay_ply}" ]]; then
    learner_work=(--examples-per-replay-ply "${examples_per_replay_ply}")
  fi
  run_stage "${cycle_metrics}" learner "${learner}" \
    "${replay_args[@]}" \
    "${learner_parent[@]}" \
    --model-config "${model_config}" \
    --device "${learner_device}" \
    --torch-threads "${torch_threads}" \
    --output-dir "${checkpoint_dir}" \
    --engine-commit "${engine_commit}" \
    "${learner_work[@]}" \
    --window-shards "${replay_window_shards}" \
    --replay-cache-games "${replay_cache_games}" \
    "${optimizer_args[@]}" \
    --learning-rate "${learning_rate}" \
    --weight-decay "${weight_decay}" \
    --model-seed 0 \
    --seed "${cycle}"

  run_stage "${cycle_metrics}" uniform-evaluation "${evaluator}" \
    --inference "${inference}" \
    --candidate-latest "${checkpoint_dir}/latest.json" \
    --model-config "${model_config}" \
    --inference-device "${inference_device}" \
    --inference-torch-threads "${torch_threads}" \
    --baseline-uniform \
    --output "${evaluation_dir}" \
    --games-per-batch "${arena_games_per_batch}" \
    --cases "${eval_cases}" \
    --simulations "${simulations}" \
    --engine-commit "${engine_commit}"
  if [[ -f "${previous_reference}" ]]; then
    run_stage "${cycle_metrics}" previous-evaluation "${evaluator}" \
      --inference "${inference}" \
      --candidate-latest "${checkpoint_dir}/latest.json" \
      --baseline-latest "${previous_reference}" \
      --model-config "${model_config}" \
      --inference-device "${inference_device}" \
      --inference-torch-threads "${torch_threads}" \
      --output "${evaluation_dir}" \
      --games-per-batch "${arena_games_per_batch}" \
      --cases "${eval_cases}" \
      --simulations "${simulations}" \
      --engine-commit "${engine_commit}"
  fi
done
