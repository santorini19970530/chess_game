# Backend Route Inventory

This file lists all routes currently registered in `go_backend/router.go`.

## Route Types

- **Page/static routes**: serve HTML, JS, CSS, images, sounds (not data APIs).
- **HTTP API routes**: JSON/form endpoints for game state and actions.
- **WebSocket API route**: realtime push channel for game events.

## 1) Page and Static Routes (Non-API)

| Method | Path | Handler / Source | API? | Notes |
|---|---|---|---|---|
| GET | `/` | `h.Index` | No | Main web page route |
| GET | `/styles/style.css` | inline handler -> `cssbuild.EnsureStyleCSS` | No | Builds + serves compiled stylesheet |
| GET | `/styles/input.css` | `serveNoCache(inputCSSPath)` | No | Static CSS |
| GET | `/scripts/chess_command.js` | `serveNoCache(commandScriptPath)` | No | Frontend script |
| GET | `/favicon.ico` | inline handler | No | Optional icon |
| GET | `/icon.png` | inline handler | No | Optional icon |
| GET | `/pic/*` | `http.FileServer` | No | Piece images |
| GET | `/sounds/*` | `http.FileServer` | No | Sound assets |

## 2) WebSocket API Route

| Method | Path | Handler | API? | Notes |
|---|---|---|---|---|
| GET (upgrade) | `/ws/game` | `h.GameSocket` | Yes (WS) | Realtime events per game/session |

## 3) Primary REST API Routes

| Method | Path | Handler | API? | Notes |
|---|---|---|---|---|
| POST | `/api/games` | `h.APIGames` | Yes | Create game session |
| GET | `/api/games/{gameId}` | `h.APIGameRoutes` -> `getAPIGameByID` | Yes | Fetch game snapshot |
| POST | `/api/games/{gameId}/move` | `h.APIGameRoutes` -> `postAPIGameMove` | Yes | Apply move |
| POST | `/api/games/{gameId}/config` | `h.APIGameRoutes` -> `postAPIGameConfig` | Yes | Update mode/type/setup |
| POST | `/api/games/{gameId}/flag` | `h.APIGameRoutes` -> `postAPIGameFlag` | Yes | Flag/resign current side |
| POST | `/api/games/{gameId}/new` | `h.APIGameRoutes` -> `postAPIGameNew` | Yes | Start new game from config |
| GET | `/api/games/{gameId}/legal-moves?file={1..8}&rank={1..8}` | `h.APIGameRoutes` -> `getAPIGameLegalMoves` | Yes | Legal moves for a square |
| GET | `/api/games/{gameId}/analysis/latest` | `h.APIGameRoutes` -> `getAPIGameLatestAnalysis` | Yes | Latest async analysis status/result |

## 4) Legacy Compatibility API Routes (Deprecated)

These are still active for compatibility while clients migrate to `/api/games/*`.

| Method | Path | Handler | API? | Notes |
|---|---|---|---|---|
| POST | `/command` | `h.SubmitChessCommand` | Yes (legacy) | Old move endpoint |
| POST | `/game/new` | `h.NewGame` | Yes (legacy) | Old new game endpoint |
| POST | `/game/flag` | `h.FlagGame` | Yes (legacy) | Old flag endpoint |
| POST | `/game/config` | `h.UpdateGameConfig` | Yes (legacy) | Old config endpoint |
| GET | `/game/legal-moves` | `h.GetLegalMoves` | Yes (legacy) | Old legal moves endpoint |
| GET | `/game/analysis/latest` | `h.GetLatestAnalysis` | Yes (legacy) | Old analysis status endpoint |

## Notes

- `APIGameRoutes` is a path dispatcher for subroutes under `/api/games/{gameId}/...`.
- Legacy and primary API routes currently overlap in capability; use primary REST routes for new work.
- Python AI routes (`/history`, `/policy`, `/value`) are part of the Python service contract and documented in `AI_API_CONTRACT.md`.
