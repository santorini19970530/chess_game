package movement

// ShogiGoldStrategy - gold-general steps; also used for promoted pawn/lance/knight/silver
type ShogiGoldStrategy struct{}

// Name - returns the piece strategy name
func (ShogiGoldStrategy) Name() string { return "ShogiGold" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiGoldStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return appendShogiSteps(nil, ctx, src, shogiGoldDeltas(ctx.Color))
}
