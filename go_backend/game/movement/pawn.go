// pawn.go
package movement

// PawnStrategy is a placeholder for pawn move rules.
type PawnStrategy struct{}

func (p PawnStrategy) Name() string { return "Pawn" }

func (p PawnStrategy) LegalMoves(board any, from any) []any {
	return nil
}
