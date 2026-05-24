package session

import (
	pieces "go_backend/game/piece"
	"testing"
)

func TestEvaluateGameOutcome_Checkmate(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 6, Rank: 6},   // Kf6
		{Color: pieces.White, Kind: pieces.Queen, ImgFile: "pic/chess_pic/queen_light.png", File: 7, Rank: 7}, // Qg7
		{Color: pieces.Black, Kind: pieces.King, ImgFile: "pic/chess_pic/king_dark.png", File: 8, Rank: 8},    // kh8
	}
	moveHistory = []string{"White: a2a3"} // black to move
	lastAppliedMove = nil
	resetCastlingState()
	resetGameSessionForTest()

	outcome := EvaluateGameOutcome()
	if outcome.Status != "checkmate" {
		t.Fatalf("expected checkmate, got %q", outcome.Status)
	}
	if outcome.Winner != "white" || outcome.Loser != "black" {
		t.Fatalf("expected winner=white loser=black, got winner=%q loser=%q", outcome.Winner, outcome.Loser)
	}
	if outcome.CheckedSide != "black" {
		t.Fatalf("expected checked side black, got %q", outcome.CheckedSide)
	}
	if outcome.LegalMoves != 0 {
		t.Fatalf("expected 0 legal moves in checkmate, got %d", outcome.LegalMoves)
	}
}

func TestEvaluateGameOutcome_Stalemate(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 6, Rank: 7},   // Kf7
		{Color: pieces.White, Kind: pieces.Queen, ImgFile: "pic/chess_pic/queen_light.png", File: 7, Rank: 6}, // Qg6
		{Color: pieces.Black, Kind: pieces.King, ImgFile: "pic/chess_pic/king_dark.png", File: 8, Rank: 8},    // kh8
	}
	moveHistory = []string{"White: a2a3"} // black to move
	lastAppliedMove = nil
	resetCastlingState()
	resetGameSessionForTest()

	outcome := EvaluateGameOutcome()
	if outcome.Status != "stalemate" {
		t.Fatalf("expected stalemate, got %q", outcome.Status)
	}
	if outcome.LegalMoves != 0 {
		t.Fatalf("expected 0 legal moves in stalemate, got %d", outcome.LegalMoves)
	}
}

func TestRefreshGameSessionOutcome_UpdatesMetadataAndResult(t *testing.T) {
	resetGameSessionForTest()
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 6, Rank: 6},   // Kf6
		{Color: pieces.White, Kind: pieces.Queen, ImgFile: "pic/chess_pic/queen_light.png", File: 7, Rank: 7}, // Qg7
		{Color: pieces.Black, Kind: pieces.King, ImgFile: "pic/chess_pic/king_dark.png", File: 8, Rank: 8},    // kh8
	}
	moveHistory = []string{"White: a2a3"} // black to move
	lastAppliedMove = nil
	resetCastlingState()

	game := RefreshGameSessionOutcome()
	if game.ID == "" {
		t.Fatalf("expected game id to be populated")
	}
	if game.Type != GameTypeChess {
		t.Fatalf("expected game type chess, got %q", game.Type)
	}
	if game.Mode != GameModeHumanVsHuman {
		t.Fatalf("expected game mode human_vs_human, got %q", game.Mode)
	}
	if game.Result != GameResultWhiteWin {
		t.Fatalf("expected white win result, got %q", game.Result)
	}
	if game.Outcome.Status != "checkmate" {
		t.Fatalf("expected checkmate outcome, got %q", game.Outcome.Status)
	}
}

func TestEvaluateGameOutcome_UserSequenceIsCheckNotMate(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	sequence := []string{"h2h4", "e7e5", "h1h3", "e5e4", "h3e3", "e8e7", "e3e4"}
	for _, mv := range sequence {
		if _, err := ApplyMoveByCommand(mv); err != nil {
			t.Fatalf("expected move %s to succeed, got: %v", mv, err)
		}
	}

	outcome := EvaluateGameOutcome()
	if outcome.Status != "check" {
		t.Fatalf("expected check outcome, got %q", outcome.Status)
	}
	if outcome.CheckedSide != "black" {
		t.Fatalf("expected checked side black, got %q", outcome.CheckedSide)
	}
	if outcome.LegalMoves <= 0 {
		t.Fatalf("expected legal moves for black, got %d", outcome.LegalMoves)
	}
}

