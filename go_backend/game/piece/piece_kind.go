// CM3070 FP code
// piece_kind.go - define the kinds of the chess and their motions

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
