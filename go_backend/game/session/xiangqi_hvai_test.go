package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go_backend/game/engine"
)

func TestXiangqiHumanVsAI_OneMoveCycle(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		// go test cwd is this package dir (game/session/).
		bin = filepath.Join("..", "..", "..", "py_analyser", "Fairy-Stockfish-fairy_sf_14", "src", "stockfish")
		if _, err := os.Stat(bin); err != nil {
			t.Skip("FAIRY_STOCKFISH_PATH not set and default binary missing; skipping")
		}
	}

	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsAI, GameTypeXiangqi, "white", 1, "", "beginner")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := ApplyMoveByCommandByID(game.ID, "a4a5"); err != nil {
		t.Fatalf("human move: %v", err)
	}
	fenAfterHuman, err := CurrentFENByID(game.ID)
	if err != nil {
		t.Fatalf("fen after human: %v", err)
	}
	if !strings.Contains(fenAfterHuman, " b ") {
		t.Fatalf("expected black to move, fen=%q", fenAfterHuman)
	}

	legal, err := AllLegalUCIMovesByID(game.ID)
	if err != nil {
		t.Fatalf("legal: %v", err)
	}
	if len(legal) == 0 {
		t.Fatal("expected legal moves for black")
	}
	legalSet := make(map[string]struct{}, len(legal))
	for _, mv := range legal {
		legalSet[strings.ToLower(mv)] = struct{}{}
	}

	fs, err := engine.NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if err := fs.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer fs.Close()
	if err := fs.SetVariant("xianqi"); err != nil {
		t.Fatalf("SetVariant: %v", err)
	}
	aiMove, err := fs.BestMove(fenAfterHuman, engine.Limit{Depth: 5, MoveTime: 500 * time.Millisecond})
	if err != nil {
		t.Fatalf("BestMove: %v", err)
	}
	aiMove = strings.ToLower(strings.TrimSpace(aiMove))
	if _, ok := legalSet[aiMove]; !ok {
		t.Fatalf("FS move %q not in Go legal set (%d moves)", aiMove, len(legal))
	}
	if _, err := ApplyMoveByCommandByID(game.ID, aiMove); err != nil {
		t.Fatalf("apply AI move %q: %v", aiMove, err)
	}

	fenAfterAI, err := CurrentFENByID(game.ID)
	if err != nil {
		t.Fatalf("fen after AI: %v", err)
	}
	if fenAfterAI == fenAfterHuman {
		t.Fatal("FEN unchanged after AI move")
	}
	if !strings.Contains(fenAfterAI, " w ") {
		t.Fatalf("expected white to move after AI, fen=%q", fenAfterAI)
	}
	snap, err := BuildSnapshotByID(game.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snap.History) != 2 {
		t.Fatalf("history len=%d want 2", len(snap.History))
	}
}
