package movement

// ShogiDragonStrategy - promoted rook: orthogonal slides plus diagonal steps
type ShogiDragonStrategy struct{}

// Name - returns the piece strategy name
func (ShogiDragonStrategy) Name() string { return "ShogiDragon" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiDragonStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	legal := collectShogiSlidingMoves(ctx, src, [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}, 20)
	for _, d := range [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}} {
		legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File+d[0], src.Rank+d[1])
	}
	return legal
}
