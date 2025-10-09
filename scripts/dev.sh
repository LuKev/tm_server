#!/usr/bin/env bash
set -euo pipefail

# Run Terra Mystica server (Bazel) and client (Vite) concurrently.
# Usage: ./scripts/dev.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Ensure line-buffered output if available (GNU coreutils)
STDBUF_CMD=""
if command -v stdbuf >/dev/null 2>&1; then
  STDBUF_CMD="stdbuf -oL -eL"
fi

cleanup() {
  echo "[dev] Shutting down..."
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
  fi
  if [[ -n "${CLIENT_PID:-}" ]] && kill -0 "$CLIENT_PID" 2>/dev/null; then
    kill "$CLIENT_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

(
  cd "$ROOT_DIR/server"
  echo "[server] starting: bazel run //cmd/server:server"
  exec $STDBUF_CMD bazel run //cmd/server:server
) | sed -u 's/^/[server] /' &
SERVER_PID=$!

(
  cd "$ROOT_DIR/client"
  echo "[client] starting: npm run dev"
  exec $STDBUF_CMD npm run dev
) | sed -u 's/^/[client] /' &
CLIENT_PID=$!

echo "[dev] server PID: $SERVER_PID"
echo "[dev] client PID: $CLIENT_PID"
echo "[dev] Press Ctrl+C to stop both."

wait
