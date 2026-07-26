package session

import (
	"testing"
	"time"
)

func TestApplyMove_RejectsWhenGameAlreadyFlagged(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if _, err := FlagCurrentTurnByID(game.ID); err != nil {
		t.Fatalf("FlagCurrentTurnByID: %v", err)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "e2e4"); err == nil {
		t.Fatal("expected move rejected on finished game")
	}
}

func TestApplyMove_ClockTimeoutFlagsBeforeMove(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if _, err := SetClockByID(game.ID, 1000, 1000, 0); err != nil {
		t.Fatalf("SetClockByID: %v", err)
	}
	// Force active side already out of time when the next move is attempted.
	if err := sessionStore.Update(game.ID, func(rg *RuntimeGame) error {
		rg.Session.Clock.LastTickUnixMs = time.Now().UTC().Add(-2 * time.Second).UnixMilli()
		return nil
	}); err != nil {
		t.Fatalf("backdate clock: %v", err)
	}

	if _, err := ApplyMoveByCommandByID(game.ID, "e2e4"); err == nil {
		t.Fatal("expected move rejected after flag")
	}
	updated, err := GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("GetGameSessionByID: %v", err)
	}
	if updated.Result == GameResultInProgress || updated.Outcome.Status != "resigned" {
		t.Fatalf("expected flagged terminal game, got result=%s status=%s", updated.Result, updated.Outcome.Status)
	}
	if updated.Outcome.Loser != "white" {
		t.Fatalf("loser=%q want white", updated.Outcome.Loser)
	}
	snap, err := BuildSnapshotByID(game.ID)
	if err != nil {
		t.Fatalf("BuildSnapshotByID: %v", err)
	}
	if len(snap.History) != 0 {
		t.Fatalf("flagged game should not apply move, history=%v", snap.History)
	}
}

func TestApplyMove_AwardsIncrementAfterLegalMove(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if _, err := SetClockByID(game.ID, 1000, 1000, 30_000); err != nil {
		t.Fatalf("SetClockByID: %v", err)
	}
	// Keep settle near-zero so remaining ≈ initial - tiny + increment.
	if err := sessionStore.Update(game.ID, func(rg *RuntimeGame) error {
		rg.Session.Clock.LastTickUnixMs = time.Now().UTC().UnixMilli()
		return nil
	}); err != nil {
		t.Fatalf("sync clock tick: %v", err)
	}

	if _, err := ApplyMoveByCommandByID(game.ID, "e2e4"); err != nil {
		t.Fatalf("ApplyMove: %v", err)
	}
	updated, err := GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("GetGameSessionByID: %v", err)
	}
	got := updated.Clock.Remaining("white")
	if got < 30_000 || got > 31_000 {
		t.Fatalf("white remaining after +30s inc: got %d want ~30000-31000", got)
	}
	if updated.Clock.ActiveSide() != "black" {
		t.Fatalf("active=%q want black", updated.Clock.ActiveSide())
	}
}
