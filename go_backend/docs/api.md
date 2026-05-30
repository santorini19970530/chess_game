# Chess REST API

Base URL: `http://localhost:8080`

Primary endpoints:

- `POST /api/games`
- `GET /api/games/:id`
- `POST /api/games/:id/move`

Legacy endpoints under `/command` and `/game/*` remain available temporarily as compatibility wrappers and are deprecated.

## Error format

All REST API errors return JSON:

```json
{
  "status": "error",
  "message": "human-readable message"
}
```

## Create Game

`POST /api/games`

Form fields (all optional):

- `type` (`chess` default)
- `mode` (`human_vs_human` default)
- `humanColor` (`white` default)
- `aiGameCount` (`1` default)
- `fen` (optional starting FEN)

Example:

```bash
curl -X POST http://localhost:8080/api/games \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "type=chess&mode=human_vs_human&humanColor=white&aiGameCount=1&fen="
```

Success response includes:

- `game.id`
- `currentTurn`
- `checkedSide`
- `history`, `historyDetailed`
- `state`, `captured`

## Get Game

`GET /api/games/:id`

Example:

```bash
curl http://localhost:8080/api/games/<game_id>
```

Returns the latest snapshot for that game id.

## Apply Move

`POST /api/games/:id/move`

Form fields:

- `command` (required, e.g. `e2e4`)

Example:

```bash
curl -X POST http://localhost:8080/api/games/<game_id>/move \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "command=e2e4"
```

Returns updated snapshot including:

- normalized `command`
- `from`, `to`
- `game`, `history`, `state`, `captured`

## Smoke test script

You can run:

```bash
bash scripts/api_games_smoke_test.sh
```

This verifies create/get/move on the 3 REST endpoints.
