// CM3070 FP code
// registry.go - registers the movement strategies

package movement

import "fmt"
import pieces "go_backend/game/piece"

// getStrategy returns the appropriate PieceMovementStrategy for a given piece kind
func getStrategy(kind pieces.PieceKind) PieceMovementStrategy {
	switch kind {
	case pieces.Pawn:
		return PawnStrategy{}
	default:
		return nil
	}
}

// ValidateMoveByStrategy validates movement by PieceMovementStrategy
func ValidateMoveByStrategy(kind pieces.PieceKind, fromFile, fromRank, toFile, toRank int, color pieces.PieceColor) error {
	strategy := getStrategy(kind)
	if strategy == nil {
		// Other piece strategies are pending; keep permissive for now
		return nil
	}

	legal := strategy.LegalMoves(
		MovementBoard{Color: color},
		Square{File: fromFile, Rank: fromRank},
	)
	for _, mv := range legal {
		sq, ok := mv.(Square)
		if !ok {
			continue
		}
		if sq.File == toFile && sq.Rank == toRank {
			return nil
		}
	}
	return fmt.Errorf("Invalid %s movement", strategy.Name())
}
