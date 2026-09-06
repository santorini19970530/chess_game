// CM3070 FP code
// chess_end_test.go - tests for chess ply-end strategy order (existing law only)

package session

import "testing"

func TestChessMissingKingStrategy_WhiteGone(t *testing.T) {
	ended, out := (ChessMissingKingStrategy{}).AfterPly(&plyEndContext{
		GameType:   GameTypeChess,
		WhiteKings: 0,
		BlackKings: 1,
	})
	if !ended || out.Status != "checkmate" || out.Winner != "black" {
		t.Fatalf("ended=%v status=%q winner=%q", ended, out.Status, out.Winner)
	}
}

func TestChessMateOrStalemateStrategy_MateVsStalemate(t *testing.T) {
	s := ChessMateOrStalemateStrategy{}
	ended, out := s.AfterPly(&plyEndContext{GameType: GameTypeChess, LegalMoves: 0, InCheck: true, SideToMove: "black"})
	if !ended || out.Status != "checkmate" {
		t.Fatalf("mate: ended=%v status=%q", ended, out.Status)
	}
	ended, out = s.AfterPly(&plyEndContext{GameType: GameTypeChess, LegalMoves: 0, InCheck: false, SideToMove: "black"})
	if !ended || out.Status != "stalemate" {
		t.Fatalf("stalemate: ended=%v status=%q", ended, out.Status)
	}
}

func TestChessDrawStrategies_NeedLegalMoves(t *testing.T) {
	ctx := &plyEndContext{
		GameType:              GameTypeChess,
		LegalMoves:            0,
		InsufficientMaterial:  true,
		PositionCount:        3,
		IdlePly:               100,
	}
	if ended, _ := (ChessInsufficientMaterialStrategy{}).AfterPly(ctx); ended {
		t.Fatal("insufficient material must not fire when there are no legal moves")
	}
	if ended, _ := (ChessThreefoldStrategy{}).AfterPly(ctx); ended {
		t.Fatal("threefold must not fire when there are no legal moves")
	}
	if ended, _ := (ChessFiftyMoveStrategy{}).AfterPly(ctx); ended {
		t.Fatal("fifty-move must not fire when there are no legal moves")
	}
}

func TestChessGameEndChain_MateBeforeDraws(t *testing.T) {
	clearGameEndStrategiesForTest()
	registerGameEndStrategies(GameTypeChess, chessGameEndStrategies()...)

	ended, out := runGameEndStrategies(&plyEndContext{
		GameType:             GameTypeChess,
		WhiteKings:           1,
		BlackKings:           1,
		LegalMoves:           0,
		InCheck:              true,
		SideToMove:           "black",
		InsufficientMaterial: true,
		PositionCount:       3,
		IdlePly:              100,
	})
	if !ended || out.Status != "checkmate" {
		t.Fatalf("mate must win the chess chain, ended=%v status=%q", ended, out.Status)
	}

	ended, out = runGameEndStrategies(&plyEndContext{
		GameType:             GameTypeChess,
		WhiteKings:           1,
		BlackKings:           1,
		LegalMoves:           8,
		InsufficientMaterial: true,
		PositionCount:       3,
		IdlePly:              100,
	})
	if !ended || out.Status != "draw_insufficient_material" {
		t.Fatalf("insufficient must beat threefold and fifty, ended=%v status=%q", ended, out.Status)
	}
}
