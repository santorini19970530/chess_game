package chessboard

import (
	"strings"
	"testing"
)

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

func TestNewBoard_XiangqiDrawHas90SquaresAndLabels(t *testing.T) {
	html := string(NewBoard(9, 10).Draw())
	if !strings.Contains(html, `--board-files: 9`) || !strings.Contains(html, `--board-ranks: 10`) {
		t.Fatalf("expected CSS vars for 9×10, got snippet missing vars")
	}
	if strings.Count(html, `data-sequence=`) != 90 {
		t.Fatalf("expected 90 squares, got %d", strings.Count(html, `data-sequence=`))
	}
	if !strings.Contains(html, `>i<`) || !strings.Contains(html, `>10<`) {
		t.Fatalf("expected file i and rank 10 labels")
	}
	if !strings.Contains(html, "xq_river_break") || !strings.Contains(html, "chess_board_square_palace") {
		t.Fatalf("expected river/palace cue classes on 9×10 board")
	}
	if !strings.Contains(html, "xq_edge_w") || !strings.Contains(html, "xq_edge_e") {
		t.Fatalf("expected edge cue classes on 9×10 board")
	}
}
