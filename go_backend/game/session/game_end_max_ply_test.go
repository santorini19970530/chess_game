// CM3070 FP code
// game_end_max_ply_test.go - tests for the shared ply-limit ply-end strategy

package session

import "testing"

func TestMaxPlyStrategy_EndsAtLimit(t *testing.T) {
	s := MaxPlyStrategy{Limit: 3}
	ended, _ := s.AfterPly(&plyEndContext{PlyCount: 2, LegalMoves: 8})
	if ended {
		t.Fatal("2 ply must not hit a limit of 3")
	}
	ended, out := s.AfterPly(&plyEndContext{PlyCount: 3, LegalMoves: 8})
	if !ended || out.Status != "draw_max_plies" {
		t.Fatalf("ended=%v status=%q", ended, out.Status)
	}
}

func TestChessGameEndChain_LawBeforeMaxPly(t *testing.T) {
	clearGameEndStrategiesForTest()
	registerGameEndStrategies(GameTypeChess, chessGameEndStrategies()...)

	ended, out := runGameEndStrategies(&plyEndContext{
		GameType:             GameTypeChess,
		WhiteKings:           1,
		BlackKings:           1,
		LegalMoves:           8,
		InsufficientMaterial: true,
		PlyCount:             DefaultMaxPlies,
	})
	if !ended || out.Status != "draw_insufficient_material" {
		t.Fatalf("real chess law must beat ply limit, ended=%v status=%q", ended, out.Status)
	}

	ended, out = runGameEndStrategies(&plyEndContext{
		GameType:   GameTypeChess,
		WhiteKings: 1,
		BlackKings: 1,
		LegalMoves: 8,
		PlyCount:   DefaultMaxPlies,
	})
	if !ended || out.Status != "draw_max_plies" {
		t.Fatalf("ply limit must end when no other law fired, ended=%v status=%q", ended, out.Status)
	}
}
