// CM3070 FP code
// xiangqi_end_mate.go - xiangqi mate and stalemate-as-loss ply-end strategy

package session

// XiangqiMateOrStalemateStrategy - no legal moves is a loss (mate or stalemate)
type XiangqiMateOrStalemateStrategy struct{}

func (XiangqiMateOrStalemateStrategy) Name() string { return "xiangqi_mate_or_stalemate" }

// AfterPly - ends when the side to move has no legal move
func (XiangqiMateOrStalemateStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || ctx.LegalMoves > 0 {
		return false, GameOutcome{}
	}
	loser := ctx.SideToMove
	if loser == "" {
		loser = "white"
	}
	winner := "black"
	if loser == "black" {
		winner = "white"
	}
	msg := "Stalemate! " + sideLabelFromText(winner) + " wins (Xiangqi rule)."
	if ctx.InCheck {
		msg = "Checkmate! " + sideLabelFromText(winner) + " wins."
	}
	return true, GameOutcome{
		Status:      "checkmate",
		Winner:      winner,
		Loser:       loser,
		CheckedSide: loser,
		LegalMoves:  0,
		Message:     msg,
	}
}
