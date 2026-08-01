package movement

// ShogiBishopStrategy - diagonal slides (shogi kaku)
type ShogiBishopStrategy struct{}

// Name - returns the piece strategy name
func (ShogiBishopStrategy) Name() string { return "ShogiBishop" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiBishopStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return collectShogiSlidingMoves(ctx, src, [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}, 16)
}
