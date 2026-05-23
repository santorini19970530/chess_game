// CM3070 FP code
// strategy.go
package movement

import pieces "go_backend/game/piece"

// Square is a board coordinate
type Square struct {
	File int
	Rank int
}

// MovementBoard carries context needed by movement strategies
type MovementBoard struct {
	Color pieces.PieceColor
}

// PieceMovementStrategy defines piece-specific move rules
type PieceMovementStrategy interface {
	// LegalMoves returns possible moves from a source square
	LegalMoves(board any, from any) []any

	// Name is used for logging/debugging
	Name() string
}
