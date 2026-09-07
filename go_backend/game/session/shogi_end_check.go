// CM3070 FP code
// shogi_end_check.go - shogi continuous-check ply-end strategy (FESA 3.12)

package session

// ShogiContinuousCheckStrategy - one-sided repeating checks in a fourfold lose
type ShogiContinuousCheckStrategy struct{}

func (ShogiContinuousCheckStrategy) Name() string { return "shogi_continuous_check" }

// AfterPly - ends when the fourfold cycle is continuous check by one side
func (ShogiContinuousCheckStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || ctx.PerpetualCheckLoser == "" {
		return false, GameOutcome{}
	}
	return true, xiangqiLossOutcome(
		"continuous_check",
		ctx.PerpetualCheckLoser,
		"Continuous check. "+sideLabelFromText(ctx.PerpetualCheckLoser)+" loses.",
		ctx.LegalMoves,
	)
}
