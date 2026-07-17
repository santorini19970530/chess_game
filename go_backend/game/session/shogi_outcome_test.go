package session

import "testing"

func TestShogiOutcome_Checkmate(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	// Gold on e8 checks king on e9; rook on e5 protects the gold (king cannot capture).
	const mateFEN = "4k4/4G4/9/9/4R4/9/9/9/4K4[] b - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, mateFEN, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if game.Outcome.Status != "checkmate" {
		t.Fatalf("status=%q want checkmate (msg=%q legal=%d)", game.Outcome.Status, game.Outcome.Message, game.Outcome.LegalMoves)
	}
	if game.Outcome.Winner != "white" || game.Result != GameResultWhiteWin {
		t.Fatalf("winner=%q result=%q", game.Outcome.Winner, game.Result)
	}
}

func TestShogiOutcome_MissingKing(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	const fen = "9/9/9/9/9/9/9/9/4K4[] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if game.Outcome.Status != "checkmate" || game.Outcome.Winner != "white" {
		t.Fatalf("status=%q winner=%q", game.Outcome.Status, game.Outcome.Winner)
	}
}

func TestShogiOutcome_Flag(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	flagged, err := FlagCurrentTurnByID(game.ID)
	if err != nil {
		t.Fatalf("flag: %v", err)
	}
	if flagged.Outcome.Status != "resigned" || flagged.Outcome.Winner != "black" {
		t.Fatalf("status=%q winner=%q", flagged.Outcome.Status, flagged.Outcome.Winner)
	}
	if flagged.Result != GameResultBlackWin {
		t.Fatalf("result=%q", flagged.Result)
	}
}

func TestShogiOutcome_InProgressAtStart(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if game.Outcome.Status != "in_progress" && game.Outcome.Status != "check" {
		t.Fatalf("status=%q", game.Outcome.Status)
	}
	if game.Outcome.LegalMoves == 0 {
		t.Fatal("start position should have legal moves")
	}
}
