#!/usr/bin/env bash
# three_model_smoke.sh - one session: diagram → load-fen → FS NNUE analyze → Ollama
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
BASE_URL="${BASE_URL:-http://localhost:8080}"
PY_URL="${PY_ANALYSER_URL:-http://127.0.0.1:8001}"
FIXTURE="${FIXTURE:-$ROOT_DIR/gameplay_capture/chess/chess-08082026.webp}"
EXPLAIN_LOG="${EXPLAIN_LOG:-$ROOT_DIR/py_analyser/data/explain_logs/explain.jsonl}"
NNUE_DEFAULT="$ROOT_DIR/../_local_nnue/nn-3475407dc199.nnue"
CORR="${CORR:-issue0068-c-$(date +%s)}"
POLL_SECS="${POLL_SECS:-90}"

if [[ ! -f "$FIXTURE" ]]; then
  echo "missing fixture: $FIXTURE" >&2
  exit 1
fi

if ! curl -sS -o /dev/null --connect-timeout 2 "$BASE_URL/"; then
  echo "need Go on $BASE_URL (./run.sh)" >&2
  exit 1
fi
if ! curl -sS -o /dev/null --connect-timeout 2 "$PY_URL/"; then
  echo "need Python analyser on $PY_URL (./run.sh)" >&2
  exit 1
fi
if ! curl -fsS -o /dev/null --connect-timeout 2 "http://127.0.0.1:11434/api/tags"; then
  echo "need Ollama on :11434 for pretrained text evidence" >&2
  exit 1
fi

if [[ -z "${FAIRY_STOCKFISH_NNUE_PATH:-}" && -f "$NNUE_DEFAULT" ]]; then
  export FAIRY_STOCKFISH_NNUE_PATH="$NNUE_DEFAULT"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "==> correlation $CORR"
echo "==> POST /api/games (template)"
curl -fsS -X POST "$BASE_URL/api/games" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "type=chess&mode=human_vs_human&humanColor=white&skillLevel=intermediate" \
  -o "$tmp/create.json"
TEMPLATE_ID="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["game"]["id"])' "$tmp/create.json")"
echo "template $TEMPLATE_ID"

echo "==> POST /api/diagram/fen request_id=$CORR"
curl -fsS -X POST "$BASE_URL/api/diagram/fen" \
  -F "image=@$FIXTURE" \
  -F "game=chess" \
  -F "request_id=$CORR" \
  -o "$tmp/diagram.json"
python3 - <<PY
import json, sys
p = json.load(open("$tmp/diagram.json"))
assert p.get("status") == "ok", p
assert p.get("request_id") == "$CORR", p
assert "/" in (p.get("fen") or ""), p
print("diagram fen", p["fen"])
print("diagram request_id", p["request_id"])
PY
FEN="$(python3 -c 'import json; print(json.load(open("'"$tmp"'/diagram.json"))["fen"])')"

echo "==> POST /api/games/$TEMPLATE_ID/load-fen (confirm)"
python3 -c 'import json; print(json.dumps({"fen":"'"$FEN"'","game":"chess"}))' > "$tmp/load.json"
curl -fsS -X POST "$BASE_URL/api/games/${TEMPLATE_ID}/load-fen" \
  -H "Content-Type: application/json" \
  -d @"$tmp/load.json" \
  -o "$tmp/loaded.json"
GAME_ID="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["game"]["id"])' "$tmp/loaded.json")"
LOADED_FEN="$(python3 -c 'import json; print(json.load(open("'"$tmp"'/loaded.json"))["game"]["config"]["startFen"])')"
python3 - <<PY
assert "$GAME_ID" != "$TEMPLATE_ID", "load-fen must create a new session"
assert "$LOADED_FEN" == "$FEN", ("fen mismatch", "$LOADED_FEN", "$FEN")
print("confirmed game", "$GAME_ID")
PY

ANALYZE_ID="${GAME_ID}-move-1"
EXPLAIN_ID="${GAME_ID}-explain-1"

