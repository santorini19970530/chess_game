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
```

Profiles: `beginner` | `intermediate` | `advanced` | `master`.  
Results write-up: FYP repo log sheet `123_chess_formal_ai_vs_ai_evaluation.md`.
