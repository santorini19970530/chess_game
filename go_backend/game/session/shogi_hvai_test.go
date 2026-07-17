package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go_backend/game/engine"
)

func TestShogiHumanVsAI_OneMoveCycle(t *testing.T) {
	bin := os.Getenv("FAIRY_STOCKFISH_PATH")
	if bin == "" {
		bin = filepath.Join("..", "..", "..", "py_analyser", "Fairy-Stockfish-fairy_sf_14", "src", "stockfish")
		if _, err := os.Stat(bin); err != nil {
			t.Skip("FAIRY_STOCKFISH_PATH not set and default binary missing; skipping")
		}
	}

	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsAI, GameTypeShogi, "white", 1, "", "beginner")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := ApplyMoveByCommandByID(game.ID, "c3c4"); err != nil {
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
		legalSet[normalizeShogiUCI(mv)] = struct{}{}
	}

	fs, err := engine.NewFairyStockfish(bin)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if err := fs.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer fs.Close()
	if err := fs.SetVariant("shogi"); err != nil {
		t.Fatalf("SetVariant: %v", err)
	}
	aiMove, err := fs.BestMove(fenAfterHuman, engine.Limit{Depth: 5, MoveTime: 500 * time.Millisecond})
	if err != nil {
		t.Fatalf("BestMove: %v", err)
	}
	aiMove = normalizeShogiUCI(aiMove)
	if aiMove == "" || aiMove == "(none)" {
		t.Fatalf("empty AI move: %q", aiMove)
	}
	if _, ok := legalSet[aiMove]; !ok {
		// Engine drop notation may use @ instead of *.
		alt := strings.Replace(aiMove, "@", "*", 1)
		if _, ok2 := legalSet[alt]; ok2 {
			aiMove = alt
		} else {
			n := 8
			if len(legal) < n {
				n = len(legal)
			}
			t.Fatalf("engine move %q not in Go legal set (%d moves); sample=%v", aiMove, len(legal), legal[:n])
		}
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

func normalizeShogiUCI(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// Drops: P*e5 / p@e5 → P*e5
	if len(s) >= 4 && (s[1] == '*' || s[1] == '@') {
		return strings.ToUpper(s[:1]) + "*" + strings.ToLower(s[2:])
	}
	return strings.ToLower(s)
}
