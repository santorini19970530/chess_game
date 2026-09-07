// CM3070 FP code
// shogi_end_sennichite.go - shogi fourfold-repetition ply-end strategy (FESA 5.2)

package session

// ShogiSennichiteStrategy - same board, hands, and side four times is a draw
type ShogiSennichiteStrategy struct{}

func (ShogiSennichiteStrategy) Name() string { return "shogi_sennichite" }

// AfterPly - ends as a draw when this position has occurred four times
func (ShogiSennichiteStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || !ctx.CycleRepeat {
		return false, GameOutcome{}
	}
	return true, GameOutcome{
		Status:     "draw_sennichite",
		LegalMoves: ctx.LegalMoves,
		Message:    "Draw by sennichite (fourfold repetition).",
	}
}
