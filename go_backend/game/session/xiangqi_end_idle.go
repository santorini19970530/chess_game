// CM3070 FP code
// xiangqi_end_idle.go - xiangqi house-rule no-capture ply-end strategy

package session

// XiangqiNoCaptureStrategy - 60 ply with no capture is a house-rule draw
type XiangqiNoCaptureStrategy struct{}

func (XiangqiNoCaptureStrategy) Name() string { return "xiangqi_no_capture" }

// AfterPly - ends as a draw when idle ply reaches the house count
func (XiangqiNoCaptureStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || ctx.IdlePly < xiangqiNoCapturePlies {
		return false, GameOutcome{}
	}
	return true, GameOutcome{
		Status:     "draw_no_capture",
		LegalMoves: ctx.LegalMoves,
		Message:    "Draw by no capture (house rule, 60 ply).",
	}
}
