#!/usr/bin/env bash
set -euo pipefail

resolve_script_path() {
  local source_path="$1"
  while [[ -L "${source_path}" ]]; do
    local dir
    dir="$(cd "$(dirname "${source_path}")" && pwd)"
    source_path="$(readlink "${source_path}")"
    [[ "${source_path}" == /* ]] || source_path="${dir}/${source_path}"
  done
  cd "$(dirname "${source_path}")" && pwd
}

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <server-binary>" >&2
  exit 1
fi

SERVER_BIN="$1"
if [[ "${SERVER_BIN}" != /* ]]; then
  SERVER_BIN="$(pwd)/${SERVER_BIN}"
fi
SCRIPT_DIR="$(resolve_script_path "${BASH_SOURCE[0]}")"
DEFAULT_REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
REPO_ROOT="${TM_REPO_ROOT:-${DEFAULT_REPO_ROOT}}"
CLIENT_DIR="${REPO_ROOT}/client"
SERVER_DIR="${REPO_ROOT}/server"

if [[ ! -x "${SERVER_BIN}" ]]; then
  echo "server binary is not executable: ${SERVER_BIN}" >&2
  exit 1
fi

if [[ ! -d "${CLIENT_DIR}" ]]; then
  echo "client directory not found: ${CLIENT_DIR}" >&2
  exit 1
fi

if [[ ! -d "${CLIENT_DIR}/node_modules" ]]; then
  echo "client dependencies are not installed in ${CLIENT_DIR}/node_modules" >&2
  exit 1
fi

cd "${CLIENT_DIR}"

export PATH="/opt/homebrew/bin:/usr/bin:/bin:${PATH:-}"
export CI="${TM_PLAYWRIGHT_CI:-1}"
export TM_PLAYWRIGHT_SERVER_PORT="18080"
export TM_PLAYWRIGHT_CLIENT_PORT="14173"
export TM_PLAYWRIGHT_SERVER_COMMAND="${SERVER_BIN}"
export TM_PLAYWRIGHT_SERVER_CWD="${SERVER_DIR}"
export TM_PLAYWRIGHT_CLIENT_CWD="${CLIENT_DIR}"
export TM_PLAYWRIGHT_CLIENT_COMMAND="./node_modules/.bin/vite --configLoader native --host 127.0.0.1 --port ${TM_PLAYWRIGHT_CLIENT_PORT} --strictPort"
export TM_PLAYWRIGHT_CAPTURE="${TM_PLAYWRIGHT_CAPTURE:-0}"
export PLAYWRIGHT_HTML_OUTPUT_DIR="${TEST_TMPDIR}/playwright-report"
export VITE_BACKEND_PORT="${TM_PLAYWRIGHT_SERVER_PORT}"
export VITE_CACHE_DIR="${TEST_TMPDIR}/vite-cache"

cmd=(
  ./node_modules/.bin/playwright
  test
  --workers=1
  --output="${TEST_TMPDIR}/test-results"
)

if [[ -n "${TM_PLAYWRIGHT_SPEC:-}" ]]; then
  cmd+=("${TM_PLAYWRIGHT_SPEC}")
fi

if [[ -n "${TM_PLAYWRIGHT_GREP:-}" ]]; then
  cmd+=(--grep "${TM_PLAYWRIGHT_GREP}")
fi

exec "${cmd[@]}"
