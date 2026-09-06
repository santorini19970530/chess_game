// CM3070 FP code
// chess_end_fifty.go - chess fifty-move ply-end strategy

package session

// ChessFiftyMoveStrategy - 100 halfmoves without pawn or capture is a draw
type ChessFiftyMoveStrategy struct{}

func (ChessFiftyMoveStrategy) Name() string { return "chess_fifty_move" }

// AfterPly - ends as a draw when idle ply is at least 100 and a legal move remains
func (ChessFiftyMoveStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || ctx.LegalMoves == 0 || ctx.IdlePly < 100 {
		return false, GameOutcome{}
	}
	return true, GameOutcome{
		Status:     "draw_fifty_move_rule",
		LegalMoves: ctx.LegalMoves,
		Message:    "Draw by 50-move rule.",
	}
}
