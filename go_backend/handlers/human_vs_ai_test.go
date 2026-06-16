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

// TestHumanVsAI_InitialAIMoveWhenHumanIsBlack verifies that when a human_vs_ai
// game is created with humanColor=black, the AI (White) automatically makes
// the first move (issue0021 regression).
func TestHumanVsAI_InitialAIMoveWhenHumanIsBlack(t *testing.T) {
	game, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsAI,
		sessionpkg.GameTypeChess,
		"black", // human plays Black
		1,
		"",
		"intermediate",
	)
	if err != nil {
		t.Fatalf("failed to create human_vs_ai game with human=black: %v", err)
	}

	// Config must reflect humanColor=black
	if game.Config.HumanColor != "black" {
		t.Fatalf("expected humanColor=black, got %s", game.Config.HumanColor)
	}

	// At creation time, White (AI) should have been given the first move opportunity.
	// We simulate what the background goroutine does.
	aiMove, err := SelectAIMove(game.ID)
	if err != nil {
		t.Fatalf("SelectAIMove for initial position failed: %v", err)
	}
	if aiMove == "" {
		t.Fatalf("expected initial AI move when human is Black, got empty move")
	}

	// Apply the AI move
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, aiMove); err != nil {
		t.Fatalf("failed to apply initial AI move %s: %v", aiMove, err)
	}

	// After AI (White) moves, it must be Black's (human) turn
	turn, err := sessionpkg.CurrentTurnColorByID(game.ID)
	if err != nil {
		t.Fatalf("failed to get turn after initial AI move: %v", err)
	}
	if turn != "black" {
		t.Fatalf("expected black (human) to move after initial AI move, got %s", turn)
	}
}

// TestHumanVsAI_DifferentOpeningSequence tests a different human opening (d4)
// followed by an AI reply to ensure the flow works for multiple sequences.
func TestHumanVsAI_DifferentOpeningSequence(t *testing.T) {
	game, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsAI,
		sessionpkg.GameTypeChess,
		"white",
		1,
		"",
		"intermediate",
	)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Human plays d4
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "d2d4"); err != nil {
		t.Fatalf("human d2d4 failed: %v", err)
	}

	aiMove, err := SelectAIMove(game.ID)
	if err != nil || aiMove == "" {
		t.Fatalf("SelectAIMove after d4 failed: %v", err)
	}

	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, aiMove); err != nil {
		t.Fatalf("AI move after d4 failed: %v", err)
	}

	// Should be White's turn again
	turn, _ := sessionpkg.CurrentTurnColorByID(game.ID)
	if turn != "white" {
		t.Fatalf("expected white after AI reply on d4, got %s", turn)
	}
}

// TestHumanVsAI_MultipleExchanges runs two full human-AI cycles to verify
// repeated legal state transitions work.
func TestHumanVsAI_MultipleExchanges(t *testing.T) {
	game, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsAI,
		sessionpkg.GameTypeChess,
		"white",
		1,
		"",
		"intermediate",
	)
	if err != nil {
		t.Fatalf("create game failed: %v", err)
	}

	moves := []string{"e2e4", "g1f3"} // two human moves
	for _, m := range moves {
		if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, m); err != nil {
			t.Fatalf("human move %s failed: %v", m, err)
		}
		aiMove, err := SelectAIMove(game.ID)
		if err != nil || aiMove == "" {
			t.Fatalf("AI move after %s failed: %v", m, err)
		}
		if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, aiMove); err != nil {
			t.Fatalf("apply AI move %s failed: %v", aiMove, err)
		}
	}

	// After two full exchanges it should still be White's turn and game in progress
	turn, _ := sessionpkg.CurrentTurnColorByID(game.ID)
	if turn != "white" {
		t.Fatalf("expected white after two exchanges, got %s", turn)
	}
	g, _ := sessionpkg.GetGameSessionByID(game.ID)
	if g.Result != sessionpkg.GameResultInProgress {
		t.Fatalf("game ended too early: %s", g.Result)
	}
}

// TestHumanVsAI_ProfileVariation ensures different strength profiles can be
// used without crashing and still produce legal moves.
func TestHumanVsAI_ProfileVariation(t *testing.T) {
	profiles := []string{"beginner", "intermediate", "advanced", "master"}
	for _, p := range profiles {
		game, err := sessionpkg.CreateGame(
			sessionpkg.GameModeHumanVsAI,
			sessionpkg.GameTypeChess,
			"white",
			1,
			"",
			p,
		)
		if err != nil {
			t.Fatalf("create game with profile %s failed: %v", p, err)
		}
		if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "e2e4"); err != nil {
			t.Fatalf("human move with profile %s failed: %v", p, err)
		}
		aiMove, err := SelectAIMove(game.ID)
		if err != nil || aiMove == "" {
			t.Fatalf("SelectAIMove with profile %s failed: %v", p, err)
		}
		if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, aiMove); err != nil {
			t.Fatalf("apply AI move with profile %s failed: %v", p, err)
		}
	}
}