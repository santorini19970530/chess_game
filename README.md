# UoLCS CM3070 Final Project

Chess game players with Orchestrating AI models

## Project Goal

Build a web app for board game AI play.
Start with Chess.
Add more variants after Chess is stable.
Use short, clear move input.
Support Human vs AI.
Support AI vs AI evaluation.

## System Parts

The system has three parts.

1. Frontend web app.
2. Go backend API and game flow.
3. Python analyst service for text feedback.

## Core Replan Direction

Keep one full vertical slice first.
That slice is Chess end to end.
Then expand to more variants.
Do not expand scope too early.

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
Add analysis button.

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
