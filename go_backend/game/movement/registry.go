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
	case pieces.Rook:
		return RookStrategy{}
	case pieces.Bishop:
		return BishopStrategy{}
	case pieces.Knight:
		return KnightStrategy{}
	case pieces.Queen:
		return QueenStrategy{}
	case pieces.King:
		return KingStrategy{}
	default:
		return nil
	}
}

// getXiangqiStrategy returns Xiangqi piece movement strategies (9×10 board rules).
func getXiangqiStrategy(kind pieces.PieceKind) PieceMovementStrategy {
	switch kind {
	case pieces.Rook:
		return XiangqiChariotStrategy{}
	case pieces.Knight:
		return XiangqiHorseStrategy{}
	case pieces.Elephant:
		return XiangqiElephantStrategy{}
	case pieces.Advisor:
		return XiangqiAdvisorStrategy{}
	case pieces.King:
		return XiangqiGeneralStrategy{}
	case pieces.Cannon:
		return XiangqiCannonStrategy{}
	case pieces.Pawn:
		return XiangqiSoldierStrategy{}
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

// ValidateXiangqiMoveByStrategy validates a Xiangqi geometry move (no check filter).
func ValidateXiangqiMoveByStrategy(kind pieces.PieceKind, fromFile, fromRank, toFile, toRank int, color pieces.PieceColor) error {
	strategy := getXiangqiStrategy(kind)
	if strategy == nil {
		return fmt.Errorf("unsupported xiangqi piece kind %q", kind)
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

// XiangqiLegalSquares returns pseudo-legal destinations for a Xiangqi piece on the current board.
func XiangqiLegalSquares(kind pieces.PieceKind, color pieces.PieceColor, fromFile, fromRank int) []Square {
	strategy := getXiangqiStrategy(kind)
	if strategy == nil {
		return nil
	}
	raw := strategy.LegalMoves(
		MovementBoard{Color: color},
		Square{File: fromFile, Rank: fromRank},
	)
	out := make([]Square, 0, len(raw))
	for _, mv := range raw {
		if sq, ok := mv.(Square); ok {
			out = append(out, sq)
		}
	}
	return out
}
