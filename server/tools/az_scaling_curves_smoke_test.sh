#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 4 ]]; then
  echo "usage: $0 CURVE_RUNNER MODEL_BENCHMARK EVAL INFERENCE" >&2
  exit 2
fi

output="${TEST_TMPDIR}/curves"
if TM_AZ_CURVE_MODEL_CONFIG=debug TM_AZ_CURVE_BATCH_SIZES=1 TM_AZ_CURVE_SEARCH_BUDGETS=1 \
  "$1" "$2" "$3" "$4" "" "${TEST_TMPDIR}/must-refuse" test-curve cpu 1; then
  echo "scaling curve accepted a missing production checkpoint" >&2
  exit 1
fi
TM_AZ_CURVE_MODEL_CONFIG=debug \
TM_AZ_CURVE_BATCH_SIZES=1 \
TM_AZ_CURVE_SEARCH_BUDGETS=1 \
TM_AZ_CURVE_WARMUP=0 \
TM_AZ_CURVE_ITERATIONS=1 \
TM_AZ_CURVE_ALLOW_RANDOM=1 \
"$1" "$2" "$3" "$4" "" "${output}" test-curve cpu 1

batch_report="${output}/model-batches/batch-1.json"
search_report=("${output}"/search-reports/evaluation-*.json)
search_summary="${output}/search-summaries/candidate-1-baseline-1.json"
[[ -s "${batch_report}" && -s "${search_report[0]}" && -s "${search_summary}" ]]
grep -q '"model_config": "debug"' "${batch_report}"
grep -q '"format_version":5' "${search_report[0]}"
grep -q '"holdout_suite_id":"tm-v0-paired-v2-prefix-1"' "${search_report[0]}"
grep -q '"candidate_simulations":1' "${search_report[0]}"
grep -q '"baseline_simulations":1' "${search_report[0]}"
