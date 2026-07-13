# Chess formal evaluation — 2026-07-12

Issue0033 evidence folder for AI-vs-AI pairing summaries.

## Layout

- `*.json` here — batch summaries from `cmd/match` (keep for the report).
- Raw per-game archives stay under `go_backend/data/simulations/<run>-Ngames/` (gitignored).

## Required matrix

1. **Same-level baselines** (expect ~balanced W/D/L): beg vs beg, int vs int, mas vs mas  
2. **Cross-strength** (both colours): beg↔int, beg↔mas, int↔mas  

## Results so far

| File | White | Black | W / B / D | Avg moves | Notes |
|------|-------|-------|-----------|-----------|-------|
| `eval_beg_vs_beg.json` | beginner | beginner | 18 / 29 / 3 | 89.4 | Done (baseline; some first-move bias) |
| `eval_int_vs_int.json` | intermediate | intermediate | 27 / 18 / 5 | 138.1 | Done (baseline; some first-move bias) |
| `eval_mas_vs_mas.json` | master | master | 3 / 11 / 36 | 146.9 | Done (many draws — expected at equal strength) |
| `eval_beg_vs_int.json` | beginner | intermediate | 1 / 46 / 3 | 80.8 | Done |
| `eval_int_vs_beg.json` | intermediate | beginner | 50 / 0 / 0 | 70.2 | Done |
| `eval_beg_vs_mas.json` | beginner | master | 0 / 49 / 1 | 47.5 | Done |
| `eval_mas_vs_beg.json` | master | beginner | 50 / 0 / 0 | 44.0 | Done |
| `eval_int_vs_mas.json` | intermediate | master | 0 / 50 / 0 | 73.2 | Done |
| `eval_mas_vs_int.json` | master | intermediate | 50 / 0 / 0 | 65.4 | Done (re-run OK) |

## Status

**All 9 pairings done.** Next for issue0033: one summary table for the report + short log sheet.

## Full command set (run from `go_backend`)

```bash
cd go_backend
OUT=data/evaluations/2026-07-12
mkdir -p "$OUT"

# Same-level baselines
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -profile beginner -format json > "$OUT/eval_beg_vs_beg.json"
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -profile intermediate -format json > "$OUT/eval_int_vs_int.json"
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -profile master -format json > "$OUT/eval_mas_vs_mas.json"

# Easy vs Medium (both colours)
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -white-profile beginner -black-profile intermediate -format json > "$OUT/eval_beg_vs_int.json"
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -white-profile intermediate -black-profile beginner -format json > "$OUT/eval_int_vs_beg.json"

# Easy vs Hard
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -white-profile beginner -black-profile master -format json > "$OUT/eval_beg_vs_mas.json"
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -white-profile master -black-profile beginner -format json > "$OUT/eval_mas_vs_beg.json"

# Medium vs Hard
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -white-profile intermediate -black-profile master -format json > "$OUT/eval_int_vs_mas.json"
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 50 -white-profile master -black-profile intermediate -format json > "$OUT/eval_mas_vs_int.json"
```
