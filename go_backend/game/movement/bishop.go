// CM3070 FP code
// bishop.go - implements the bishop movement strategy

package movement

// BishopStrategy handles bishop movement rules
type BishopStrategy struct{}

// Name - returns the name of the bishop strategy
func (b BishopStrategy) Name() string { return "Bishop" }

// LegalMoves - simulates bishop movement on diagonals
func (b BishopStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}

	directions := [][2]int{
		{1, 1},
		{1, -1},
		{-1, 1},
		{-1, -1},
	}
	return collectSlidingMoves(ctx, src, directions, 13)
}