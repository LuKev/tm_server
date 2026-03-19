#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="${TM_REPO_ROOT:-/Users/kevin/projects/tm_server}"
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