func TestFlagCurrentTurn_SetsWinLossOutcome(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game := FlagCurrentTurn() // white to move initially, so white flags and black wins.
	if game.Outcome.Status != "resigned" {
		t.Fatalf("expected resigned outcome, got %q", game.Outcome.Status)
	}
	if game.Outcome.Loser != "white" || game.Outcome.Winner != "black" {
		t.Fatalf("expected loser=white winner=black, got loser=%q winner=%q", game.Outcome.Loser, game.Outcome.Winner)
	}
	if game.Result != GameResultBlackWin {
		t.Fatalf("expected black win result, got %q", game.Result)
	}
}

func TestUpdateGameConfig_WithFENForcesSingleGameAndLoadsPosition(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	fen := "8/8/8/8/8/8/4k3/4K3 b - - 0 1"
	game, err := UpdateGameConfig(GameModeAIVsAI, GameTypeChess, "white", 20, fen)
	if err != nil {
		t.Fatalf("expected config update success, got %v", err)
	}
	if game.Config.AIGameCount != 1 {
		t.Fatalf("expected fen to force ai game count 1, got %d", game.Config.AIGameCount)
	}

	started, err := StartConfiguredNewGame()
	if err != nil {
		t.Fatalf("expected configured new game success, got %v", err)
	}
	if started.Config.StartFEN != fen {
		t.Fatalf("expected fen to persist in config")
	}
	if CurrentTurnColor() != pieces.Black {
		t.Fatalf("expected black to move from FEN")
	}
}

func TestEvaluateGameOutcome_DrawByInsufficientMaterial(t *testing.T) {
	resetGameSessionForTest()
	resetTurnOverride()
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 5, Rank: 1},
		{Color: pieces.Black, Kind: pieces.King, ImgFile: "pic/chess_pic/king_dark.png", File: 5, Rank: 8},
	}
	moveHistory = nil
	lastAppliedMove = nil
	resetCastlingState()
	resetDrawTracking()

	outcome := EvaluateGameOutcome()
	if outcome.Status != "draw_insufficient_material" {
		t.Fatalf("expected draw_insufficient_material, got %q", outcome.Status)
	}
}

func TestEvaluateGameOutcome_DrawByFiftyMoveRule(t *testing.T) {
	resetGameSessionForTest()
	resetTurnOverride()
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 5, Rank: 1},
		{Color: pieces.White, Kind: pieces.Rook, ImgFile: "pic/chess_pic/rook_light.png", File: 1, Rank: 1},
		{Color: pieces.Black, Kind: pieces.King, ImgFile: "pic/chess_pic/king_dark.png", File: 5, Rank: 8},
	}
	moveHistory = nil
	lastAppliedMove = nil
	resetCastlingState()
	resetDrawTracking()
	halfmoveClock = 100

	outcome := EvaluateGameOutcome()
	if outcome.Status != "draw_fifty_move_rule" {
		t.Fatalf("expected draw_fifty_move_rule, got %q", outcome.Status)
	}
}

func TestEvaluateGameOutcome_DrawByThreefoldRepetition(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	positionCounts[currentPositionKey()] = 3

	outcome := EvaluateGameOutcome()
	if outcome.Status != "draw_threefold_repetition" {
		t.Fatalf("expected draw_threefold_repetition, got %q", outcome.Status)
	}
}

func TestResetGame_RestoresInitialState(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, ImgFile: "pic/chess_pic/king_light.png", File: 4, Rank: 4},
	}
	moveHistory = []string{"White: e2e4", "Black: e7e5"}
	lastAppliedMove = &LastMove{FromFile: 5, FromRank: 7, ToFile: 5, ToRank: 5, PieceKind: pieces.Pawn, Color: pieces.Black}
	whiteKingMoved = true

	ResetGame()

	if len(pieces.ChessPieces) != len(initialPiecesSnapshot) {
		t.Fatalf("expected initial piece count %d, got %d", len(initialPiecesSnapshot), len(pieces.ChessPieces))
	}
	if len(moveHistory) != 0 {
		t.Fatalf("expected move history to be cleared")
	}
	if lastAppliedMove != nil {
		t.Fatalf("expected last move to be cleared")
	}
	if whiteKingMoved || blackKingMoved || whiteRookAMoved || whiteRookHMoved || blackRookAMoved || blackRookHMoved {
		t.Fatalf("expected castling state to be reset")
	}
}
