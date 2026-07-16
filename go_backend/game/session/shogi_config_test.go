package session

import "testing"

func TestCreateGame_AllowsShogiAndSetsDefaultStartFEN(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("expected shogi create to succeed, got %v", err)
	}
	if game.Type != GameTypeShogi {
		t.Fatalf("expected type %q, got %q", GameTypeShogi, game.Type)
	}
	if game.Config.StartFEN != DefaultShogiStartFEN {
		t.Fatalf("expected default shogi FEN, got %q", game.Config.StartFEN)
	}
}

func TestCreateGame_RejectsChessFENForShogi(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	chessFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	_, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, chessFEN, "")
	if err == nil {
		t.Fatal("expected chess FEN to be rejected for shogi")
	}
}

func TestCreateGame_RejectsXiangqiFENForShogi(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	_, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, DefaultXiangqiStartFEN, "")
	if err == nil {
		t.Fatal("expected xiangqi FEN to be rejected for shogi")
	}
}

func TestCreateGame_ShogiMaterializesStartBoard(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	fen, err := CurrentFENByID(game.ID)
	if err != nil {
		t.Fatalf("fen: %v", err)
	}
	if fen != DefaultShogiStartFEN {
		t.Fatalf("BoardFEN=%q want default start", fen)
	}

	snap, err := BuildSnapshotByID(game.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snap.State) != 40 {
		t.Fatalf("piece count=%d want 40", len(snap.State))
	}

	// Sente (white) king on e1 (file 5, rank 1); Gote king on e9.
	var whiteKing, blackKing bool
	for _, p := range snap.State {
		if p.Kind == "king" && p.Color == "white" && p.File == 5 && p.Rank == 1 {
			whiteKing = true
		}
		if p.Kind == "king" && p.Color == "black" && p.File == 5 && p.Rank == 9 {
			blackKing = true
		}
	}
	if !whiteKing || !blackKing {
		t.Fatalf("expected kings on e1/e9, white=%v black=%v state=%+v", whiteKing, blackKing, snap.State)
	}
	if snap.CurrentTurn != "White" {
		t.Fatalf("turn=%q want White", snap.CurrentTurn)
	}
}

func TestUpdateGameConfigByID_AllowsShogi(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("seed chess game: %v", err)
	}
	updated, err := UpdateGameConfigByID(game.ID, GameModeHumanVsHuman, GameTypeShogi, "black", 1, "", "beginner")
	if err != nil {
		t.Fatalf("expected shogi config update to succeed, got %v", err)
	}
	if updated.Type != GameTypeShogi {
		t.Fatalf("expected type %q, got %q", GameTypeShogi, updated.Type)
	}
	if updated.Config.StartFEN != DefaultShogiStartFEN {
		t.Fatalf("expected default shogi FEN on config update, got %q", updated.Config.StartFEN)
	}
}
