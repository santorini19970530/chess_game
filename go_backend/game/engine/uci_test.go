package engine

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestFairyStockfish_StartsAndResponds(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		t.Skip("FAIRY_STOCKFISH_PATH not set; skipping UCI integration test")
	}

	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("NewFairyStockfish failed: %v", err)
	}

	if err := fs.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer fs.Close()

	if err := fs.IsReady(); err != nil {
		t.Fatalf("IsReady failed: %v", err)
	}
}

func TestFairyStockfish_BestMoveReturnsLegalMove(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		t.Skip("FAIRY_STOCKFISH_PATH not set; skipping UCI integration test")
	}

	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("NewFairyStockfish failed: %v", err)
	}
	if err := fs.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer fs.Close()

	move, err := fs.BestMove("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", Limit{Depth: 5})
	if err != nil {
		t.Fatalf("BestMove failed: %v", err)
	}
	if move == "" {
		t.Fatalf("BestMove returned empty move")
	}
}

func TestFairyStockfish_SetStrengthProfile(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		t.Skip("FAIRY_STOCKFISH_PATH not set; skipping UCI integration test")
	}

	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("NewFairyStockfish failed: %v", err)
	}
	if err := fs.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer fs.Close()

	for _, p := range []string{"beginner", "intermediate", "advanced", "master"} {
		if err := fs.SetStrengthProfile(p); err != nil {
			t.Fatalf("SetStrengthProfile(%s) failed: %v", p, err)
		}
	}
}

func TestFairyStockfish_Restart(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		t.Skip("FAIRY_STOCKFISH_PATH not set; skipping UCI integration test")
	}

	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("NewFairyStockfish failed: %v", err)
	}
	if err := fs.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := fs.Restart(); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}
	defer fs.Close()

	if err := fs.IsReady(); err != nil {
		t.Fatalf("IsReady after restart failed: %v", err)
	}
}

func TestFairyStockfish_TopKReturnsLegalMoves(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		t.Skip("FAIRY_STOCKFISH_PATH not set; skipping UCI integration test")
	}

	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("NewFairyStockfish failed: %v", err)
	}
	if err := fs.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer fs.Close()

	results, err := fs.TopK("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", 3, Limit{Depth: 8})
	if err != nil {
		t.Fatalf("TopK failed: %v", err)
	}
	if len(results) == 0 || len(results) > 3 {
		t.Fatalf("expected 1-3 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Move == "" {
			t.Fatalf("TopK returned empty move")
		}
	}
}

func TestFairyStockfish_TopKRespectsProfile(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		t.Skip("FAIRY_STOCKFISH_PATH not set; skipping UCI integration test")
	}

	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("NewFairyStockfish failed: %v", err)
	}
	if err := fs.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer fs.Close()

	for _, p := range []string{"beginner", "master"} {
		if err := fs.SetStrengthProfile(p); err != nil {
			t.Fatalf("SetStrengthProfile(%s) failed: %v", p, err)
		}
		results, err := fs.TopK("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", 3, Limit{Depth: 8})
		if err != nil {
			t.Fatalf("TopK with profile %s failed: %v", p, err)
		}
		if len(results) == 0 {
			t.Fatalf("TopK with profile %s returned no results", p)
		}
	}
}

func TestFairyStockfish_LegalMovesXiangqiStart(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		bin = "../../../py_analyser/Fairy-Stockfish-fairy_sf_14/src/stockfish"
		if _, err := os.Stat(bin); err != nil {
			t.Skip("FAIRY_STOCKFISH_PATH not set and default binary missing; skipping")
		}
	}
	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("NewFairyStockfish: %v", err)
	}
	if err := fs.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer fs.Close()
	if err := fs.SetVariant("xianqi"); err != nil {
		t.Fatalf("SetVariant: %v", err)
	}
	const xiangqiStart = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
	moves, err := fs.LegalMoves(xiangqiStart)
	if err != nil {
		t.Fatalf("LegalMoves: %v", err)
	}
	if len(moves) < 40 {
		t.Fatalf("expected ~44 start moves, got %d: %v", len(moves), moves)
	}
	found := false
	for _, m := range moves {
		if m == "a4a5" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("a4a5 missing from legal moves: %v", moves)
	}
	newFEN, err := fs.FENAfterMove(xiangqiStart, "a4a5")
	if err != nil {
		t.Fatalf("FENAfterMove: %v", err)
	}
	if newFEN == xiangqiStart || !strings.Contains(newFEN, " b ") {
		t.Fatalf("unexpected fen after a4a5: %q", newFEN)
	}
}

func TestUCIVariantName(t *testing.T) {
	cases := map[string]string{
		"chess":   "chess",
		"xianqi":  "xiangqi",
		"xiangqi": "xiangqi",
		"Xianqi":  "xiangqi",
		"shogi":   "shogi",
		"":        "chess",
	}
	for in, want := range cases {
		if got := UCIVariantName(in); got != want {
			t.Fatalf("UCIVariantName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestFairyStockfish_SetVariantXiangqiBestMove(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		// go test cwd is this package dir (game/engine/).
		bin = "../../../py_analyser/Fairy-Stockfish-fairy_sf_14/src/stockfish"
		if _, err := os.Stat(bin); err != nil {
			t.Skip("FAIRY_STOCKFISH_PATH not set and default binary missing; skipping")
		}
	}

	fs, err := NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("NewFairyStockfish failed: %v", err)
	}
	if err := fs.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer fs.Close()

	if err := fs.SetVariant("xianqi"); err != nil {
		t.Fatalf("SetVariant(xianqi) failed: %v", err)
	}
	if fs.Variant() != "xiangqi" {
		t.Fatalf("Variant()=%q want xiangqi", fs.Variant())
	}

	const xiangqiStart = "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
	move, err := fs.BestMove(xiangqiStart, Limit{Depth: 5, MoveTime: 500 * time.Millisecond})
	if err != nil {
		t.Fatalf("BestMove on xiangqi start failed: %v", err)
	}
	if move == "" || move == "(none)" {
		t.Fatalf("BestMove returned empty/none: %q", move)
	}

	// Switching back to chess must work on the same process.
	if err := fs.SetVariant("chess"); err != nil {
		t.Fatalf("SetVariant(chess) failed: %v", err)
	}
	chessMove, err := fs.BestMove("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", Limit{Depth: 5, MoveTime: 500 * time.Millisecond})
	if err != nil {
		t.Fatalf("BestMove on chess after variant switch failed: %v", err)
	}
	if chessMove == "" {
		t.Fatal("chess BestMove empty after variant switch")
	}
}

