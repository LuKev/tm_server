#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 5 ]]; then
  echo "usage: $0 RUNNER SELFPLAY INFERENCE LEARNER EVAL" >&2
  exit 2
fi

runner="$1"
selfplay="$2"
inference="$3"
learner="$4"
evaluator="$5"
run_dir="${TEST_TMPDIR}/az-loop"
TM_AZ_ACTOR_GAMES_PER_BATCH=1 TM_AZ_INFERENCE_MAX_WAIT=0s TM_AZ_REPLAY_CACHE_GAMES=1 TM_AZ_REPLAY_WINDOW_SHARDS=3 "${runner}" "${selfplay}" "${inference}" "${learner}" "${evaluator}" "${run_dir}" test 2 2 1 2 1 debug

test "$(find "${run_dir}/replay" -name 'trajectory-*.json.gz' -type f | wc -l | tr -d ' ')" = 4
test "$(find "${run_dir}/checkpoints" -name 'checkpoint-*.pt' -type f | wc -l | tr -d ' ')" = 2
test "$(find "${run_dir}/evaluation" -name 'evaluation-*.json' -type f | wc -l | tr -d ' ')" = 3
test "$(find "${run_dir}/metrics" -name '*.stdout.json' -type f | wc -l | tr -d ' ')" = 7
test "$(find "${run_dir}/metrics" -name '*.status.json' -type f | wc -l | tr -d ' ')" = 7
test "$(find "${run_dir}/metrics" -name '*.status.json' -type f -exec grep -L '"exit_code":0' {} + | wc -l | tr -d ' ')" = 0
test -r "${run_dir}/run-manifest.txt"
grep -q '^replay_window_shards=3$' "${run_dir}/run-manifest.txt"
grep -q '^inference_max_batch=32$' "${run_dir}/run-manifest.txt"
grep -q '^inference_max_wait=0s$' "${run_dir}/run-manifest.txt"
grep -q '^root_search_mode=concurrent$' "${run_dir}/run-manifest.txt"
grep -q '"configured_root_search_mode":"concurrent"' "${run_dir}/metrics/cycle-1/selfplay.stdout.json"
grep -q '"configured_inference_max_batch":32' "${run_dir}/metrics/cycle-1/selfplay.stdout.json"
grep -q '"configured_inference_max_wait":"0s"' "${run_dir}/metrics/cycle-1/selfplay.stdout.json"
if "${runner}" "${selfplay}" "${inference}" "${learner}" "${evaluator}" "${run_dir}" smoke 1 1 1 1 1 debug; then
  echo "runner unexpectedly reused a nonempty run directory" >&2
  exit 1
fi

failed_run_dir="${TEST_TMPDIR}/failed-run"
if "${runner}" "${selfplay}" "${TEST_TMPDIR}/missing-inference" "${learner}" "${evaluator}" "${failed_run_dir}" smoke 1 1 1 1 1 debug; then
  echo "runner unexpectedly completed a failed stage" >&2
  exit 1
fi
test -s "${failed_run_dir}/metrics/cycle-1/selfplay.stderr.log"
grep -q '"exit_code":[1-9]' "${failed_run_dir}/metrics/cycle-1/selfplay.status.json"
test ! -e "${failed_run_dir}/checkpoints/latest.json"

invalid_optimizer_dir="${TEST_TMPDIR}/invalid-optimizer"
if TM_AZ_OPTIMIZER=typo "${runner}" "${selfplay}" "${inference}" "${learner}" "${evaluator}" "${invalid_optimizer_dir}" smoke 1 1 1 1 1 debug; then
  echo "runner unexpectedly accepted an invalid optimizer" >&2
  exit 1
fi
test ! -e "${invalid_optimizer_dir}"
test -s "${run_dir}/checkpoints/latest.json"
