// CM3070 FP code
// piece.go - defines the chess piece model
// each chess piece is having color, type, image file, and board position

package pieces

// ChessPiece stores piece type, image file, and board position, for each chess piece
type ChessPiece struct {
	Color   PieceColor
	Kind    PieceKind
	ImgFile string
	File    int
	Rank    int
}

// moving chess pieces
func (p *ChessPiece) Move(file int, rank int) {
	p.File = file
	p.Rank = rank
}
