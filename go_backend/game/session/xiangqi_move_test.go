package session

import (
	"strings"
	"testing"
)

func TestXiangqiHumanMove_LegalUpdatesFENAndHistory(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	normalized, err := ApplyMoveByCommandByID(game.ID, "a4a5")
	if err != nil {
		t.Fatalf("legal move a4a5: %v", err)
	}
	if normalized != "a4a5" {
		t.Fatalf("normalized=%q", normalized)
	}

	fen, err := CurrentFENByID(game.ID)
	if err != nil {
		t.Fatalf("fen: %v", err)
	}
	if fen == DefaultXiangqiStartFEN {
		t.Fatal("FEN should change after move")
	}
	if !strings.Contains(fen, " b ") {
		t.Fatalf("expected black to move after white move, fen=%q", fen)
	}

	snap, err := BuildSnapshotByID(game.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snap.History) != 1 {
		t.Fatalf("history len=%d want 1", len(snap.History))
	}
	if snap.CurrentTurn != "Black" {
		t.Fatalf("turn=%q want Black", snap.CurrentTurn)
	}
	foundA5 := false
	for _, p := range snap.State {
		if p.File == 1 && p.Rank == 5 && p.Kind == "pawn" && p.Color == "white" {
			foundA5 = true
		}
	}
	if !foundA5 {
		t.Fatalf("expected white pawn on a5 in state: %+v", snap.State)
	}
}

func TestXiangqiHumanMove_IllegalRejected(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = ApplyMoveByCommandByID(game.ID, "a4a6") // soldier cannot jump two
	if err == nil {
		t.Fatal("expected illegal move to be rejected")
	}
	fen, err := CurrentFENByID(game.ID)
	if err != nil {
		t.Fatalf("fen: %v", err)
	}
	if fen != DefaultXiangqiStartFEN {
		t.Fatalf("FEN should be unchanged after illegal move, got %q", fen)
	}
}

func TestXiangqiHumanMove_Rank10Notation(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// White cannon on h3 can capture along file to black rook? h3h10 captures through screen.
	// From start: h3 cannon, screen at h7? Board: black pawns on rank 7, empty rank 9, rook on h10.
	// Cannon needs exactly one screen — pawn on h7 is the screen, so h3h10 is a capture of rook.
	if _, err := ApplyMoveByCommandByID(game.ID, "h3h10"); err != nil {
		t.Fatalf("h3h10 should be legal: %v", err)
	}
}
