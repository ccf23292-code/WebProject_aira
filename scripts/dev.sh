#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

kill_port() {
  local port="$1"
  local pids=""
  if command -v lsof >/dev/null 2>&1; then
    pids="$(lsof -ti tcp:"${port}" || true)"
  fi
  if [ -n "${pids}" ]; then
    echo "Killing processes on port ${port}: ${pids}"
    kill -9 ${pids} || true
  fi
}

start_backend() {
  kill_port 3001
  "$ROOT_DIR/scripts/dev-backend.sh" &
  BACK_PID=$!
  echo "Backend started (PID ${BACK_PID}). Logs follow in this terminal."
}

start_frontend() {
  kill_port 3000
  "$ROOT_DIR/scripts/dev-frontend.sh" &
  FRONT_PID=$!
  echo "Frontend started (PID ${FRONT_PID}). Logs follow in this terminal."
}

mode="${1:-all}"
case "${mode}" in
  all)
    start_backend
    start_frontend
    ;;
  backend)
    start_backend
    ;;
  frontend)
    start_frontend
    ;;
  *)
    echo "Usage: $(basename "$0") [all|backend|frontend]"
    exit 1
    ;;
esac

wait
