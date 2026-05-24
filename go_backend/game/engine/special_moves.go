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

// CanCastle validates king-side / queen-side castling conditions,
// excluding check-related rules for now.
func CanCastle(
	source pieces.ChessPiece,
	fromFile, fromRank, toFile, toRank int,
	castlingRightsAvailable bool,
) bool {
	if source.Kind != pieces.King {
		return false
	}
	if !castlingRightsAvailable {
		return false
	}
	if fromFile != 5 {
		return false
	}
	if source.Color == pieces.White && fromRank != 1 {
		return false
	}
	if source.Color == pieces.Black && fromRank != 8 {
		return false
	}
	if toRank != fromRank {
		return false
	}

	kingSide := toFile == 7
	queenSide := toFile == 3
	if !kingSide && !queenSide {
		return false
	}

	rookFromFile := 1
	if kingSide {
		rookFromFile = 8
	}
	rook, rookFound := getPieceAt(rookFromFile, fromRank)
	if !rookFound || rook.Kind != pieces.Rook || rook.Color != source.Color {
		return false
	}

	// Squares between king and rook must be empty.
	pathFiles := []int{6, 7}
	if queenSide {
		pathFiles = []int{4, 3, 2}
	}
	for _, f := range pathFiles {
		if _, occupied := getPieceAt(f, fromRank); occupied {
			return false
		}
	}

	return true
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func getPieceAt(file, rank int) (pieces.ChessPiece, bool) {
	for _, p := range pieces.ChessPieces {
		if p.File == file && p.Rank == rank {
			return p, true
		}
	}
	return pieces.ChessPiece{}, false
}
