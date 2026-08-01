// CM3070 FP code
// ai_decision_test.go - tests for ai decision

package handlers

import (
	"testing"

	sessionpkg "go_backend/game/session"
)

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
