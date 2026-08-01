// CM3070 FP code
// shogi_king.go - implements the shogi king movement strategy

package movement

// ShogiKingStrategy - one step in any direction (shogi ou/gyoku)
type ShogiKingStrategy struct{}

// Name - returns the piece strategy name
func (ShogiKingStrategy) Name() string { return "ShogiKing" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiKingStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	legal := make([]any, 0, 8)
	for df := -1; df <= 1; df++ {
		for dr := -1; dr <= 1; dr++ {
			if df == 0 && dr == 0 {
				continue
			}
			legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File+df, src.Rank+dr)
		}
	}
	return legal
}
