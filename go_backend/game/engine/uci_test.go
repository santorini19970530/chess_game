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
