package session

import (
	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// EvaluateXiangqiGameOutcome uses Go legal-move generation.
// No legal moves → loss for the side to move (checkmate or stalemate; both are losses in Xiangqi).
func EvaluateXiangqiGameOutcome() GameOutcome {
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
			Message:     "Black wins (white general missing).",
		}
	}
	if !blackOK {
		return GameOutcome{
			Status:      "checkmate",
			Winner:      "white",
			Loser:       "black",
			CheckedSide: "black",
			LegalMoves:  0,
			Message:     "White wins (black general missing).",
		}
	}

	sideToMove := CurrentTurnColor()
	legal, err := xiangqiAllLegalUCIMoves()
	if err != nil {
		return GameOutcome{Status: "in_progress", Message: "in progress"}
	}
	legalCount := len(legal)
	inCheck := movement.XiangqiCheckedColor() == sideToMove

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
		msg = "Stalemate! " + sideLabel(winner) + " wins (Xiangqi rule)."
	}
	return GameOutcome{
		Status:      "checkmate", // maps to a decisive win (not chess draw)
		Winner:      string(winner),
		Loser:       string(sideToMove),
		CheckedSide: string(sideToMove),
		LegalMoves:  0,
		Message:     msg,
	}
}

func evaluateOutcomeForGameType(gameType GameType) GameOutcome {
	switch gameType {
	case GameTypeXiangqi:
		return EvaluateXiangqiGameOutcome()
	case GameTypeShogi:
		// Terminal detection lands with strategies + legal list (later step).
		return GameOutcome{Status: "in_progress", Message: "in progress"}
	default:
		return EvaluateGameOutcome()
	}
}
