// CM3070 FP code
// board_square_html.go - square and piece html helpers for board ssr

package handlers

import (
	"fmt"
	chessboard "go_backend/game/board"
	pieces "go_backend/game/piece"
	"strings"
)

// boardPieceHTML - one piece image to place on a square
type boardPieceHTML struct {
	src   string
	color string
	kind  string
}

// chessStartPieces - maps file_rank → piece for classic 8×8 start only
func chessStartPieces(c *chessboard.ChessBoard) map[string]boardPieceHTML {
	if c.Files != 8 || c.Ranks != 8 {
		return nil
	}
	out := make(map[string]boardPieceHTML, len(pieces.ChessPieces))
	for _, p := range pieces.ChessPieces {
		key := fmt.Sprintf("%d_%d", p.File, p.Rank)
		out[key] = boardPieceHTML{
			src:   "/" + p.ImgFile,
			color: string(p.Color),
			kind:  string(p.Kind),
		}
	}
	return out
}

// squareCSSClass - light/dark plus optional xiangqi cue classes
func squareCSSClass(c *chessboard.ChessBoard, square chessboard.ChessBoardSquare, file, rank int) string {
	class := "chess_board_square_dark"
	if square.IsLight {
		class = "chess_board_square_light"
	}
	if extra := chessboard.SquareCueClasses(c.Files, c.Ranks, file, rank); extra != "" {
		class = class + " " + extra
	}
	return class
}

// writeSquareHTML - writes one square div and optional piece img
func writeSquareHTML(b *strings.Builder, c *chessboard.ChessBoard, square chessboard.ChessBoardSquare, pieceAt map[string]boardPieceHTML) {
	file, rank := chessboard.FileRankFromSequence(square.Sequence, c.Files, c.Ranks)
	key := fmt.Sprintf("%d_%d", file, rank)
	fmt.Fprintf(
		b,
		`<div class="chess_board_square %s" data-sequence="%d" data-file="%d" data-rank="%d">`,
		squareCSSClass(c, square, file, rank),
		square.Sequence,
		file,
		rank,
	)
	if piece, ok := pieceAt[key]; ok {
		writePieceImg(b, key, piece)
	}
	b.WriteString(`</div>`)
}

// writePieceImg - writes a draggable piece image tag
func writePieceImg(b *strings.Builder, key string, piece boardPieceHTML) {
	fmt.Fprintf(
		b,
		`<img class="piece_img" src="%s" alt="piece_%s" data-color="%s" data-kind="%s" draggable="true">`,
		piece.src,
		key,
		piece.color,
		piece.kind,
	)
}
