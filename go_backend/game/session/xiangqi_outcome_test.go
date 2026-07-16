package session

import "testing"

func TestXiangqiOutcome_Checkmate(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	// Two chariots on the back rank: black general has no flight squares.
	const mateFEN = "R3k3R/9/9/9/9/9/9/9/9/4K4 b - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, mateFEN, "")
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

func TestXiangqiOutcome_StalemateIsLoss(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	// Black to move, not in check, zero legal moves → loss (Xiangqi).
	const stalemateFEN = "5k3/4R4/9/9/9/9/9/9/9/4K4 b - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, stalemateFEN, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if game.Outcome.Status != "checkmate" {
		t.Fatalf("status=%q want checkmate mapping for decisive stalemate loss", game.Outcome.Status)
	}
	if game.Outcome.Winner != "white" || game.Result != GameResultWhiteWin {
		t.Fatalf("winner=%q result=%q", game.Outcome.Winner, game.Result)
	}
	if game.Outcome.Message == "" || game.Outcome.LegalMoves != 0 {
		t.Fatalf("msg=%q legal=%d", game.Outcome.Message, game.Outcome.LegalMoves)
	}
}

func TestXiangqiOutcome_Flag(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, "", "")
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
