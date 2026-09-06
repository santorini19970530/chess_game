// CM3070 FP code
// xiangqi_end_repetition.go - xiangqi mutual-repetition ply-end strategy

package session

// XiangqiMutualRepetitionStrategy - threefold with no check/chase claim is a draw
type XiangqiMutualRepetitionStrategy struct{}

func (XiangqiMutualRepetitionStrategy) Name() string { return "xiangqi_mutual_repetition" }

// AfterPly - ends as a draw when the position has repeated three times
func (XiangqiMutualRepetitionStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || !ctx.CycleRepeat {
		return false, GameOutcome{}
	}
	return true, GameOutcome{
		Status:     "draw_mutual_repetition",
		LegalMoves: ctx.LegalMoves,
		Message:    "Draw by repetition.",
	}
}
