package movement

// ShogiPawnStrategy - one step forward (shogi fu)
type ShogiPawnStrategy struct{}

// Name - returns the piece strategy name
func (ShogiPawnStrategy) Name() string { return "ShogiPawn" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiPawnStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return appendShogiIfEnemyOrEmpty(nil, ctx, src.File, src.Rank+shogiForward(ctx.Color))
}
