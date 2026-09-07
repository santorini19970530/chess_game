// CM3070 FP code
// shogi_end_mate.go - shogi mate and no-move-loss ply-end strategy

package session

// ShogiMateOrStalemateStrategy - no legal moves is a loss (mate or no-move)
type ShogiMateOrStalemateStrategy struct{}

func (ShogiMateOrStalemateStrategy) Name() string { return "shogi_mate_or_stalemate" }

// AfterPly - ends when the side to move has no legal move
func (ShogiMateOrStalemateStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
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
	msg := "No legal moves! " + sideLabelFromText(winner) + " wins."
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
