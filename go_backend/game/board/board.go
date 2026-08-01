// CM3070 FP code
// chessboard.go defines board geometry for files×ranks grids

package chessboard

import (
	"strings"
)

// chessBoard groups all board squares for a files×ranks grid
type ChessBoard struct {
	Files   int
	Ranks   int
	squares []ChessBoardSquare
}

// NewChessBoard - creates an 8×8 chess board (SSR default)
func NewChessBoard() *ChessBoard {
	return NewBoard(8, 8)
}

// NewBoard - creates a files×ranks square board (Xiangqi 9×10, Shogi 9×9, …)
func NewBoard(files, ranks int) *ChessBoard {
	if files <= 0 {
		files = 8
	}
	if ranks <= 0 {
		ranks = 8
	}
	n := files * ranks
	board := &ChessBoard{
		Files:   files,
		Ranks:   ranks,
		squares: make([]ChessBoardSquare, 0, n),
	}
	for i := 0; i < n; i++ {
		board.squares = append(board.squares, NewBoardSquare(i, files))
	}
	return board
}

// Squares - returns board squares in display sequence order
func (c *ChessBoard) Squares() []ChessBoardSquare {
	return c.squares
}

// SequenceByFileRank - maps 1-based file/rank to data-sequence (rank max at top)
func SequenceByFileRank(file, rank, files, maxRank int) int {
	return (maxRank-rank)*files + (file - 1)
}

// FileRankFromSequence - the inverse of SequenceByFileRank
func FileRankFromSequence(sequence, files, maxRank int) (file, rank int) {
	file = (sequence % files) + 1
	rank = maxRank - (sequence / files)
	return file, rank
}

// SquareCueClasses - adds Xiangqi edge/river/palace cues (9×10 only)
func SquareCueClasses(files, ranks, file, rank int) string {
	if files != 9 || ranks != 10 {
		return ""
	}
	var parts []string
	if file == 1 {
		parts = append(parts, "xq_edge_w")
	}
	if file == files {
		parts = append(parts, "xq_edge_e")
	}
	if rank == ranks {
		parts = append(parts, "xq_edge_n")
	}
	if rank == 1 {
		parts = append(parts, "xq_edge_s")
	}
	inner := file > 1 && file < files
	if inner && rank == 6 {
		parts = append(parts, "xq_river_break")
	}
	if inner && rank == 5 {
		parts = append(parts, "xq_river_break_low")
	}
	if file >= 4 && file <= 6 && ((rank >= 1 && rank <= 3) || (rank >= 8 && rank <= 10)) {
		parts = append(parts, "chess_board_square_palace")
	}
	return strings.Join(parts, " ")
}
