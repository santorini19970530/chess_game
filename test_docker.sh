#!/usr/bin/env bash
# test_docker.sh - runs Go and Python tests in throwaway images; not compose, and it does not build Fairy-Stockfish
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
FAILED=0

# run_step - runs a labeled command and tracks failure without aborting early
run_step() {
  local label="$1"
  shift
  echo ""
  echo "=== ${label} ==="
  if "$@"; then
    echo "OK: ${label}"
  else
    echo "FAIL: ${label}"
    FAILED=1
  fi
}

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found. Install Docker, or on a host with Go and Python run ./test_all.sh" >&2
  exit 1
fi

run_step "Go backend (go test ./...)" \
  docker run --rm \
    -v "${ROOT_DIR}/go_backend":/src -w /src \
    golang:1.22-bookworm \
    go test ./... -count=1

run_step "Python analyzer (unittest discover)" \
  docker run --rm \
    -v "${ROOT_DIR}/py_analyser":/src -w /src \
    python:3.12-bookworm-slim \
    bash -c "pip install -q -r requirements.txt && python tester/run_all.py"

echo ""
if [[ "${FAILED}" -ne 0 ]]; then
  echo "test_docker: FAILED"
  exit 1
fi
echo "test_docker: ALL OK"
