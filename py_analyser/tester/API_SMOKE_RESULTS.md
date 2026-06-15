# API Smoke Results

Smoke verification for:
- `POST /history`
- `POST /policy`
- `POST /value`

Server:
- URL: `http://127.0.0.1:8001`
- Service: `final_project/chess_game/py_analyser/server.py`

## 1) `/history`

Request:

```bash
curl -sS -X POST "http://127.0.0.1:8001/history" \
  -H "Content-Type: application/json" \
  -d '{"request_id":"smoke-history-1","game_id":"game-smoke-1","game_type":"chess","variant":"chess","fen":"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1","color":"white","move_number":1,"move_history":[]}'
```

Response:

```json
{
  "features": {
    "is_check": false,
    "is_checkmate": false,
    "is_stalemate": false,
    "material_delta_cp": 0,
    "move_count": 0
  },
  "latency_ms": 0,
  "phase": "opening",
  "request_id": "smoke-history-1",
  "source": "rule_based_v1",
  "status": "ok",
  "tags": [
    "balanced",
    "book_like"
  ]
}
```

## 2) `/policy`

Request:

```bash
curl -sS -X POST "http://127.0.0.1:8001/policy" \
  -H "Content-Type: application/json" \
  -d '{"request_id":"smoke-policy-1","game_id":"game-smoke-1","game_type":"chess","variant":"chess","fen":"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1","color":"white","move_number":1,"move_history":[],"top_k":3}'
```

Response:

```json
{
  "best_move_uci": "e2e3",
  "candidates": [
    {
      "prob": 0.337748,
      "rank": 1,
      "san": "e3",
      "score_cp": 20,
      "uci": "e2e3"
    },
    {
      "prob": 0.337748,
      "rank": 2,
      "san": "e4",
      "score_cp": 20,
      "uci": "e2e4"
    },
    {
      "prob": 0.324504,
      "rank": 3,
      "san": "d4",
      "score_cp": 16,
      "uci": "d2d4"
    }
  ],
  "latency_ms": 1,
  "request_id": "smoke-policy-1",
  "source": "heuristic",
  "status": "ok"
}
```

## 3) `/value`

Request:

```bash
curl -sS -X POST "http://127.0.0.1:8001/value" \
  -H "Content-Type: application/json" \
  -d '{"request_id":"smoke-value-1","game_id":"game-smoke-1","game_type":"chess","variant":"chess","fen":"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1","color":"white","move_number":1,"move_history":[]}'
```

Response:

```json
{
  "latency_ms": 0,
  "mate_in": 0,
  "request_id": "smoke-value-1",
  "score_cp": 0,
  "source": "heuristic",
  "status": "ok",
  "value": 0.0,
  "win_chance_black": 0.5,
  "win_chance_white": 0.5
}
```

## Result

All three smoke checks returned HTTP 200 with expected schema fields.
