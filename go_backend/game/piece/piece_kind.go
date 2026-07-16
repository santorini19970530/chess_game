// CM3070 FP code
// piece_kind.go - define the kinds of the chess and their motions

package pieces

type PieceKind string

const (
	Pawn     PieceKind = "pawn"
	Rook     PieceKind = "rook"
	Knight   PieceKind = "knight"
	Bishop   PieceKind = "bishop"
	Queen    PieceKind = "queen"
	King     PieceKind = "king"
	Advisor  PieceKind = "advisor"
	Cannon   PieceKind = "cannon"
	Elephant PieceKind = "elephant"

	// Shogi (and promoted forms).
	Lance           PieceKind = "lance"
	Silver          PieceKind = "silver"
	Gold            PieceKind = "gold"
	PromotedPawn    PieceKind = "promoted_pawn"
	PromotedLance   PieceKind = "promoted_lance"
	PromotedKnight  PieceKind = "promoted_knight"
	PromotedSilver  PieceKind = "promoted_silver"
	Dragon          PieceKind = "dragon" // promoted rook
	Horse           PieceKind = "horse"  // promoted bishop
)
