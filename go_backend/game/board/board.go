// CM3070 FP code
// chessboard.go defines chessboard rendering for the index page

package chessboard

import (
	"fmt"
	pieces "go_backend/game/piece"
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

	type pieceRender struct {
		src   string
		color string
		kind  string
	}
	pieceAt := make(map[string]pieceRender, len(pieces.ChessPieces))
	for _, p := range pieces.ChessPieces {
		key := fmt.Sprintf("%d_%d", p.File, p.Rank)
		pieceAt[key] = pieceRender{
			src:   "/" + p.ImgFile,
			color: string(p.Color),
			kind:  string(p.Kind),
		}
	}

	for _, square := range c.squares {
		squareClass := "chess_board_square_dark"
		if square.IsLight {
			squareClass = "chess_board_square_light"
		}

		file := (square.Sequence % 8) + 1
		rank := 8 - (square.Sequence / 8)
		key := fmt.Sprintf("%d_%d", file, rank)

		fmt.Fprintf(
			&htmlBuilder,
			`<div class="chess_board_square %s" data-sequence="%d">`,
			squareClass,
			square.Sequence,
		)

		// draw also the chess piece if there is
		if pieceMeta, ok := pieceAt[key]; ok {
			fmt.Fprintf(
				&htmlBuilder,
				`<img class="piece_img" src="%s" alt="piece_%s" data-color="%s" data-kind="%s" draggable="false">`,
				pieceMeta.src,
				key,
				pieceMeta.color,
				pieceMeta.kind,
			)
		}
		htmlBuilder.WriteString(`</div>`)
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
