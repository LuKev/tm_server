#!/usr/bin/env bash
set -euo pipefail
# Run the custom-domain router's existing tools through the Bazel workflow.
REPO_ROOT="${TM_REPO_ROOT:-${BUILD_WORKSPACE_DIRECTORY}/..}"
cd "${REPO_ROOT}/cloudflare-worker/tm-router"
export PATH="/opt/homebrew/bin:/usr/bin:/bin:${PATH:-}"
exec npm "$@"
