# UoLCS CM3070 Final Project

Chess game players with Orchestrating AI models

## Project Goal

1. Build a web app for board game AI play.
2. Start with Chess.
3. Add more variants after Chess is stable.
4. Use short, clear move input.
5. Support Human vs AI.
6. Support AI vs AI evaluation.

## System Parts

1. Frontend web app.
2. Go backend API and game flow.
3. Python analyst service for text feedback.

## Main Phases

### Phase 1: Literature and evaluation design

Write literature with critical comparison.
Define metrics early.
Use win rate, game length, and latency.

### Phase 2: Architecture and API freeze

Freeze API contracts.
Freeze data flow between services.
Keep frontend thin.

### Phase 3: Chess vertical slice

Create game.
Accept human move.
Return AI move.
Show game status.
Auto analysis after every successful move.

### Phase 4: Variant expansion

Add next variant with same backend pattern.
Keep feature parity minimal first.
Stabilize before adding more.

### Phase 5: Quality and polish

Tune easy, medium, and hard profiles.
Improve UI clarity.
Improve analyst response quality.

### Phase 6: Testing and instrumentation

Add API tests.
Add move-flow tests.
Log per-game metadata for evaluation.

### Phase 7: Evaluation

Run AI vs AI tournaments.
Produce tables and charts.
Write short result analysis.

### Phase 8: Final documentation

Finalize report.
Finalize README and run steps.
Prepare demo assets.

## Weekly Rule

End each week with three outputs.
A runnable build.
One evaluation artifact.
One documentation update.

## Scope Rules

Finish Chess first.
Prefer fewer complete variants.
Keep analyst output simple if needed.

## Formal AI-vs-AI eval (`cmd/match`)

From `go_backend` (Fairy-Stockfish required; Ollama not required):

```bash
cd go_backend
OUT=data/evaluations/YYYY-MM-DD
mkdir -p "$OUT"

USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -profile intermediate -format json > "$OUT/eval_int_vs_int.json"
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -white-profile beginner -black-profile master -format json > "$OUT/eval_beg_vs_mas.json"
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 5 -game xianqi -profile beginner -format json > "$OUT/eval_xianqi_smoke.json"
```

Profiles: `beginner` | `intermediate` | `advanced` | `master`.  
Results write-up: FYP repo log sheet `123_chess_formal_ai_vs_ai_evaluation.md`.

## Xiangqi (`game=xianqi`) — backend notes

Session type ID is `xianqi` (stable). Fairy-Stockfish UCI variant name is `xiangqi`.

**Rules vs AI:** Go movement strategies own legality, apply-move, legal-move lists, and terminal detection (checkmate / stalemate-as-loss). Fairy-Stockfish is used only for AI search (`UCI_Variant=xiangqi` + strength profiles), same split as Chess.

**Board coords:** files `a`–`i` (1–9), ranks `1`–`10` (Red/White at ranks 1–3).

**Move codec (API / history / FS):** UCI-like `fromto` with no promotion suffix, e.g. `a4a5`, `h3h10`. Rank `10` is two digits in the string (`h3h10`).

**Start FEN:** `rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1`  
Chess FEN (8 ranks) is rejected when `game=xianqi`.

**Simulate / match:** `POST /api/simulate` and `cmd/match -game xianqi` accept Xiangqi (aliases `xiangqi` / `xianqi`).