echo "==> poll GET /api/games/$GAME_ID/analysis/latest"
python3 - <<PY
import json, time, urllib.request
url = "$BASE_URL/api/games/$GAME_ID/analysis/latest"
deadline = time.time() + $POLL_SECS
last = {}
while time.time() < deadline:
    last = json.load(urllib.request.urlopen(url, timeout=10))
    latest = last.get("latest") or {}
    analysis = latest.get("analysis") or {}
    if (
        not last.get("pending")
        and analysis.get("evaluation_source") == "fairy-stockfish"
        and analysis.get("request_id") == "$ANALYZE_ID"
    ):
        json.dump(last, open("$tmp/analysis.json", "w"), indent=2)
        print("analysis request_id", analysis.get("request_id"))
        print("evaluation_source", analysis.get("evaluation_source"))
        print("best_move_uci", analysis.get("best_move_uci"))
        print("eval_cp_white", analysis.get("eval_cp_white"))
        raise SystemExit(0)
    time.sleep(1)
print(json.dumps(last, indent=2)[:2000])
raise SystemExit("analysis did not become fairy-stockfish $ANALYZE_ID")
PY

echo "==> GET /api/games/$GAME_ID/top-moves (Go legal-filtered)"
curl -fsS "$BASE_URL/api/games/${GAME_ID}/top-moves?k=3" -o "$tmp/top.json" || true

echo "==> poll explain.jsonl $EXPLAIN_ID"
python3 - <<PY
import json, time
from pathlib import Path
path = Path("$EXPLAIN_LOG")
deadline = time.time() + $POLL_SECS
hit = None
while time.time() < deadline:
    if path.is_file():
        for line in path.read_text(encoding="utf-8").splitlines():
            if not line.strip():
                continue
            row = json.loads(line)
            if row.get("request_id") == "$EXPLAIN_ID":
                hit = row
                if row.get("source") == "ollama" and (row.get("explanation") or "").strip():
                    json.dump(row, open("$tmp/explain.json", "w"), indent=2)
                    print("explain request_id", row.get("request_id"))
                    print("explain source", row.get("source"))
                    print("explain text", (row.get("explanation") or "").strip()[:240])
                    raise SystemExit(0)
    time.sleep(1)
print(json.dumps(hit, indent=2)[:2000] if hit else "no explain line yet")
raise SystemExit("ollama explain $EXPLAIN_ID not found")
PY

echo "==> UCI eval probe on the same FEN (NNUE evidence)"
cd "$ROOT_DIR/py_analyser"
python3 - <<PY
import json
from fs_engine import nnue_evidence_check, nnue_eval_probe
nnue_evidence_check()
mode, lines = nnue_eval_probe("chess", "$FEN")
assert mode == "nnue", (mode, lines)
key = [ln for ln in lines if "NNUE evaluation using" in ln or ln.startswith("Final evaluation")]
json.dump({"mode": mode, "key_lines": key}, open("$tmp/nnue_probe.json", "w"), indent=2)
print("eval_mode", mode)
for ln in key:
    print(ln)
PY

python3 - <<PY
import json
from pathlib import Path
tmp = Path("$tmp")
diagram = json.loads((tmp / "diagram.json").read_text())
analysis = json.loads((tmp / "analysis.json").read_text())
explain = json.loads((tmp / "explain.json").read_text())
probe = json.loads((tmp / "nnue_probe.json").read_text())
top = {}
if (tmp / "top.json").is_file():
    top = json.loads((tmp / "top.json").read_text())
a = (analysis.get("latest") or {}).get("analysis") or {}
out = {
    "correlation_id": "$CORR",
    "template_id": "$TEMPLATE_ID",
    "game_id": "$GAME_ID",
    "analyze_request_id": "$ANALYZE_ID",
    "explain_request_id": "$EXPLAIN_ID",
    "fen": "$FEN",
    "diagram_request_id": diagram.get("request_id"),
    "evaluation_source": a.get("evaluation_source"),
    "analyze_source": a.get("source"),
    "best_move_uci": a.get("best_move_uci"),
    "eval_cp_white": a.get("eval_cp_white"),
    "threat_summary": a.get("threat_summary"),
    "nnue_eval_mode": probe.get("mode"),
    "nnue_key_lines": probe.get("key_lines"),
    "explain_source": explain.get("source"),
    "explanation": (explain.get("explanation") or "").strip(),
    "top_moves": top.get("suggestions") or top,
}
print("==> PART C EVIDENCE")
print(json.dumps(out, indent=2))
(tmp / "evidence.json").write_text(json.dumps(out, indent=2) + "\n")
PY

echo "ok three-model workflow $CORR -> $GAME_ID"
