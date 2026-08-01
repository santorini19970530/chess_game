package movement

// ShogiKnightStrategy - forward 2 then side 1 (shogi kei)
type ShogiKnightStrategy struct{}

// Name - returns the piece strategy name
func (ShogiKnightStrategy) Name() string { return "ShogiKnight" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiKnightStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	fwd := shogiForward(ctx.Color)
	legal := make([]any, 0, 2)
	legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File-1, src.Rank+2*fwd)
	legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File+1, src.Rank+2*fwd)
	return legal
}
