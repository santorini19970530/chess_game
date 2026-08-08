// CM3070 FP code
// board_draw.go - composes board html from labels and squares

package handlers

import (
	"fmt"
	chessboard "go_backend/game/board"
	"html/template"
	"strings"
)

// DrawChessBoard - renders board wrapper, labels, and squares as html
func DrawChessBoard(c *chessboard.ChessBoard) template.HTML {
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="chess_board_wrapper" style="--board-files: %d; --board-ranks: %d;">`, c.Files, c.Ranks)
	writeWrapped(&b, "board_ranks board_ranks_left", generateRankLabels(c.Ranks))
	b.WriteString(string(DrawChessBoardSquares(c)))
	b.WriteString(`<div class="board_spacer"></div>`)
	writeWrapped(&b, "board_files board_files_bottom", generateFileLabels(c.Files))
	b.WriteString(`</div>`)
	return template.HTML(b.String())
}

// DrawChessBoardSquares - renders only square tiles
func DrawChessBoardSquares(c *chessboard.ChessBoard) template.HTML {
	var b strings.Builder
	b.WriteString(`<div class="chess_board">`)
	pieceAt := chessStartPieces(c)
	for _, square := range c.Squares() {
		writeSquareHTML(&b, c, square, pieceAt)
	}
	b.WriteString(`</div>`)
	return template.HTML(b.String())
}

// writeWrapped - writes a div with class around inner html
func writeWrapped(b *strings.Builder, class, inner string) {
	b.WriteString(`<div class="`)
	b.WriteString(class)
	b.WriteString(`">`)
	b.WriteString(inner)
	b.WriteString(`</div>`)
}
