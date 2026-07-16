package session

import (
	"fmt"

	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// xiangqiLegalDestinationsForSquare uses Go Xiangqi strategies (not Fairy-Stockfish).
func xiangqiLegalDestinationsForSquare(file, rank int) ([]LegalDestination, error) {
	sourcePiece, found := getPieceAt(file, rank)
	if !found {
		return []LegalDestination{}, nil
	}
	if sourcePiece.Color != CurrentTurnColor() {
		return []LegalDestination{}, nil
	}
	squares := movement.XiangqiLegalSquares(sourcePiece.Kind, sourcePiece.Color, file, rank)
	out := make([]LegalDestination, 0, len(squares))
	for _, sq := range squares {
		if movement.XiangqiWouldLeaveGeneralInCheck(sourcePiece, file, rank, sq.File, sq.Rank) {
			continue
		}
		_, isCapture := getPieceAt(sq.File, sq.Rank)
		out = append(out, LegalDestination{
			File:      sq.File,
			Rank:      sq.Rank,
			IsCapture: isCapture,
		})
	}
	return out, nil
}

// xiangqiAllLegalUCIMoves lists legal UCI moves for the side to move via Go rules.
func xiangqiAllLegalUCIMoves() ([]string, error) {
	side := CurrentTurnColor()
	out := make([]string, 0, 64)
	for _, p := range pieces.ChessPieces {
		if p.Color != side {
			continue
		}
		dests, err := xiangqiLegalDestinationsForSquare(p.File, p.Rank)
		if err != nil {
			return nil, err
		}
		for _, d := range dests {
			out = append(out, formatXiangqiUCI(p.File, p.Rank, d.File, d.Rank))
		}
	}
	return out, nil
}

func formatXiangqiUCI(fromFile, fromRank, toFile, toRank int) string {
	return fmt.Sprintf("%c%d%c%d", byte('a'+fromFile-1), fromRank, byte('a'+toFile-1), toRank)
}
