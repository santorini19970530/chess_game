// CM3070 FP code
// shogi_end_impasse_test.go - tests for shogi FESA 5.3 declaration (入玉宣言)

package session

import (
	"testing"

	pieces "go_backend/game/piece"
)

// white king in zone, 10 camp pieces, hand RRBB → 30 points (FESA 5.3 pass)
const shogiImpassePassFEN = "9/G3K4/PPPPPPPPP/9/9/9/9/9/4k4[RRBB] w - - 0 1"

// same camp, empty hand → 10 points, short of Sente 28
const shogiImpasseShortPointsFEN = "9/G3K4/PPPPPPPPP/9/9/9/9/9/4k4[] w - - 0 1"

func TestShogiImpasse_DeclarationWins(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, shogiImpassePassFEN, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	ended, err := DeclareShogiImpasseByID(game.ID)
	if err != nil {
		t.Fatalf("declare: %v", err)
	}
	if ended.Outcome.Status != "impasse" {
		t.Fatalf("status=%q want impasse", ended.Outcome.Status)
	}
	if ended.Result != GameResultWhiteWin || ended.Outcome.Winner != "white" {
		t.Fatalf("result=%q winner=%q", ended.Result, ended.Outcome.Winner)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "e8e9"); err == nil {
		t.Fatal("next /move must be rejected after impasse")
	}
}

func TestShogiImpasse_ShortPointsLoses(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, shogiImpasseShortPointsFEN, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	ended, err := DeclareShogiImpasseByID(game.ID)
	if err != nil {
		t.Fatalf("declare: %v", err)
	}
	if ended.Outcome.Status != "impasse_failed" {
		t.Fatalf("status=%q want impasse_failed", ended.Outcome.Status)
	}
	if ended.Result != GameResultBlackWin || ended.Outcome.Loser != "white" {
		t.Fatalf("result=%q loser=%q", ended.Result, ended.Outcome.Loser)
	}
}

func TestShogiImpasse_ChessRejected(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := DeclareShogiImpasseByID(game.ID); err == nil {
		t.Fatal("chess must not accept an impasse declaration")
	}
	got, err := GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Result != GameResultInProgress {
		t.Fatalf("chess must stay in progress, result=%q", got.Result)
	}
}

func TestShogiImpasse_TryOnlyWhenChecklistPasses(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	start, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create start: %v", err)
	}
	declared, after, err := TryShogiImpasseDeclarationByID(start.ID)
	if err != nil {
		t.Fatalf("try start: %v", err)
	}
	if declared || after.Result != GameResultInProgress {
		t.Fatalf("start must not auto-declare, declared=%v result=%q", declared, after.Result)
	}

	resetGameSessionForTest()
	ResetGame()
	pass, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, shogiImpassePassFEN, "")
	if err != nil {
		t.Fatalf("create pass: %v", err)
	}
	declared, ended, err := TryShogiImpasseDeclarationByID(pass.ID)
	if err != nil {
		t.Fatalf("try pass: %v", err)
	}
	if !declared || ended.Outcome.Status != "impasse" {
		t.Fatalf("pass FEN must declare, declared=%v status=%q", declared, ended.Outcome.Status)
	}
}

func TestShogiImpasse_RefreshKeepsDeclaration(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, shogiImpassePassFEN, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := DeclareShogiImpasseByID(game.ID); err != nil {
		t.Fatalf("declare: %v", err)
	}
	refreshed, err := RefreshGameSessionOutcomeByID(game.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.Outcome.Status != "impasse" || refreshed.Result != GameResultWhiteWin {
		t.Fatalf("refresh wiped declaration, status=%q result=%q", refreshed.Outcome.Status, refreshed.Result)
	}
}

func TestShogiImpasse_Sente28Gote27(t *testing.T) {
	if shogiImpassePointsNeeded(pieces.White) != 28 {
		t.Fatalf("sente need=%d", shogiImpassePointsNeeded(pieces.White))
	}
	if shogiImpassePointsNeeded(pieces.Black) != 27 {
		t.Fatalf("gote need=%d", shogiImpassePointsNeeded(pieces.Black))
	}
}

// TestShogiImpasse_KingNotInZoneLoses - start-position declare fails FESA 5.3
func TestShogiImpasse_KingNotInZoneLoses(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	ended, err := DeclareShogiImpasseByID(game.ID)
	if err != nil {
		t.Fatalf("declare: %v", err)
	}
	if ended.Outcome.Status != "impasse_failed" || ended.Outcome.Loser != "white" {
		t.Fatalf("status=%q loser=%q", ended.Outcome.Status, ended.Outcome.Loser)
	}
}

// TestShogiImpasse_InCheckLoses - king in zone with enough points still fails if in check
func TestShogiImpasse_InCheckLoses(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	const fen = "4r4/G3K4/PPPPPPPPP/9/9/9/9/9/4k4[RRBB] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	ended, err := DeclareShogiImpasseByID(game.ID)
	if err != nil {
		t.Fatalf("declare: %v", err)
	}
	if ended.Outcome.Status != "impasse_failed" || ended.Outcome.Loser != "white" {
		t.Fatalf("in check must fail, status=%q loser=%q", ended.Outcome.Status, ended.Outcome.Loser)
	}
}

// TestShogiImpasse_Gote27PassesAnd26Fails - gote 27 on a black-to-move board wins; 26 loses
func TestShogiImpasse_Gote27PassesAnd26Fails(t *testing.T) {
	const pass = "4K4/9/9/9/9/9/ppppppppp/g3k4/9[rrbpp] b - - 0 1"
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, pass, "")
	if err != nil {
		t.Fatalf("create pass: %v", err)
	}
	ended, err := DeclareShogiImpasseByID(game.ID)
	if err != nil {
		t.Fatalf("declare pass: %v", err)
	}
	if ended.Outcome.Status != "impasse" || ended.Outcome.Winner != "black" {
		t.Fatalf("gote 27 must win, status=%q winner=%q", ended.Outcome.Status, ended.Outcome.Winner)
	}

	const fail = "4K4/9/9/9/9/9/ppppppppp/g3k4/9[rrbp] b - - 0 1"
	resetGameSessionForTest()
	ResetGame()
	game, err = CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fail, "")
	if err != nil {
		t.Fatalf("create fail: %v", err)
	}
	ended, err = DeclareShogiImpasseByID(game.ID)
	if err != nil {
		t.Fatalf("declare fail: %v", err)
	}
	if ended.Outcome.Status != "impasse_failed" || ended.Outcome.Loser != "black" {
		t.Fatalf("gote 26 must lose, status=%q loser=%q", ended.Outcome.Status, ended.Outcome.Loser)
	}
}
