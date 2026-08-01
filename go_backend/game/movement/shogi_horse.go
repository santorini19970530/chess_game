package movement

// ShogiHorseStrategy - promoted bishop: diagonal slides plus orthogonal steps
type ShogiHorseStrategy struct{}

// Name - returns the piece strategy name
func (ShogiHorseStrategy) Name() string { return "ShogiHorse" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiHorseStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	legal := collectShogiSlidingMoves(ctx, src, [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}, 20)
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File+d[0], src.Rank+d[1])
	}
	return legal
}
