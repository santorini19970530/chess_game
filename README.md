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

Coach pipe for Xiangqi/Shogi (`/analyze` + `/explain`) is Done (issue0049; logs 129–130), including FS hints, captured icons, and White-POV win%.

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
OUT=data/evaluations/2026-07-17
mkdir -p "$OUT"

USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 5 -game xianqi -profile beginner -format json > "$OUT/eval_xianqi_smoke.json"
USE_FAIRY_STOCKFISH=true go run ./cmd/match -games 5 -game shogi -profile beginner -format json > "$OUT/eval_shogi_smoke.json"
```

Profiles: `beginner` | `intermediate` | `advanced` | `master`.  
Results write-up: FYP repo log sheet `123_chess_formal_ai_vs_ai_evaluation.md`.

---

## UI board rendering (Chess / Xiangqi / Shogi)

The playable board is a **CSS grid of div squares**, not a board image.

- Server markup: `go_backend/game/board/board.go` builds `.chess_board_wrapper` → `.chess_board` → `.chess_board_square` with `data-sequence`.
- Client: `frontend/scripts/chess_command.js` maps `file`/`rank` → sequence, paints pieces into those squares, and reuses legal/suggested square classes.
- Style: `input.css` imports per-game board sheets — `chessboard.css` / `xianqiboard.css` / `shogiboard.css` (active via `data-game-type`); Tailwind builds one `style.css`.
- **Piece placement:** Chess/Shogi = inside the square. Xiangqi = on junctions; line layer uses real spacing (`x=i/8`, `y=j/9`), not 9×10 cell centers.
- Client rebuild: `chess_command.js` `ensureBoardGeometry` / `rebuildBoardGrid` when `game.type` changes; sequence = `(maxRank - rank) * files + (file - 1)`.
- Assets: piece art only (`pic/chess_pic/` PNGs, `pic/xianqi_pic/` PNGs, `pic/shogi_pic/` SVGs). Do not use a full-board picture for layout.

Xiangqi: API kinds → `xianqi_pic` (e.g. king→`general_*`, elephant→`bear_*`). Shogi: kinds → `shogi_pic/{kind}.svg` (black via CSS rotate); hands from snapshot `captured`; drops `P*e5`; optional promote dialog / must-promote auto `+`. Rules stay in Go; the board divs only display state and collect moves.

---

## How to read FEN / make moves (Xiangqi + Shogi)

Both variants use the **same coordinate style as this API**:

| Axis | Meaning |
|------|---------|
| **File** | column letter `a` … `i` (left → right) |
| **Rank** | row number (bottom → top from **White/Red/Sente**) |
| **API square** | `file` 1–9 = `a`–`i`, `rank` = number |
| **Move string** | `fromSquare` + `toSquare`, e.g. `c3c4` = from c3 to c4 |

**FEN placement rule (both games):** the text before the first space is the board. Ranks are separated by `/`.  
**First segment = highest rank** (Black/Gote back rank). **Last segment = rank 1** (White/Red/Sente back rank).  
Digits = empty squares in a row. Uppercase = White/Red/Sente; lowercase = Black/Gote.

---

## Xiangqi (`game=xianqi`)

Session ID: `xianqi`. Go owns rules; Fairy-Stockfish is AI search only (`UCI_Variant=xiangqi`).

**Start FEN:**

```text
rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1
```

### Board map (start) — ranks 10 → 1

```text
rank 10  r n b a k a b n r     ← Black back (FEN first segment)
rank  9  . . . . . . . . .
rank  8  . c . . . . . c .
rank  7  p . p . p . p . p
rank  6  . . . . . . . . .     ← river
rank  5  . . . . . . . . .
rank  4  P . P . P . P . P
rank  3  . C . . . . . C .
rank  2  . . . . . . . . .
rank  1  R N B A K A B N R     ← Red/White back (FEN last segment)
         a b c d e f g h i
```

(`.` = empty; palace is files `d`–`f`, ranks 1–3 and 8–10.)

### Piece letters

| Letter | Piece |
|--------|--------|
| K/k | General |
| A/a | Advisor |
| B/b | Elephant |
| N/n | Horse |
| R/r | Chariot |
| C/c | Cannon |
| P/p | Soldier |

### How to form a move

1. Find the piece’s square: file letter + rank number (e.g. leftmost Red soldier is **a4**).
2. Find destination the same way (one step forward → **a5**).
3. Send **`a4a5`**. Rank 10 uses two digits: cannon on **h3** capturing up the file → **`h3h10`**.

Examples from start:

| Idea | Move |
|------|------|
| Red soldier a4 → a5 | `a4a5` |
| Red cannon h3 → h7 (need screen) / h10 capture | `h3h10` (legal at start) |
| Black to move after Red | FEN has ` b ` instead of ` w ` |

**Simulate / match:** `POST /api/simulate` / `cmd/match -game xianqi` (alias `xiangqi`).

---

## Shogi (`game=shogi`)

Session ID: `shogi`. Go owns rules (including hands/drops); Fairy-Stockfish is AI search only (`UCI_Variant=shogi`).

**Start FEN** (hands in `[]` after the board; empty at start):

```text
lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[] w - - 0 1
```

### Board map (start) — ranks 9 → 1

```text
rank 9  l n s g k g s n l     ← Gote/Black back
rank 8  . r . . . . . b .
rank 7  p p p p p p p p p
rank 6  . . . . . . . . .
rank 5  . . . . . . . . .
rank 4  . . . . . . . . .
rank 3  P P P P P P P P P
rank 2  . B . . . . . R .
rank 1  L N S G K G S N L     ← Sente/White back
        a b c d e f g h i
```

Promotion zone: White ranks **7–9**, Black ranks **1–3**.

### Piece letters

| Letter | Piece | Promoted |
|--------|--------|----------|
| K/k | King | — |
| G/g | Gold | — |
| S/s | Silver | `+S` → gold-like |
| N/n | Knight | `+N` |
| L/l | Lance | `+L` |
| P/p | Pawn | `+P` (tokin) |
| B/b | Bishop | `+B` (horse) |
| R/r | Rook | `+R` (dragon) |

Hands field `[Ppg]` = White has Pawn; Black has Pawn and Gold (uppercase = White’s hand, lowercase = Black’s).

### How to form a move

**Board move:** same as Xiangqi — `from` + `to` on `a`–`i` / `1`–`9`.

| Idea | Move |
|------|------|
| Sente pawn c3 → c4 | `c3c4` |
| Promote (optional `+`, forced on last ranks for P/L/N) | `e8e9` becomes `e8e9+` |
| Drop pawn from hand onto e5 | `P*e5` or `p*e5` (also `@` accepted) |

**Relife:** capture → piece goes to your **hand** (unpromoted); later **drop** with `P*e5`.

Snapshot field `captured` for shogi = **hands** (White/Black counts).

Examples:

1. Read FEN segment for rank 3: `PPPPPPPPP` → pawns on a3…i3.  
2. Move the c-file pawn forward → **`c3c4`**.  
3. After you capture a pawn, hand shows `pawn: 1`; drop with **`P*e5`**.

**Simulate / match:** `POST /api/simulate` / `cmd/match -game shogi`.

### Quick API create

```json
{ "mode": "human_vs_human", "game": "shogi", "humanColor": "white" }
```

```json
{ "mode": "human_vs_human", "game": "xianqi", "humanColor": "white" }
```

Move body uses the same command string as above (`c3c4`, `a4a5`, `P*e5`, …).
