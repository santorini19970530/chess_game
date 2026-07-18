package handlers

import (
	"testing"

	sessionpkg "go_backend/game/session"
)

func TestMoveAppliedPayload_IncludesCaptureFlag(t *testing.T) {
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "e2e4"); err != nil {
		t.Fatalf("quiet move: %v", err)
	}
	quiet := moveAppliedPayload(game.ID, "e2e4")
	if quiet["isCapture"] != false {
		t.Fatalf("quiet move isCapture=%v", quiet["isCapture"])
	}

	// Set up a capture: e7e5 then d2d4 then e5d4.
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "e7e5"); err != nil {
		t.Fatalf("e7e5: %v", err)
	}
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "d2d4"); err != nil {
		t.Fatalf("d2d4: %v", err)
	}
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "e5d4"); err != nil {
		t.Fatalf("capture: %v", err)
	}
	cap := moveAppliedPayload(game.ID, "e5d4")
	if cap["isCapture"] != true {
		t.Fatalf("capture isCapture=%v want true", cap["isCapture"])
	}
	if cap["command"] != "e5d4" {
		t.Fatalf("command=%v", cap["command"])
	}
}
