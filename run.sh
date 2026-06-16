#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"

TAILWIND="$ROOT_DIR/frontend/styles/tailwindcss"
INPUT_CSS="$ROOT_DIR/frontend/styles/input.css"
OUTPUT_CSS="$ROOT_DIR/frontend/styles/style.css"
PY_SERVER="$ROOT_DIR/py_analyser/server.py"
GO_DIR="$ROOT_DIR/go_backend"
PY_PID=""
GO_PID=""

"$TAILWIND" -i "$INPUT_CSS" -o "$OUTPUT_CSS"

if lsof -t -nP -iTCP:8080 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "port 8080 is already in use; stop that process first."
  echo "example: lsof -t -nP -iTCP:8080 -sTCP:LISTEN | xargs kill"
  exit 1
fi

if lsof -t -nP -iTCP:8001 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "port 8001 is already in use; stop that process first."
  echo "example: lsof -t -nP -iTCP:8001 -sTCP:LISTEN | xargs kill"
  exit 1
fi

echo "starting python analyzer server on http://127.0.0.1:8001 ..."
python3 "$PY_SERVER" &
PY_PID=$!

cleanup() {
  trap - EXIT INT TERM

  if [[ -n "$GO_PID" ]] && kill -0 "$GO_PID" >/dev/null 2>&1; then
    kill "$GO_PID" >/dev/null 2>&1 || true
  fi
  if kill -0 "$PY_PID" >/dev/null 2>&1; then
    kill "$PY_PID" >/dev/null 2>&1 || true
  fi

  if [[ -n "$GO_PID" ]]; then
    wait "$GO_PID" 2>/dev/null || true
  fi
  if [[ -n "$PY_PID" ]]; then
    wait "$PY_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

sleep 1
if ! kill -0 "$PY_PID" >/dev/null 2>&1; then
  echo "python analyzer failed to start."
  exit 1
fi

echo "starting go backend on http://localhost:8080 ..."
(
  cd "$GO_DIR"
  go build .
  # Enable Fairy-Stockfish UCI path for FS top-3 hints (issue0024) and related features.
  # Set to false/omit this line to fall back to the Python heuristic analyzer only.
  USE_FAIRY_STOCKFISH=true \
  PY_ANALYSER_URL="${PY_ANALYSER_URL:-http://127.0.0.1:8001}" \
  go run .
) &
GO_PID=$!

sleep 1
if ! kill -0 "$GO_PID" >/dev/null 2>&1; then
  echo "go backend failed to start."
  exit 1
fi

echo "python analyzer pid: $PY_PID"
echo "go backend pid: $GO_PID"
echo "both services started. press Ctrl+C to stop both."

while true; do
  if ! kill -0 "$PY_PID" >/dev/null 2>&1; then
    echo "python analyzer exited, shutting down go backend..."
    break
  fi
  if ! kill -0 "$GO_PID" >/dev/null 2>&1; then
    echo "go backend exited, shutting down python analyzer..."
    break
  fi
  sleep 1
done
