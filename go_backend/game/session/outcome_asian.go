// CM3070 FP code
// outcome_asian.go - shared xiangqi/shogi outcome evaluation skeleton

package session

import (
	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// asianOutcomeHooks - per-game diffs for Xiangqi/Shogi (no-draw, no-move = loss)
type asianOutcomeHooks struct {
	missingPieceNoun string
	allLegalMoves    func() ([]string, error)
	checkedColor     func() pieces.PieceColor
	zeroLegalMessage func(inCheck bool, winner pieces.PieceColor) string
}

// evaluateAsianGameOutcome - shared check/mate skeleton; hooks keep rule wording and move generation
func evaluateAsianGameOutcome(h asianOutcomeHooks) GameOutcome {
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
			Message:     "Black wins (white " + h.missingPieceNoun + " missing).",
		}
	}
	if !blackOK {
		return GameOutcome{
			Status:      "checkmate",
			Winner:      "white",
			Loser:       "black",
			CheckedSide: "black",
			LegalMoves:  0,
			Message:     "White wins (black " + h.missingPieceNoun + " missing).",
		}
	}

	sideToMove := CurrentTurnColor()
	legal, err := h.allLegalMoves()
	if err != nil {
		return GameOutcome{Status: "in_progress", Message: "in progress"}
	}
	legalCount := len(legal)
	inCheck := h.checkedColor() == sideToMove

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
	return GameOutcome{
		Status:      "checkmate",
		Winner:      string(winner),
		Loser:       string(sideToMove),
		CheckedSide: string(sideToMove),
		LegalMoves:  0,
		Message:     h.zeroLegalMessage(inCheck, winner),
	}
}

// EvaluateXiangqiGameOutcome - Xiangqi outcome: no legal moves is always a loss (mate or stalemate)
func EvaluateXiangqiGameOutcome() GameOutcome {
	return evaluateAsianGameOutcome(asianOutcomeHooks{
		missingPieceNoun: "general",
		allLegalMoves:    xiangqiAllLegalUCIMoves,
		checkedColor:     movement.XiangqiCheckedColor,
		zeroLegalMessage: func(inCheck bool, winner pieces.PieceColor) string {
			if inCheck {
				return "Checkmate! " + sideLabel(winner) + " wins."
			}
			return "Stalemate! " + sideLabel(winner) + " wins (Xiangqi rule)."
		},
	})
}

// EvaluateShogiGameOutcome - Shogi outcome: no legal moves (board + drops) is a loss
func EvaluateShogiGameOutcome() GameOutcome {
	return evaluateAsianGameOutcome(asianOutcomeHooks{
		missingPieceNoun: "king",
		allLegalMoves:    shogiAllLegalUCIMoves,
		checkedColor:     movement.ShogiCheckedColor,
		zeroLegalMessage: func(inCheck bool, winner pieces.PieceColor) string {
			if inCheck {
				return "Checkmate! " + sideLabel(winner) + " wins."
			}
			return "No legal moves! " + sideLabel(winner) + " wins."
		},
	})
}
