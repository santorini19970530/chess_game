# AI API Contract (Issue0016)

This document defines the stable request/response schemas for Step 3 Issue0016:
- `POST /history`
- `POST /policy`
- `POST /value`

These endpoints belong to the Python AI service. They are API endpoints (machine-to-machine), not page routes.

## Goals

- Keep schemas stable for backend integration (Issue0017).
- Support offline execution with local engine/model providers.
- Ensure backend can handle both success and failure in a predictable way.

## Common Rules

- Content type: `application/json`.
- Character encoding: UTF-8.
- Timeouts:
  - client timeout should be configurable from backend env.
  - service should return fast validation errors for bad payloads.
- Unknown extra fields in request: ignored.
- Unknown fields in response: must not be required by backend.

## Shared Request Fields

All endpoints use this common envelope:

```json
{
  "request_id": "game-123-move-8",
  "game_id": "game-123",
  "game_type": "chess",
  "variant": "chess",
  "fen": "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 2",
  "color": "white",
  "move_number": 8,
  "move_history": ["e2e4", "e7e5"]
}
```

Field notes:
- `request_id` (string, required): traceable request identifier.
- `game_id` (string, optional): optional session id for logs.
- `game_type` (string, required): `chess` | `xiangqi` | `shogi`.
- `variant` (string, optional): engine-specific variant label; defaults from `game_type`.
- `fen` (string, required): current position.
- `color` (string, required): side to evaluate/move (`white` or `black`).
- `move_number` (integer, optional): ply/move counter.
- `move_history` (array[string], optional): normalized move list.

## Error Response (All Endpoints)

On non-200, use:

```json
{
  "request_id": "game-123-move-8",
  "status": "error",
  "error_kind": "validation",
  "message": "fen is required"
}
```

`error_kind` values:
- `validation`
- `timeout`
- `unavailable`
- `internal`

## POST /history

Purpose:
- Return game context features for decision-layer weighting.

Request:
- Uses shared request fields.

Success response:

```json
{
  "request_id": "game-123-move-8",
  "status": "ok",
  "source": "rule_based_v1",
  "phase": "opening",
  "features": {
    "is_check": false,
    "is_checkmate": false,
    "is_stalemate": false,
    "material_delta_cp": 0,
    "move_count": 8
  },
  "tags": ["book_like", "balanced"],
  "latency_ms": 6
}
```

Contract notes:
- `phase` required: `opening` | `middlegame` | `endgame`.
- `features` required object; unknown keys allowed.
- `latency_ms` required integer.

## POST /policy

Purpose:
- Return ranked legal move candidates with confidence-like weights.

Request:
- Shared fields plus:

```json
{
  "top_k": 5
}
```

`top_k` rules:
- optional; default `5`.
- clamp to `[1, 20]`.

Success response:

```json
{
  "request_id": "game-123-move-8",
  "status": "ok",
  "source": "fairy_stockfish",
  "best_move_uci": "g1f3",
  "candidates": [
    { "rank": 1, "uci": "g1f3", "san": "Nf3", "score_cp": 32, "prob": 0.44 },
    { "rank": 2, "uci": "f1c4", "san": "Bc4", "score_cp": 25, "prob": 0.29 },
    { "rank": 3, "uci": "d2d4", "san": "d4",  "score_cp": 18, "prob": 0.17 }
  ],
  "latency_ms": 43
}
```

Contract notes:
- `candidates` required array (can be empty only on terminal position).
- `rank` starts at 1 and increases strictly.
- `prob` in `[0, 1]`; backend must not assume exact sum of `1.0`.

## POST /value

Purpose:
- Return scalar position evaluation used by decision/risk control.

Request:
- Uses shared request fields.

Success response:

```json
{
  "request_id": "game-123-move-8",
  "status": "ok",
  "source": "fairy_stockfish",
  "score_cp": 32,
  "mate_in": 0,
  "value": 0.11,
  "win_chance_white": 0.55,
  "win_chance_black": 0.45,
  "latency_ms": 29
}
```

Contract notes:
- `score_cp` required integer (from white POV).
- `mate_in` required integer (`0` means no forced mate detected).
- `value` required float in `[-1, 1]`.
- `win_chance_white` and `win_chance_black` required floats in `[0, 1]`.

## Backend Route Inventory (Current)

The backend currently contains three route categories:

- Page/static routes (not API): `/`, `/styles/*`, `/scripts/*`, `/pic/*`, `/sounds/*`.
- WebSocket API route: `/ws/game`.
- HTTP API routes:
  - Primary REST API: `/api/games`, `/api/games/{id}`, `/api/games/{id}/move`, `/api/games/{id}/config`, `/api/games/{id}/flag`, `/api/games/{id}/new`, `/api/games/{id}/legal-moves`, `/api/games/{id}/analysis/latest`.
  - Legacy compatibility API: `/command`, `/game/new`, `/game/flag`, `/game/config`, `/game/legal-moves`, `/game/analysis/latest`.

So yes, most of those backend routed handlers are APIs. The root page and static assets are web routes, not data APIs.
