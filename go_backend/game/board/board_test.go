package chessboard

import (
	"strings"
	"testing"
)

// TestSequenceByFileRank_ChessAndXiangqi - checks sequence by file rank chess and xiangqi
func TestSequenceByFileRank_ChessAndXiangqi(t *testing.T) {
	// Chess a8 = 0, h1 = 63
	if got := SequenceByFileRank(1, 8, 8, 8); got != 0 {
		t.Fatalf("chess a8: got %d want 0", got)
	}
	if got := SequenceByFileRank(8, 1, 8, 8); got != 63 {
		t.Fatalf("chess h1: got %d want 63", got)
	}
	// Xiangqi a10 = 0, i1 = 89
	if got := SequenceByFileRank(1, 10, 9, 10); got != 0 {
		t.Fatalf("xiangqi a10: got %d want 0", got)
	}
	if got := SequenceByFileRank(9, 1, 9, 10); got != 89 {
		t.Fatalf("xiangqi i1: got %d want 89", got)
	}
	file, rank := FileRankFromSequence(89, 9, 10)
	if file != 9 || rank != 1 {
		t.Fatalf("FileRankFromSequence(89): got %d,%d want 9,1", file, rank)
	}
}

// TestNewBoard_XiangqiHas90SquaresAndCueClasses - checks 9×10 geometry and palace/river cues
func TestNewBoard_XiangqiHas90SquaresAndCueClasses(t *testing.T) {
	board := NewBoard(9, 10)
	if board.Files != 9 || board.Ranks != 10 {
		t.Fatalf("expected 9×10 board, got %d×%d", board.Files, board.Ranks)
	}
	if len(board.Squares()) != 90 {
		t.Fatalf("expected 90 squares, got %d", len(board.Squares()))
	}
	cues := SquareCueClasses(9, 10, 5, 6)
	if !strings.Contains(cues, "xq_river_break") {
		t.Fatalf("expected river cue on file 5 rank 6, got %q", cues)
	}
	palace := SquareCueClasses(9, 10, 5, 2)
	if !strings.Contains(palace, "chess_board_square_palace") {
		t.Fatalf("expected palace cue on file 5 rank 2, got %q", palace)
	}
}
