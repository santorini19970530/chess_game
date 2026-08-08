// CM3070 FP code
// shogi_lance.go - implements the shogi lance movement strategy

package movement

// ShogiLanceStrategy - slides forward only (shogi kyo)
type ShogiLanceStrategy struct{}

// Name - returns the piece strategy name
func (ShogiLanceStrategy) Name() string { return "ShogiLance" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiLanceStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return collectShogiSlidingMoves(ctx, src, [][2]int{{0, shogiForward(ctx.Color)}}, 8)
}
