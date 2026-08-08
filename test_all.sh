#!/usr/bin/env bash
# test_all.sh - runs Go, Python, and FE smoke checks for submission
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

run_step "Go backend (go test ./...)" \
  bash -c "cd \"${ROOT_DIR}/go_backend\" && go test ./... -count=1"

run_step "Python analyzer (unittest discover)" \
  bash -c "cd \"${ROOT_DIR}/py_analyser\" && python3 tester/run_all.py"

run_step "FE JS syntax (node --check)" \
  bash -c "cd \"${ROOT_DIR}/frontend/scripts\" && node --check chess_command.js && for f in js_parts/*.js; do node --check \"\$f\"; done"

run_step "FE board geometry self-check" \
  node "${ROOT_DIR}/frontend/scripts/js_parts/board.js"

run_step "FE move history self-check" \
  node "${ROOT_DIR}/frontend/scripts/js_parts/move_history.js"

run_step "FE hints coach self-check" \
  node "${ROOT_DIR}/frontend/scripts/js_parts/hints_coach.js"

echo ""
if [[ "${FAILED}" -ne 0 ]]; then
  echo "test_all: FAILED"
  exit 1
fi
echo "test_all: ALL OK"
