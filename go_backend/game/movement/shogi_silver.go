// CM3070 FP code
// shogi_silver.go - implements the shogi silver movement strategy

package movement

// ShogiSilverStrategy - silver-general steps (shogi gin)
type ShogiSilverStrategy struct{}

// Name - returns the piece strategy name
func (ShogiSilverStrategy) Name() string { return "ShogiSilver" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiSilverStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return appendShogiSteps(nil, ctx, src, shogiSilverDeltas(ctx.Color))
}
