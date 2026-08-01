package handlers

import (
	"testing"

	sessionpkg "go_backend/game/session"
)

// TestChooseBestLegalCandidate_PicksFirstLegal - picks the first policy candidate present in the legal set
func TestChooseBestLegalCandidate_PicksFirstLegal(t *testing.T) {
	legal := map[string]struct{}{
		"e2e4": {},
		"d2d4": {},
	}
	candidates := []AIPolicyCandidate{
		{Rank: 1, UCI: "g1f3"},
		{Rank: 2, UCI: "E2E4"},
		{Rank: 3, UCI: "d2d4"},
	}
	got := chooseBestLegalCandidate(candidates, legal)
	if got != "e2e4" {
		t.Fatalf("expected e2e4, got %q", got)
	}
}

// TestRunAIGame_CompletesWithTerminalResult - runs one ai-vs-ai game to a win or draw with moves played
func TestRunAIGame_CompletesWithTerminalResult(t *testing.T) {
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeAIVsAI, sessionpkg.GameTypeChess, "white", 1, "", "beginner")
	if err != nil {
		t.Fatalf("create ai_vs_ai: %v", err)
	}
	res, err := RunAIGame(game.ID)
	if err != nil {
		t.Fatalf("RunAIGame: %v", err)
	}
	if res.Result != sessionpkg.GameResultWhiteWin &&
		res.Result != sessionpkg.GameResultBlackWin &&
		res.Result != sessionpkg.GameResultDraw {
		t.Fatalf("bad result %q", res.Result)
	}
	if res.MoveCount <= 0 {
		t.Fatalf("moveCount %d", res.MoveCount)
	}
}
