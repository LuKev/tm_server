#!/usr/bin/env bash
set -euo pipefail
runner="$1"
selfplay="$2"
inference="$3"
learner="$4"
evaluator="$5"
run_dir="${TEST_TMPDIR}/az-loop"
export TM_AZ_ACTOR_GAMES_PER_BATCH=1 TM_AZ_INFERENCE_MAX_WAIT=0s TM_AZ_REPLAY_CACHE_GAMES=1 TM_AZ_REPLAY_WINDOW_SHARDS=3
export TM_AZ_EVAL_BASELINES=uniform,previous
"${runner}" --mode new "${selfplay}" "${inference}" "${learner}" "${evaluator}" "${run_dir}" test 2 2 1 2 1 debug
test "$(find "${run_dir}/replay" -name 'trajectory-*.json.gz' -type f | wc -l | tr -d ' ')" = 4
test "$(find "${run_dir}/checkpoints" -name 'checkpoint-*.pt' -type f | wc -l | tr -d ' ')" = 2
test "$(find "${run_dir}/evaluation" -name 'evaluation-*.json' -type f | wc -l | tr -d ' ')" = 3
test "$(find "${run_dir}" -name stage.json -type f | wc -l | tr -d ' ')" = 7
grep -q '"configured_root_search_mode":"concurrent"' "${run_dir}/replay/cycle-1/stdout.json"
grep -q '"configured_inference_max_batch":32' "${run_dir}/replay/cycle-1/stdout.json"
grep -q '"configured_inference_max_wait":"0s"' "${run_dir}/replay/cycle-1/stdout.json"
if "${runner}" --mode new "${selfplay}" "${inference}" "${learner}" "${evaluator}" "${run_dir}" test 2 2 1 2 1 debug; then
  echo "runner unexpectedly reused nonempty directory" >&2
  exit 1
fi
restored="${TEST_TMPDIR}/restored"
cp -R "${run_dir}" "${restored}"
"${runner}" --mode resume "${selfplay}" "${inference}" "${learner}" "${evaluator}" "${restored}" test 2 2 1 2 1 debug
diff -r "${run_dir}" "${restored}"
if "${runner}" --mode resume "${selfplay}" "${inference}" "${learner}" "${evaluator}" "${restored}" test 2 2 2 2 1 debug; then
  echo "runner accepted changed configuration" >&2
  exit 1
fi
test -s "${restored}/checkpoints/latest.json"
