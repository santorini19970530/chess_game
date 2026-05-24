// CM3070 FP code
// special_moves.go - implements special move rules

package engine

import pieces "go_backend/game/piece"

// LastMoveInfo is a lightweight move snapshot for special-move checks.
type LastMoveInfo struct {
	FromFile       int
	FromRank       int
	ToFile         int
	ToRank         int
	PieceKind      pieces.PieceKind
	Color          pieces.PieceColor
	PawnDoubleStep bool
}

// CanEnPassant validates en passant rule conditions.
func CanEnPassant(
	source pieces.ChessPiece,
	fromFile, fromRank, toFile, toRank int,
	destinationOccupied bool,
	lastMove *LastMoveInfo,
	adjacentPawn pieces.ChessPiece,
	adjacentPawnFound bool,
) bool {
	if source.Kind != pieces.Pawn {
		return false
	}
	if destinationOccupied {
		return false
	}

	dir := 1
	if source.Color == pieces.Black {
		dir = -1
	}
	if toRank-fromRank != dir || absInt(toFile-fromFile) != 1 {
		return false
	}

	if lastMove == nil {
		return false
	}
	if lastMove.PieceKind != pieces.Pawn || !lastMove.PawnDoubleStep {
		return false
	}
	if lastMove.Color == source.Color {
		return false
	}
	if lastMove.ToFile != toFile || lastMove.ToRank != fromRank {
		return false
	}
	if !adjacentPawnFound {
		return false
	}
	return adjacentPawn.Kind == pieces.Pawn && adjacentPawn.Color != source.Color
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
