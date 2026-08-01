package session

import "testing"

// TestCreateGame_ClockDisabledByDefault - checks create game clock disabled by default
func TestCreateGame_ClockDisabledByDefault(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if game.Clock == nil {
		t.Fatal("expected Clock on session")
	}
	if game.Clock.Enabled {
		t.Fatal("expected clock disabled by default")
	}
}

// TestSetClockByID_AsymmetricAndStartsActiveSide - checks set clock by id asymmetric and starts active side
func TestSetClockByID_AsymmetricAndStartsActiveSide(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsAI, GameTypeChess, "white", 1, "", "beginner")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	updated, err := SetClockByID(game.ID, 5*60*1000, 60*1000, 30_000)
	if err != nil {
		t.Fatalf("SetClockByID: %v", err)
	}
	if !updated.Clock.Enabled {
		t.Fatal("expected clock enabled")
	}
	if updated.Clock.Remaining("white") != 5*60*1000 {
		t.Fatalf("white remaining=%d", updated.Clock.Remaining("white"))
	}
	if updated.Clock.Remaining("black") != 60*1000 {
		t.Fatalf("black remaining=%d", updated.Clock.Remaining("black"))
	}
	if updated.Clock.IncrementMs != 30_000 {
		t.Fatalf("increment=%d", updated.Clock.IncrementMs)
	}
	if updated.Clock.ActiveSide() != "white" {
		t.Fatalf("active=%q want white", updated.Clock.ActiveSide())
	}
}

// TestClockSidesFromHumanAI - checks clock sides from human ai
func TestClockSidesFromHumanAI(t *testing.T) {
	w, b := ClockSidesFromHumanAI("white", 300_000, 60_000)
	if w != 300_000 || b != 60_000 {
		t.Fatalf("human white: got %d/%d", w, b)
	}
	w, b = ClockSidesFromHumanAI("black", 300_000, 60_000)
	if w != 60_000 || b != 300_000 {
		t.Fatalf("human black: got %d/%d", w, b)
	}
}

// TestSetClockByID_Disable - checks set clock by id disable
func TestSetClockByID_Disable(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if _, err := SetClockByID(game.ID, 60_000, 60_000, 0); err != nil {
		t.Fatalf("enable: %v", err)
	}
	updated, err := SetClockByID(game.ID, 0, 0, 0)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if updated.Clock.Enabled {
		t.Fatal("expected clock disabled")
	}
}
