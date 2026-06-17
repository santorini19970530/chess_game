#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

if [[ "${1:-}" == "--help" ]]; then
  echo "Usage: BASE_URL=http://localhost:8080 bash scripts/api_games_smoke_test.sh"
  echo "Runs smoke tests for:"
  echo "  POST /api/games"
  echo "  GET  /api/games/:id"
  echo "  POST /api/games/:id/move"
  echo "  POST /api/simulate"
  exit 0
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

create_json="$tmp_dir/create.json"
get_json="$tmp_dir/get.json"
move_json="$tmp_dir/move.json"

echo "==> POST /api/games"
curl -fsS -X POST "${BASE_URL}/api/games" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "type=chess&mode=human_vs_human&humanColor=white&aiGameCount=1&fen=" \
  -o "$create_json"

game_id="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["game"]["id"])' "$create_json")"
if [[ -z "${game_id}" ]]; then
  echo "Smoke test failed: missing game id in create response"
  exit 1
fi
echo "Created game id: ${game_id}"

echo "==> GET /api/games/${game_id}"
curl -fsS "${BASE_URL}/api/games/${game_id}" -o "$get_json"
get_id="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["game"]["id"])' "$get_json")"
if [[ "${get_id}" != "${game_id}" ]]; then
  echo "Smoke test failed: GET id mismatch (${get_id} != ${game_id})"
  exit 1
fi
echo "GET returned matching game id."

echo "==> POST /api/games/${game_id}/move (e2e4)"
curl -fsS -X POST "${BASE_URL}/api/games/${game_id}/move" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "command=e2e4" \
  -o "$move_json"

move_cmd="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["command"])' "$move_json")"
if [[ "${move_cmd}" != "e2e4" ]]; then
  echo "Smoke test failed: move command mismatch (${move_cmd} != e2e4)"
  exit 1
fi
echo "Move accepted."

simulate_json="$tmp_dir/simulate.json"
echo "==> POST /api/simulate"
curl -fsS -X POST "${BASE_URL}/api/simulate" \
  -H "Content-Type: application/json" \
  -d '{"games":1,"profile":"beginner"}' \
  -o "$simulate_json"

sim_games="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["games"])' "$simulate_json")"
if [[ "$sim_games" != "1" ]]; then
  echo "Smoke test failed: simulate games count mismatch"
  exit 1
fi
echo "Simulate endpoint returned valid summary."

echo
echo "Smoke test passed for all /api/games endpoints."
