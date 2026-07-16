package session

import "testing"

func TestCreateGame_AllowsXiangqiAndSetsDefaultStartFEN(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("expected xianqi create to succeed, got %v", err)
	}
	if game.Type != GameTypeXiangqi {
		t.Fatalf("expected type %q, got %q", GameTypeXiangqi, game.Type)
	}
	if game.Config.StartFEN != DefaultXiangqiStartFEN {
		t.Fatalf("expected default xiangqi FEN, got %q", game.Config.StartFEN)
	}
}

func TestCreateGame_RejectsChessFENForXiangqi(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	chessFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	_, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, chessFEN, "")
	if err == nil {
		t.Fatal("expected chess FEN to be rejected for xianqi")
	}
}

func TestCreateGame_StillRejectsShogi(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	_, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err == nil {
		t.Fatal("expected shogi create to remain rejected")
	}
}

func TestUpdateGameConfigByID_AllowsXiangqi(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("seed chess game: %v", err)
	}
	updated, err := UpdateGameConfigByID(game.ID, GameModeHumanVsHuman, GameTypeXiangqi, "black", 1, "", "beginner")
	if err != nil {
		t.Fatalf("expected xianqi config update to succeed, got %v", err)
	}
	if updated.Type != GameTypeXiangqi {
		t.Fatalf("expected type %q, got %q", GameTypeXiangqi, updated.Type)
	}
	if updated.Config.StartFEN != DefaultXiangqiStartFEN {
		t.Fatalf("expected default xiangqi FEN on config update, got %q", updated.Config.StartFEN)
	}
	if updated.Config.HumanColor != "black" {
		t.Fatalf("expected humanColor black, got %q", updated.Config.HumanColor)
	}
}
