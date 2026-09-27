#!/usr/bin/env bash
# test_all.sh - runs Go and Python checks for submission
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
FAILED=0

# pick_python - uses the same interpreter as run.sh when it can import flask
pick_python() {
  local candidate
  for candidate in \
    "${ROOT_DIR}/../_local_Chess_diagram_to_FEN/.venv/bin/python" \
    "${ROOT_DIR}/py_analyser/.venv/bin/python" \
    python3
  do
    if [[ -x "$candidate" || "$candidate" == "python3" ]] && "$candidate" -c "import flask,chess" >/dev/null 2>&1; then
      echo "$candidate"
      return 0
    fi
  done
  echo "python3"
}

PY_BIN="$(pick_python)"

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
  bash -c "cd \"${ROOT_DIR}/py_analyser\" && \"${PY_BIN}\" tester/run_all.py"

echo ""
if [[ "${FAILED}" -ne 0 ]]; then
  echo "test_all: FAILED"
  exit 1
fi
echo "test_all: ALL OK"
