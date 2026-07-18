# Variant parity + smoke checklist (issue0038)

Human-readable gate for Chess / Xiangqi / Shogi. **Automated tests stay with the adapters** (issue0034 / issue0036 and UI follow-ups); this doc does not replace them.

Game IDs: `"chess"`, `"xianqi"`, `"shogi"`.

---

## Parity table

| Capability | Chess | Xiangqi (`xianqi`) | Shogi (`shogi`) |
|------------|-------|--------------------|-----------------|
| HvH (UI + API) | Yes | Yes | Yes |
| HvAI (FS propose → Go legal apply) | Yes | Yes | Yes |
| AI vs AI (`POST /api/simulate`, `cmd/match`) | Yes | Yes | Yes |
| Legal highlights (board click) | Yes | Yes | Yes (+ hand drop highlights) |
| FS top-move hints (Shift / refresh) | Yes (pale cyan wash + corner rank) | Yes (same model; deep blue/amber wash) | Yes (same; deep blue/amber; drop → hand chip) |
| FS strength profiles | Yes | Yes (`UCI_Variant`) | Yes (`UCI_Variant`) |
| Explain / LLM coach | Yes (`/analyze` + `/explain`) | Yes (FS UCI analyze; variant-safe explain) | Yes (same) |
| Win% bar | `cp_to_win_chance` from White-POV eval | Same mapping; White-POV FS eval (log 130) | Same |
| Captured / hands panel | Piece **icons** + counts | Piece **icons** + counts | **Hands** icons (drop inventory) |
| Promotion UX | Chess picker Q/R/B/N | N/A | Must → auto `+`; optional → Promote / Do not |
| Drops | N/A | N/A | Yes (`P*e5`, hand UI) |
| Board UI | 8×8 squares | Point board (junctions) | 9×9 wood squares |
| Automated tests present? | Yes | Yes | Yes |
| Known gaps | — | Terms/tone → issue0051; HPV chess-only | *Uchifuzume* MVP skip; terms → issue0051; HPV chess-only |

---

## Where automated tests live

Do not relocate these into this optional docs issue.

| Area | Paths / commands |
|------|------------------|
| Xiangqi rules + session | `go_backend/game/movement/xiangqi_*`, `go_backend/game/session/xiangqi_*_test.go` → `go test ./game/session/ ./game/movement/ -run Xiangqi` |
| Shogi rules + session | `go_backend/game/movement/shogi_*`, `go_backend/game/session/shogi_*_test.go` → `go test ./game/session/ ./game/movement/ -run Shogi` |
| HTTP / UCI gate | `go_backend/handlers/move_squares_test.go`, `api_games_test.go` → `go test ./handlers/ -run 'Xiangqi\|Shogi\|ParseVariant'` |
| Piece assets on disk | `go_backend/handlers/piece_assets_test.go` → `go test ./handlers/ -run PieceAssets` |
| Variant coach pipe | `py_analyser/tester/test_variant_analyze.py`, `test_variant_explain.py`; `handlers/analyzer_client_test.go` (`PassesXiangqiGameType`) — log [129](../../report/log_sheets/stage_7_xiangqi_and_shogi/129_variant_analyze_explain_pipe.md) |
| Coach UX follow-ups | win% White-POV + icons — log [130](../../report/log_sheets/stage_7_xiangqi_and_shogi/130_variant_coach_ux_followups.md); hint/legal CSS — log [131](../../report/log_sheets/stage_7_xiangqi_and_shogi/131_hint_highlight_css_fixes.md) |
| Match smoke batches (optional) | `cmd/match -game xianqi\|shogi` → eval JSON under `go_backend/data/evaluations/` (logs 124/125) |

UI: [126](../../report/log_sheets/stage_7_xiangqi_and_shogi/126_xiangqi_ui_board.md) Xiangqi, [127](../../report/log_sheets/stage_7_xiangqi_and_shogi/127_shogi_ui_board.md) Shogi. Coach: [129](../../report/log_sheets/stage_7_xiangqi_and_shogi/129_variant_analyze_explain_pipe.md)–[131](../../report/log_sheets/stage_7_xiangqi_and_shogi/131_hint_highlight_css_fixes.md).

---

## Manual smoke checklists

Operator ticks. Prefer a hard refresh after switching variants.

### Chess

1. [ ] New Game (Chess) → 8×8 + chess pieces  
2. [ ] Legal: `e2e4` accepts; illegal `e2e5` rejected  
3. [ ] Click/drag: select piece → legal squares → move  
4. [ ] HvAI: one human move → AI reply  
5. [ ] Notes/coach may update after moves (when analyzer up)  
6. [ ] Captured panel shows piece icons (not unicode-only) after captures  
7. [ ] AI vs AI: Run Simulation (`game=chess`, N≥1)  
8. [ ] Switch to Xiangqi or Shogi → New Game → board rebuilds (not stuck on 8×8 chess art)

### Xiangqi

1. [ ] Select Xiangqi → Apply Setup / New Game → point board + `xianqi_pic` pieces  
2. [ ] Legal: `i4i5` and `a4a5` accept; history icon matches piece  
3. [ ] Illegal move rejected with status message  
4. [ ] After Chess → New Game Xiangqi: captured panel empty (no ♛/♝ leftovers)  
5. [ ] HvAI: one human move → AI reply; no Python `expected 8 rows` spam  
6. [ ] Notes/coach update after a move (win% / explain); win% does not invert every black ply  
7. [ ] FS hints: **deep blue/amber square wash** + corner rank (not pale cyan); UCI ranks like `h10` OK; select piece → black legal dots still show  
8. [ ] Capture → **icon** chips in captured panel  
9. [ ] AI vs AI: Run Simulation with Xiangqi  
10. [ ] Switch back to Chess → New Game → 8×8 + chess pieces  

### Shogi

1. [ ] Select Shogi → Apply Setup / New Game → 9×9 wood + SVG pieces  
2. [ ] Legal board move (e.g. `c3c4`); history icon matches piece  
3. [ ] Capture → hand **icon** chip; click chip → drop on lit square (`P*…`)  
4. [ ] Optional promote in zone: dialog; must-promote last rank: auto `+`  
5. [ ] HvAI: one human move → AI reply; no Python FEN spam  
6. [ ] Notes/coach update after a move; win% stable White-POV mapping  
7. [ ] FS hints: **deep blue/amber wash** + corner rank; drop hint lights hand chip + dashed dest; select piece → black legal dots (no black dot on origin)  
8. [ ] AI vs AI: Run Simulation with Shogi  
9. [ ] Switch Chess / Xiangqi → New Game → geometry + pieces correct  

---

## Related issues / logs

| Issue / log | Role |
|-------------|------|
| issue0034 / 0035 | Xiangqi backend + UI |
| issue0036 / 0037 | Shogi backend + UI |
| issue0038 | This doc (parity + smoke) |
| issue0039 / 0040 | Real shared clock (not claimed here) |
| issue0049 + log 129 | Variant `/analyze` + `/explain` pipe — **Done** |
| log 130 | Coach UX follow-ups (icons, win% POV, first hint pass) — **Done** |
| log 131 | Hint/legal CSS: shared wash model, rank/`::before` fix, drop/relife — **Done** |
| issue0051 | Variant terms/tone JSON (after chess teacher pattern) |
