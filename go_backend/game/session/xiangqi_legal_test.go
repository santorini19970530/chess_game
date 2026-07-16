package session

import (
	"testing"
)

func TestXiangqiLegalMoves_AllAndPerSquare(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	all, err := AllLegalUCIMovesByID(game.ID)
	if err != nil {
		t.Fatalf("AllLegalUCIMovesByID: %v", err)
	}
	if len(all) < 40 {
		t.Fatalf("expected ~44 start moves, got %d", len(all))
	}
	foundA4A5 := false
	for _, mv := range all {
		if mv == "a4a5" {
			foundA4A5 = true
			break
		}
	}
	if !foundA4A5 {
		t.Fatalf("a4a5 missing from all legal moves: %v", all)
	}

	dests, err := LegalMovesForSquareByID(game.ID, 1, 4)
	if err != nil {
		t.Fatalf("LegalMovesForSquareByID: %v", err)
	}
	foundA5 := false
	for _, d := range dests {
		if d.File == 1 && d.Rank == 5 {
			foundA5 = true
		}
	}
	if !foundA5 {
		t.Fatalf("expected destination a5 from a4, got %+v", dests)
	}

	empty, err := LegalMovesForSquareByID(game.ID, 1, 5)
	if err != nil {
		t.Fatalf("empty square: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty square should have no moves, got %+v", empty)
	}
}
