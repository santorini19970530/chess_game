// CM3070 FP code
// piece_kind.go - defines chess piece types

package pieces

type PieceKind string

const (
	Pawn   PieceKind = "pawn"
	Rook   PieceKind = "rook"
	Knight PieceKind = "knight"
	Bishop PieceKind = "bishop"
	Queen  PieceKind = "queen"
	King   PieceKind = "king"
)
