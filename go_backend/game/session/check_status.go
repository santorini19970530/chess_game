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
	total := 0
	for toFile := 1; toFile <= 8; toFile++ {
		for toRank := 1; toRank <= 8; toRank++ {
			if sourcePiece.File == toFile && sourcePiece.Rank == toRank {
				continue
			}

			enPassant := false
			castling := false

			_, err := engine.ValidateMove(sourcePiece.File, sourcePiece.Rank, toFile, toRank, "")
			if err != nil {
				kingSide := toFile == 7
				queenSide := toFile == 3
				if sourcePiece.Kind == pieces.King && (kingSide || queenSide) && engine.CanCastle(
					sourcePiece,
					sourcePiece.File, sourcePiece.Rank,
					toFile, toRank,
					CanCastleByState(sourcePiece.Color, kingSide),
				) {
					if castlingViolatesCheckRules(sourcePiece.Color, sourcePiece.Rank, toFile) {
						continue
					}
					castling = true
				} else {
					_, destinationOccupied := getPieceAt(toFile, toRank)
					adjacentPawn, adjacentPawnFound := getPieceAt(toFile, sourcePiece.Rank)
					lastMove := toEngineLastMove(GetLastMove())
					if sourcePiece.Kind == pieces.Pawn && engine.CanEnPassant(
						sourcePiece,
						sourcePiece.File, sourcePiece.Rank,
						toFile, toRank,
						destinationOccupied,
						lastMove,
						adjacentPawn,
						adjacentPawnFound,
					) {
						enPassant = true
					} else {
						continue
					}
				}
			}

			requiresPromotion := sourcePiece.Kind == pieces.Pawn && ((sourcePiece.Color == pieces.White && toRank == 8) || (sourcePiece.Color == pieces.Black && toRank == 1))
			promotionKind := pieces.Queen

			if engine.WouldLeaveKingInCheck(
				sourcePiece,
				sourcePiece.File, sourcePiece.Rank,
				toFile, toRank,
				enPassant, castling,
				requiresPromotion, promotionKind,
			) {
				continue
			}
			total++
		}
	}
	return total
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
