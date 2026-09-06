// CM3070 FP code
// chess_end_repetition.go - chess threefold-repetition ply-end strategy

package session

// ChessThreefoldStrategy - same position three times is a draw
type ChessThreefoldStrategy struct{}

func (ChessThreefoldStrategy) Name() string { return "chess_threefold" }

// AfterPly - ends as a draw when the position count is at least three and a legal move remains
func (ChessThreefoldStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || ctx.LegalMoves == 0 || ctx.PositionCount < 3 {
		return false, GameOutcome{}
	}
	return true, GameOutcome{
		Status:     "draw_threefold_repetition",
		LegalMoves: ctx.LegalMoves,
		Message:    "Draw by threefold repetition.",
	}
}
