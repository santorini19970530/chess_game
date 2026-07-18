// CM3070 FP code
// chessboard.go defines chessboard rendering for the index page

package chessboard

import (
	"fmt"
	pieces "go_backend/game/piece"
	"html/template"
	"strings"
)

// ChessBoard groups all board squares for a files×ranks grid.
type ChessBoard struct {
	Files   int
	Ranks   int
	squares []ChessBoardSquare
}

// NewChessBoard creates an 8×8 chess board (SSR default).
func NewChessBoard() *ChessBoard {
	return NewBoard(8, 8)
}

// NewBoard creates a files×ranks div-square board (Xiangqi 9×10, Shogi 9×9, …).
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

// SequenceByFileRank maps 1-based file/rank to data-sequence (rank max at top).
func SequenceByFileRank(file, rank, files, maxRank int) int {
	return (maxRank-rank)*files + (file - 1)
}

// FileRankFromSequence is the inverse of SequenceByFileRank.
func FileRankFromSequence(sequence, files, maxRank int) (file, rank int) {
	file = (sequence % files) + 1
	rank = maxRank - (sequence / files)
	return file, rank
}

// Draw renders the board wrapper, labels, and squares
func (c *ChessBoard) Draw() template.HTML {
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

	htmlBuilder.WriteString(string(c.DrawChessBoardSquares()))

	htmlBuilder.WriteString(`<div class="board_spacer"></div>`)

	htmlBuilder.WriteString(`<div class="board_files board_files_bottom">`)
	htmlBuilder.WriteString(generateFileLabels(c.Files))
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

	for _, square := range c.squares {
		squareClass := "chess_board_square_dark"
		if square.IsLight {
			squareClass = "chess_board_square_light"
		}

		file, rank := FileRankFromSequence(square.Sequence, c.Files, c.Ranks)
		key := fmt.Sprintf("%d_%d", file, rank)
		extra := squareCueClasses(c.Files, c.Ranks, file, rank)
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

// squareCueClasses adds Xiangqi edge/river/palace cues (9×10 only). No board PNG.
func squareCueClasses(files, ranks, file, rank int) string {
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

func generateFileLabels(files int) string {
	var htmlBuilder strings.Builder
	for i := 0; i < files; i++ {
		htmlBuilder.WriteString(`<span class="board_label">`)
		htmlBuilder.WriteByte(byte('a' + i))
		htmlBuilder.WriteString(`</span>`)
	}
	return htmlBuilder.String()
}

func generateRankLabels(ranks int) string {
	var htmlBuilder strings.Builder
	for r := ranks; r >= 1; r-- {
		htmlBuilder.WriteString(`<span class="board_label">`)
		htmlBuilder.WriteString(fmt.Sprintf("%d", r))
		htmlBuilder.WriteString(`</span>`)
	}
	return htmlBuilder.String()
}
