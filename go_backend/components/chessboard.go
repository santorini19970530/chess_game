package components

import (
	"fmt"
	"html/template"
	"strings"
)

type ChessBoard struct {
	squares []ChessBoardSquare
}

func NewChessBoard() *ChessBoard {
	board := &ChessBoard{
		squares: make([]ChessBoardSquare, 0, 64),
	}

	for i := 0; i < 64; i++ {
		board.squares = append(board.squares, NewChessBoardSquare(i))
	}

	return board
}

func (c *ChessBoard) Draw() template.HTML {
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
