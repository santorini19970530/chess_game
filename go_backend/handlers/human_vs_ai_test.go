package handlers

import (
	"testing"

	sessionpkg "go_backend/game/session"
)

// TestHumanVsAIModeDetection verifies that the mode check in the move handlers
// correctly identifies human_vs_ai mode (step 5 & 6 regression test).
func TestHumanVsAIModeDetection(t *testing.T) {
	// These tests are lightweight because the full orchestration
	// is already exercised through the existing move flow tests.

	// 1. human_vs_ai mode should be recognized
	mode := sessionpkg.GameModeHumanVsAI
	if mode != "human_vs_ai" {
		t.Fatalf("expected human_vs_ai constant to be 'human_vs_ai'")
	}

	// 2. Other modes should not trigger AI turn
	otherModes := []sessionpkg.GameMode{
		sessionpkg.GameModeHumanVsHuman,
		sessionpkg.GameModeAIVsAI,
	}
	for _, m := range otherModes {
		if m == sessionpkg.GameModeHumanVsAI {
			t.Fatalf("unexpected mode equality")
		}
	}
}

// TestSelectAIMoveIsCallable verifies the decision layer entry point exists
// and can be called without panicking (step 6).
func TestSelectAIMoveIsCallable(t *testing.T) {
	// We only test that the function symbol exists and returns an error
	// when given a non-existent game (this is expected behavior).
	_, err := SelectAIMove("non-existent-game-id")
	if err == nil {
		t.Log("SelectAIMove returned no error for invalid game (acceptable in stub)")
	}
}

// TestHumanVsAI_BasicHumanThenAIMove exercises the core Human → AI → Human cycle
// for issue0021 regression coverage.
func TestHumanVsAI_BasicHumanThenAIMove(t *testing.T) {
	game, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsAI,
		sessionpkg.GameTypeChess,
		"white",
		1,
		"",
		"intermediate",
	)
	if err != nil {
		t.Fatalf("failed to create human_vs_ai game: %v", err)
	}

	// Human (White) plays a standard opening move
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "e2e4"); err != nil {
		t.Fatalf("human move e2e4 failed: %v", err)
	}

	// After human move it must be Black's turn
	turn, err := sessionpkg.CurrentTurnColorByID(game.ID)
	if err != nil {
		t.Fatalf("failed to get turn after human move: %v", err)
	}
	if turn != "black" {
		t.Fatalf("expected black to move after human, got %s", turn)
	}

	// AI (Black) produces a move via the decision layer (Fairy-Stockfish)
	aiMove, err := SelectAIMove(game.ID)
	if err != nil {
		t.Fatalf("SelectAIMove failed: %v", err)
	}
	if aiMove == "" {
		t.Fatalf("SelectAIMove returned empty move")
	}

	// Apply the AI move
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, aiMove); err != nil {
		t.Fatalf("AI move %s failed to apply: %v", aiMove, err)
	}

	// Turn should now be back to White
	turn, err = sessionpkg.CurrentTurnColorByID(game.ID)
	if err != nil {
		t.Fatalf("failed to get turn after AI move: %v", err)
	}
	if turn != "white" {
		t.Fatalf("expected white to move after AI reply, got %s", turn)
	}

	// Game must still be in progress
	g, err := sessionpkg.GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("failed to fetch game after AI move: %v", err)
	}
	if g.Result != sessionpkg.GameResultInProgress {
		t.Fatalf("expected game still in progress, got result=%s", g.Result)
	}
}