// commandlist.go

// Supported command standards (string-level representation only)
// 1) UCI/long algebraic style: "e2e4", "g1f3", "e7e8q" (promotion suffix)
// 2) Piece-prefixed style used by this project: "<piece><from><to>", e.g. "ng1f3"

package pieces

// CommandPieceMap maps piece command letters to piece kinds
// p: pawn, r: rook, n: knight, b: bishop, q: queen, k: king
var CommandPieceMap = map[string]PieceKind{
	"p": Pawn,
	"r": Rook,
	"n": Knight,
	"b": Bishop,
	"q": Queen,
	"k": King,
}

// StandardSANPieceLetter maps PieceKind to standard SAN piece letters
// SAN omits pawn letter; we keep "P" here for explicit internal mapping
var StandardSANPieceLetter = map[PieceKind]string{
	Pawn:   "P",
	Rook:   "R",
	Knight: "N",
	Bishop: "B",
	Queen:  "Q",
	King:   "K",
}

// PromotionPieceMap maps promotion suffix letters to promotion piece kinds
// Standard promotion letters: q, r, b, n
var PromotionPieceMap = map[string]PieceKind{
	"q": Queen,
	"r": Rook,
	"b": Bishop,
	"n": Knight,
}
