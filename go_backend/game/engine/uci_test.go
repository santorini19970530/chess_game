package engine

import (
	"os"
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

func TestBuildGoCmd_PrefersMoveTime(t *testing.T) {
	fs := &FairyStockfish{}
	got := fs.buildGoCmd(Limit{Depth: 20, MoveTime: 1800 * time.Millisecond})
	want := "go movetime 1800"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = fs.buildGoCmd(Limit{Depth: 8})
	if got != "go depth 8" {
		t.Fatalf("got %q", got)
	}
}
