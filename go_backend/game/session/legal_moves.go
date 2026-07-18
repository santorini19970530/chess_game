package session

import (
	"go_backend/game/engine"
	pieces "go_backend/game/piece"
)

type LegalDestination struct {
	File              int  `json:"file"`
	Rank              int  `json:"rank"`
	RequiresPromotion bool `json:"requiresPromotion"`
	CanPromote        bool `json:"canPromote,omitempty"` // shogi optional zone (must also sets requiresPromotion)
	IsCapture         bool `json:"isCapture"`
}

// LegalMovesForSquare returns legal destinations for a source square on current turn.
func LegalMovesForSquare(file, rank int) []LegalDestination {
	sourcePiece, found := getPieceAt(file, rank)
	if !found {
		return []LegalDestination{}
	}
	if sourcePiece.Color != CurrentTurnColor() {
		return []LegalDestination{}
	}
	return pieceLegalDestinations(sourcePiece)
}

func pieceLegalDestinations(sourcePiece pieces.ChessPiece) []LegalDestination {
	out := make([]LegalDestination, 0, 32)
	for toFile := 1; toFile <= 8; toFile++ {
		for toRank := 1; toRank <= 8; toRank++ {
			if sourcePiece.File == toFile && sourcePiece.Rank == toRank {
				continue
			}

			enPassant := false
			castling := false
			isCapture := false

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
					isCapture = destinationOccupied
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
						isCapture = true
					} else {
						continue
					}
				}
			} else {
				_, destinationOccupied := getPieceAt(toFile, toRank)
				isCapture = destinationOccupied
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
			out = append(out, LegalDestination{
				File:              toFile,
				Rank:              toRank,
				RequiresPromotion: requiresPromotion,
				IsCapture:         isCapture,
			})
		}
	}
	return out
}
