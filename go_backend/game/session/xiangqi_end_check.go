// CM3070 FP code
// xiangqi_end_check.go - xiangqi perpetual-check ply-end strategy

package session

// XiangqiPerpetualCheckStrategy - one-sided repeating checks lose
type XiangqiPerpetualCheckStrategy struct{}

func (XiangqiPerpetualCheckStrategy) Name() string { return "xiangqi_perpetual_check" }

// AfterPly - ends when the repeating cycle is perpetual check by one side
func (XiangqiPerpetualCheckStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || ctx.PerpetualCheckLoser == "" {
		return false, GameOutcome{}
	}
	return true, xiangqiLossOutcome(
		"perpetual_check",
		ctx.PerpetualCheckLoser,
		"Perpetual check. "+sideLabelFromText(ctx.PerpetualCheckLoser)+" loses.",
		ctx.LegalMoves,
	)
}
