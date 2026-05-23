// CM3070 FP code
// rook.go - implements the rook movement strategy

package movement

// RookStrategy handles rook movement rules.
type RookStrategy struct{}

// Name - returns the name of the rook strategy
func (r RookStrategy) Name() string { return "Rook" }

// LegalMoves - simulates rook movement on straight lines
func (r RookStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}

	directions := [][2]int{
		{1, 0},
		{-1, 0},
		{0, 1},
		{0, -1},
	}
	return collectSlidingMoves(ctx, src, directions, 14)
}