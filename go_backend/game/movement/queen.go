// CM3070 FP code
// queen.go - implements the queen movement strategy

package movement

// QueenStrategy handles queen movement rules
type QueenStrategy struct{}

// Name - returns the name of the queen strategy
func (q QueenStrategy) Name() string { return "Queen" }

// LegalMoves - simulates queen movement as rook + bishop movement
func (q QueenStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}

	directions := [][2]int{
		{1, 0}, { -1, 0}, {0, 1}, {0, -1}, // rook movement
		{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, // bishop movement
	}

	// queen can move like rook and bishop
	return collectSlidingMoves(ctx, src, directions, 27)
}
