// strategy.go
package movement

// PieceMovementStrategy defines piece-specific move rules.
type PieceMovementStrategy interface {
	// LegalMoves returns possible moves from a source square.
	LegalMoves(board any, from any) []any

	// Name is used for logging/debugging.
	Name() string
}
