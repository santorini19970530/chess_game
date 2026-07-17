package session

import (
	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// EvaluateShogiGameOutcome uses Go legal-move generation (board + drops).
// No legal moves → loss for the side to move (tsume / no-move loss).
func EvaluateShogiGameOutcome() GameOutcome {
	whiteOK, blackOK := false, false
	for _, p := range pieces.ChessPieces {
		if p.Kind != pieces.King {
			continue
		}
		if p.Color == pieces.White {
			whiteOK = true
		} else {
			blackOK = true
		}
	}
	if !whiteOK {
		return GameOutcome{
			Status:      "checkmate",
			Winner:      "black",
			Loser:       "white",
			CheckedSide: "white",
			LegalMoves:  0,
			Message:     "Black wins (white king missing).",
		}
	}
	if !blackOK {
		return GameOutcome{
			Status:      "checkmate",
			Winner:      "white",
			Loser:       "black",
			CheckedSide: "black",
			LegalMoves:  0,
			Message:     "White wins (black king missing).",
		}
	}

	sideToMove := CurrentTurnColor()
	legal, err := shogiAllLegalUCIMoves()
	if err != nil {
		return GameOutcome{Status: "in_progress", Message: "in progress"}
	}
	legalCount := len(legal)
	inCheck := movement.ShogiCheckedColor() == sideToMove

	if legalCount > 0 {
		if inCheck {
			return GameOutcome{
				Status:      "check",
				CheckedSide: string(sideToMove),
				LegalMoves:  legalCount,
				Message:     sideLabel(sideToMove) + " is in check.",
			}
		}
		return GameOutcome{
			Status:     "in_progress",
			LegalMoves: legalCount,
		}
	}

	winner := opponentOf(sideToMove)
	msg := "Checkmate! " + sideLabel(winner) + " wins."
	if !inCheck {
		msg = "No legal moves! " + sideLabel(winner) + " wins."
	}
	return GameOutcome{
		Status:      "checkmate",
		Winner:      string(winner),
		Loser:       string(sideToMove),
		CheckedSide: string(sideToMove),
		LegalMoves:  0,
		Message:     msg,
	}
}
