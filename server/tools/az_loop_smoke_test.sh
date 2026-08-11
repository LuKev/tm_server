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
TM_AZ_ACTOR_GAMES_PER_BATCH=1 TM_AZ_REPLAY_CACHE_GAMES=1 "${runner}" "${selfplay}" "${inference}" "${learner}" "${evaluator}" "${run_dir}" test 2 2 1 2 1 debug

test "$(find "${run_dir}/replay" -name 'trajectory-*.json.gz' -type f | wc -l | tr -d ' ')" = 4
test "$(find "${run_dir}/checkpoints" -name 'checkpoint-*.pt' -type f | wc -l | tr -d ' ')" = 2
test "$(find "${run_dir}/evaluation" -name 'evaluation-*.json' -type f | wc -l | tr -d ' ')" = 3
test -s "${run_dir}/checkpoints/latest.json"
