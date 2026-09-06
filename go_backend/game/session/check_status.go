// CM3070 FP code
// check_status.go - implements check status rules

package session

import (
	"go_backend/game/engine"
	pieces "go_backend/game/piece"
)

// game outcome for the game
type GameOutcome struct {
	Status      string `json:"status"`
	Winner      string `json:"winner,omitempty"`
	Loser       string `json:"loser,omitempty"`
	CheckedSide string `json:"checkedSide,omitempty"`
	LegalMoves  int    `json:"legalMoves"`
	Message     string `json:"message,omitempty"`
}

// CheckedSideColor - returns the color currently in check for the game
func CheckedSideColor() pieces.PieceColor {
	return engine.CheckedColor()
}

// CheckedSideLabel - returns "white", "black", or ""
func CheckedSideLabel() string {
	return string(CheckedSideColor())
}

// EvaluateGameOutcome - runs the Chess GameEndStrategy chain for the current ply
func EvaluateGameOutcome() GameOutcome {
	ensureChessGameEndStrategies()
	ctx := buildChessPlyEndContext()
	ended, out := runGameEndStrategies(ctx)
	if ended {
		return out
	}
	if ctx.InCheck {
		return GameOutcome{
			Status:      "check",
			CheckedSide: ctx.SideToMove,
			LegalMoves:  ctx.LegalMoves,
			Message:     sideLabelFromText(ctx.SideToMove) + " is in check.",
		}
	}
	return GameOutcome{
		Status:     "in_progress",
		LegalMoves: ctx.LegalMoves,
	}
}

// countLegalMoves - counts legal moves
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

// pieceLegalMoveCount - returns piece legal move count
func pieceLegalMoveCount(sourcePiece pieces.ChessPiece) int {
	return len(pieceLegalDestinations(sourcePiece))
}

// opponentOf - performs opponent of
func opponentOf(color pieces.PieceColor) pieces.PieceColor {
	if color == pieces.White {
		return pieces.Black
	}
	return pieces.White
}

// sideLabel - returns side label
func sideLabel(color pieces.PieceColor) string {
	if color == pieces.Black {
		return "Black"
	}
	return "White"
}
