// CM3070 FP code
// commandlist.go - lists supported chess command strings

package command

import pieces "go_backend/game/piece"

// commandPieceMap maps piece command letters to piece kinds
// p: pawn, r: rook, n: knight, b: bishop, q: queen, k: king
var CommandPieceMap = map[string]pieces.PieceKind{
	"p": pieces.Pawn,
	"r": pieces.Rook,
	"n": pieces.Knight,
	"b": pieces.Bishop,
	"q": pieces.Queen,
	"k": pieces.King,
}

// standardSANPieceLetter maps PieceKind to standard SAN piece letters
// sAN omits pawn letter; we keep "P" here for explicit internal mapping
var StandardSANPieceLetter = map[pieces.PieceKind]string{
	pieces.Pawn:   "P",
	pieces.Rook:   "R",
	pieces.Knight: "N",
	pieces.Bishop: "B",
	pieces.Queen:  "Q",
	pieces.King:   "K",
}

// promotionPieceMap maps promotion suffix letters to promotion piece kinds
// standard promotion letters: q, r, b, n
var PromotionPieceMap = map[string]pieces.PieceKind{
	"q": pieces.Queen,
	"r": pieces.Rook,
	"b": pieces.Bishop,
	"n": pieces.Knight,
}
