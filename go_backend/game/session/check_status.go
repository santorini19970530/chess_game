// CM3070 FP code
// check_status.go - implements check status rules

package session

import (
	"go_backend/game/engine"
	pieces "go_backend/game/piece"
)

type GameOutcome struct {
	Status      string `json:"status"`
	Winner      string `json:"winner,omitempty"`
	Loser       string `json:"loser,omitempty"`
	CheckedSide string `json:"checkedSide,omitempty"`
	LegalMoves  int    `json:"legalMoves"`
	Message     string `json:"message,omitempty"`
}

// CheckedSideColor returns the color currently in check.
func CheckedSideColor() pieces.PieceColor {
	return engine.CheckedColor()
}

// CheckedSideLabel returns "white", "black", or "".
func CheckedSideLabel() string {
	return string(CheckedSideColor())
}

func EvaluateGameOutcome() GameOutcome {
	sideToMove := CurrentTurnColor()
	inCheck := engine.IsInCheck(sideToMove)
	legalMoves := countLegalMoves(sideToMove)
	hasLegalMove := legalMoves > 0

	if hasLegalMove {
		if isInsufficientMaterialDraw() {
			return GameOutcome{
				Status:     "draw_insufficient_material",
				LegalMoves: legalMoves,
				Message:    "Draw by insufficient material.",
			}
		}
		if isThreefoldRepetitionDraw() {
			return GameOutcome{
				Status:     "draw_threefold_repetition",
				LegalMoves: legalMoves,
				Message:    "Draw by threefold repetition.",
			}
		}
		if isFiftyMoveDraw() {
			return GameOutcome{
				Status:     "draw_fifty_move_rule",
				LegalMoves: legalMoves,
				Message:    "Draw by 50-move rule.",
			}
		}
	}

	if hasLegalMove {
		if inCheck {
			return GameOutcome{
				Status:      "check",
				CheckedSide: string(sideToMove),
				LegalMoves:  legalMoves,
				Message:     sideLabel(sideToMove) + " is in check.",
			}
		}
		return GameOutcome{
			Status:     "in_progress",
			LegalMoves: legalMoves,
		}
	}

	if inCheck {
		winner := opponentOf(sideToMove)
		return GameOutcome{
			Status:      "checkmate",
			Winner:      string(winner),
			Loser:       string(sideToMove),
			CheckedSide: string(sideToMove),
			LegalMoves:  0,
			Message:     "Checkmate! " + sideLabel(winner) + " wins.",
		}
	}

	return GameOutcome{
		Status:     "stalemate",
		LegalMoves: 0,
		Message:    "Draw by stalemate.",
	}
}

func countLegalMoves(color pieces.PieceColor) int {
	total := 0
	for _, p := range pieces.ChessPieces {
		if p.Color != color {
			continue
		}
		total += pieceLegalMoveCount(p)
	}
	return total
}

func pieceLegalMoveCount(sourcePiece pieces.ChessPiece) int {
	return len(pieceLegalDestinations(sourcePiece))
}

func opponentOf(color pieces.PieceColor) pieces.PieceColor {
	if color == pieces.White {
		return pieces.Black
	}
	return pieces.White
}

func sideLabel(color pieces.PieceColor) string {
	if color == pieces.Black {
		return "Black"
	}
	return "White"
}
