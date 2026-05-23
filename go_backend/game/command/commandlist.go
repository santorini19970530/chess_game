// commandlist.go

// Supported command standards (string-level representation only)
// 1) UCI/long algebraic style: "e2e4", "g1f3", "e7e8q" (promotion suffix)
// 2) Piece-prefixed style used by this project: "<piece><from><to>", e.g. "ng1f3"

package command

import pieces "go_backend/game/piece"

// CommandPieceMap maps piece command letters to piece kinds
// p: pawn, r: rook, n: knight, b: bishop, q: queen, k: king
var CommandPieceMap = map[string]pieces.PieceKind{
	"p": pieces.Pawn,
	"r": pieces.Rook,
	"n": pieces.Knight,
	"b": pieces.Bishop,
	"q": pieces.Queen,
	"k": pieces.King,
}

// StandardSANPieceLetter maps PieceKind to standard SAN piece letters
// SAN omits pawn letter; we keep "P" here for explicit internal mapping
var StandardSANPieceLetter = map[pieces.PieceKind]string{
	pieces.Pawn:   "P",
	pieces.Rook:   "R",
	pieces.Knight: "N",
	pieces.Bishop: "B",
	pieces.Queen:  "Q",
	pieces.King:   "K",
}

// PromotionPieceMap maps promotion suffix letters to promotion piece kinds
// Standard promotion letters: q, r, b, n
var PromotionPieceMap = map[string]pieces.PieceKind{
	"q": pieces.Queen,
	"r": pieces.Rook,
	"b": pieces.Bishop,
	"n": pieces.Knight,
}
