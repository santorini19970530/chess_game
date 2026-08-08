// CM3070 FP code
// board_draw_test.go - tests for board draw

package handlers

import (
	"strings"
	"testing"

	chessboard "go_backend/game/board"
)

// TestDrawChessBoard_XiangqiHas90SquaresAndLabels - checks 9×10 html has squares, labels, and cue classes
func TestDrawChessBoard_XiangqiHas90SquaresAndLabels(t *testing.T) {
	html := string(DrawChessBoard(chessboard.NewBoard(9, 10)))
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
