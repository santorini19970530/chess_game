// CM3070 FP code
// chessboard.go defines chessboard rendering for the index page

package chessboard

import (
	"fmt"
	"html/template"
	"strings"
)

// ChessBoard groups all board squares
type ChessBoard struct {
	squares []ChessBoardSquare
}

// NewChessBoard creates a board with 64 squares
func NewChessBoard() *ChessBoard {
	board := &ChessBoard{
		squares: make([]ChessBoardSquare, 0, 64),
	}

	for i := 0; i < 64; i++ {
		board.squares = append(board.squares, NewChessBoardSquare(i))
	}

	return board
}

// Draw renders the board wrapper, labels, and squares
func (c *ChessBoard) Draw() template.HTML {
	var htmlBuilder strings.Builder

	htmlBuilder.WriteString(`<div class="chess_board_wrapper">`)

	htmlBuilder.WriteString(`<div class="board_ranks board_ranks_left">`)
	htmlBuilder.WriteString(generateRankLabels())
	htmlBuilder.WriteString(`</div>`)

	htmlBuilder.WriteString(string(c.DrawChessBoardSquares()))

	htmlBuilder.WriteString(`<div class="board_spacer"></div>`)

	htmlBuilder.WriteString(`<div class="board_files board_files_bottom">`)
	htmlBuilder.WriteString(generateFileLabels())
	htmlBuilder.WriteString(`</div>`)

	htmlBuilder.WriteString(`</div>`)

	return template.HTML(htmlBuilder.String())
}

// DrawChessBoardSquares renders only square tiles
func (c *ChessBoard) DrawChessBoardSquares() template.HTML {
	var htmlBuilder strings.Builder

	htmlBuilder.WriteString(`<div class="chess_board">`)

	for _, square := range c.squares {
		squareClass := "chess_board_square_dark"
		if square.IsLight {
			squareClass = "chess_board_square_light"
		}

		fmt.Fprintf(
			&htmlBuilder,
			`<div class="chess_board_square %s" data-sequence="%d"></div>`,
			squareClass,
			square.Sequence,
		)
	}

	htmlBuilder.WriteString(`</div>`)

	return template.HTML(htmlBuilder.String())
}

// generateFileLabels builds file labels
func generateFileLabels() string {
	var htmlBuilder strings.Builder

	for _, file := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		htmlBuilder.WriteString(`<span class="board_label">`)
		htmlBuilder.WriteString(file)
		htmlBuilder.WriteString(`</span>`)
	}

	return htmlBuilder.String()
}

// generateRankLabels builds rank labels
func generateRankLabels() string {
	var htmlBuilder strings.Builder

	for _, rank := range []string{"8", "7", "6", "5", "4", "3", "2", "1"} {
		htmlBuilder.WriteString(`<span class="board_label">`)
		htmlBuilder.WriteString(rank)
		htmlBuilder.WriteString(`</span>`)
	}

	return htmlBuilder.String()
}
