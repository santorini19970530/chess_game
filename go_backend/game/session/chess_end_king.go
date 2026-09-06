// CM3070 FP code
// chess_end_king.go - chess missing-king ply-end strategy

package session

// ChessMissingKingStrategy - a side with no king has lost
type ChessMissingKingStrategy struct{}

func (ChessMissingKingStrategy) Name() string { return "chess_missing_king" }

// AfterPly - ends as checkmate when a king is off the board
func (ChessMissingKingStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil {
		return false, GameOutcome{}
	}
	if ctx.WhiteKings == 0 {
		return true, GameOutcome{
			Status:      "checkmate",
			Winner:      "black",
			Loser:       "white",
			CheckedSide: "white",
			LegalMoves:  0,
			Message:     "Checkmate! Black wins (king captured).",
		}
	}
	if ctx.BlackKings == 0 {
		return true, GameOutcome{
			Status:      "checkmate",
			Winner:      "white",
			Loser:       "black",
			CheckedSide: "black",
			LegalMoves:  0,
			Message:     "Checkmate! White wins (king captured).",
		}
	}
	return false, GameOutcome{}
}
