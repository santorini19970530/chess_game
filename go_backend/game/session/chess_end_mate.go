// CM3070 FP code
// chess_end_mate.go - chess mate and stalemate ply-end strategy

package session

// ChessMateOrStalemateStrategy - no legal move is mate if in check, else stalemate
type ChessMateOrStalemateStrategy struct{}

func (ChessMateOrStalemateStrategy) Name() string { return "chess_mate_or_stalemate" }

// AfterPly - ends when the side to move has no legal move
func (ChessMateOrStalemateStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || ctx.LegalMoves > 0 {
		return false, GameOutcome{}
	}
	loser := ctx.SideToMove
	if loser == "" {
		loser = "white"
	}
	if ctx.InCheck {
		winner := "black"
		if loser == "black" {
			winner = "white"
		}
		return true, GameOutcome{
			Status:      "checkmate",
			Winner:      winner,
			Loser:       loser,
			CheckedSide: loser,
			LegalMoves:  0,
			Message:     "Checkmate! " + sideLabelFromText(winner) + " wins.",
		}
	}
	return true, GameOutcome{
		Status:     "stalemate",
		LegalMoves: 0,
		Message:    "Draw by stalemate.",
	}
}
