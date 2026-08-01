// CM3070 FP code
// shogi_rook.go - implements the shogi rook movement strategy

package movement

// ShogiRookStrategy - orthogonal slides (shogi hisha)
type ShogiRookStrategy struct{}

// Name - returns the piece strategy name
func (ShogiRookStrategy) Name() string { return "ShogiRook" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiRookStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return collectShogiSlidingMoves(ctx, src, [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}, 16)
}
