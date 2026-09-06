// CM3070 FP code
// xiangqi_end_chase.go - xiangqi perpetual-chase ply-end strategy

package session

// XiangqiPerpetualChaseStrategy - repeating chase of one unprotected non-general piece loses
type XiangqiPerpetualChaseStrategy struct{}

func (XiangqiPerpetualChaseStrategy) Name() string { return "xiangqi_perpetual_chase" }

// AfterPly - ends when the repeating cycle is a one-piece chase; king/soldier chase is not a loss
func (XiangqiPerpetualChaseStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || ctx.ChaseIsKingOrSoldierOnly || ctx.PerpetualChaseLoser == "" {
		return false, GameOutcome{}
	}
	return true, xiangqiLossOutcome(
		"perpetual_chase",
		ctx.PerpetualChaseLoser,
		"Perpetual chase. "+sideLabelFromText(ctx.PerpetualChaseLoser)+" loses.",
		ctx.LegalMoves,
	)
}
