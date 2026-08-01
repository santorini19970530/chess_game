// CM3070 FP code
// strategy.go - set up the movement strategy interface and the square coordinate

package movement

import pieces "go_backend/game/piece"

// square is a board coordinate
type Square struct {
	File int
	Rank int
}

// movementBoard carries context needed by movement strategies
type MovementBoard struct {
	Color pieces.PieceColor
}

// pieceMovementStrategy defines piece-specific move rules
type PieceMovementStrategy interface {
	// legalMoves returns possible moves from a source square
	LegalMoves(board any, from any) []any

	// name is used for logging/debugging
	Name() string
}
