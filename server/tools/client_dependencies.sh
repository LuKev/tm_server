#!/usr/bin/env bash
set -euo pipefail
# Explicit dependency maintenance through the repository's Bazel workflow.
REPO_ROOT="${TM_REPO_ROOT:-${BUILD_WORKSPACE_DIRECTORY}/..}"
cd "${REPO_ROOT}/client"
export PATH="/opt/homebrew/bin:/usr/bin:/bin:${PATH:-}"
exec npm "$@"
