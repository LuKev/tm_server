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

SCRIPT_DIR="$(resolve_script_path "${BASH_SOURCE[0]}")"
DEFAULT_REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
REPO_ROOT="${TM_REPO_ROOT:-${DEFAULT_REPO_ROOT}}"
CLIENT_DIR="${REPO_ROOT}/client"

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
export VITE_CACHE_DIR="${TEST_TMPDIR}/vite-cache"
WORK_DIR="${TEST_TMPDIR}/client-build"
mkdir -p "${WORK_DIR}" "${TEST_TMPDIR}/tsbuildinfo"

tar -C "${CLIENT_DIR}" \
  --exclude=node_modules \
  --exclude=dist \
  -cf - . | tar -C "${WORK_DIR}" -xf -

ln -s "${CLIENT_DIR}/node_modules" "${WORK_DIR}/node_modules"

perl -0pi -e "s|\./node_modules/\\.tmp/tsconfig\\.app\\.tsbuildinfo|${TEST_TMPDIR}/tsbuildinfo/tsconfig.app.tsbuildinfo|g" "${WORK_DIR}/tsconfig.app.json"
perl -0pi -e "s|\./node_modules/\\.tmp/tsconfig\\.node\\.tsbuildinfo|${TEST_TMPDIR}/tsbuildinfo/tsconfig.node.tsbuildinfo|g" "${WORK_DIR}/tsconfig.node.json"

cd "${WORK_DIR}"
./node_modules/.bin/tsc -b
./node_modules/.bin/vite build --configLoader native
