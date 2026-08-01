// CM3070 FP code
// board_draw.go - renders board html for the index page

package handlers

import (
	"fmt"
	chessboard "go_backend/game/board"
	pieces "go_backend/game/piece"
	"html/template"
	"strings"
)

// DrawChessBoard - renders board wrapper, labels, and squares as html
func DrawChessBoard(c *chessboard.ChessBoard) template.HTML {
	var htmlBuilder strings.Builder

	fmt.Fprintf(
		&htmlBuilder,
		`<div class="chess_board_wrapper" style="--board-files: %d; --board-ranks: %d;">`,
		c.Files,
		c.Ranks,
	)

	htmlBuilder.WriteString(`<div class="board_ranks board_ranks_left">`)
	htmlBuilder.WriteString(generateRankLabels(c.Ranks))
	htmlBuilder.WriteString(`</div>`)

	htmlBuilder.WriteString(string(DrawChessBoardSquares(c)))

	htmlBuilder.WriteString(`<div class="board_spacer"></div>`)

	htmlBuilder.WriteString(`<div class="board_files board_files_bottom">`)
	htmlBuilder.WriteString(generateFileLabels(c.Files))
	htmlBuilder.WriteString(`</div>`)

	htmlBuilder.WriteString(`</div>`)

	return template.HTML(htmlBuilder.String())
}

// DrawChessBoardSquares - renders only square tiles
func DrawChessBoardSquares(c *chessboard.ChessBoard) template.HTML {
	var htmlBuilder strings.Builder

	htmlBuilder.WriteString(`<div class="chess_board">`)

	type pieceRender struct {
		src   string
		color string
		kind  string
	}
	pieceAt := make(map[string]pieceRender)
	// SSR initial pieces only for classic chess start layout
	if c.Files == 8 && c.Ranks == 8 {
		pieceAt = make(map[string]pieceRender, len(pieces.ChessPieces))
		for _, p := range pieces.ChessPieces {
			key := fmt.Sprintf("%d_%d", p.File, p.Rank)
			pieceAt[key] = pieceRender{
				src:   "/" + p.ImgFile,
				color: string(p.Color),
				kind:  string(p.Kind),
			}
		}
	}

	for _, square := range c.Squares() {
		squareClass := "chess_board_square_dark"
		if square.IsLight {
			squareClass = "chess_board_square_light"
		}

		file, rank := chessboard.FileRankFromSequence(square.Sequence, c.Files, c.Ranks)
		key := fmt.Sprintf("%d_%d", file, rank)
		extra := chessboard.SquareCueClasses(c.Files, c.Ranks, file, rank)
		if extra != "" {
			squareClass = squareClass + " " + extra
		}

		fmt.Fprintf(
			&htmlBuilder,
			`<div class="chess_board_square %s" data-sequence="%d" data-file="%d" data-rank="%d">`,
			squareClass,
			square.Sequence,
			file,
			rank,
		)

		if pieceMeta, ok := pieceAt[key]; ok {
			fmt.Fprintf(
				&htmlBuilder,
				`<img class="piece_img" src="%s" alt="piece_%s" data-color="%s" data-kind="%s" draggable="true">`,
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

// generateFileLabels - builds file letter labels (a…)
func generateFileLabels(files int) string {
	var htmlBuilder strings.Builder
	for i := 0; i < files; i++ {
		htmlBuilder.WriteString(`<span class="board_label">`)
		htmlBuilder.WriteByte(byte('a' + i))
		htmlBuilder.WriteString(`</span>`)
	}
	return htmlBuilder.String()
}

// generateRankLabels - builds rank number labels (n…1)
func generateRankLabels(ranks int) string {
	var htmlBuilder strings.Builder
	for r := ranks; r >= 1; r-- {
		htmlBuilder.WriteString(`<span class="board_label">`)
		htmlBuilder.WriteString(fmt.Sprintf("%d", r))
		htmlBuilder.WriteString(`</span>`)
	}
	return htmlBuilder.String()
}
