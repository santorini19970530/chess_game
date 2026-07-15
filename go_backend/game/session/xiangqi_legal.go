package session

import (
	"fmt"

	"go_backend/game/engine"
)

// xiangqiAllLegalUCIMoves asks Fairy-Stockfish for every legal move in the current BoardFEN.
func xiangqiAllLegalUCIMoves() ([]string, error) {
	fen := boardFEN
	if fen == "" {
		fen = DefaultXiangqiStartFEN
	}
	fs, err := engine.RulesEngine()
	if err != nil {
		return nil, fmt.Errorf("fairy-stockfish unavailable: %w", err)
	}
	if err := fs.SetVariant(string(GameTypeXiangqi)); err != nil {
		return nil, fmt.Errorf("set xiangqi variant: %w", err)
	}
	return fs.LegalMoves(fen)
}

// xiangqiLegalDestinationsForSquare filters FS legal moves that start on (file, rank).
func xiangqiLegalDestinationsForSquare(file, rank int) ([]LegalDestination, error) {
	sourcePiece, found := getPieceAt(file, rank)
	if !found {
		return []LegalDestination{}, nil
	}
	if sourcePiece.Color != CurrentTurnColor() {
		return []LegalDestination{}, nil
	}
	all, err := xiangqiAllLegalUCIMoves()
	if err != nil {
		return nil, err
	}
	out := make([]LegalDestination, 0, 8)
	for _, mv := range all {
		fromFile, fromRank, toFile, toRank, err := parseXiangqiUCISquares(mv)
		if err != nil {
			continue
		}
		if fromFile != file || fromRank != rank {
			continue
		}
		_, isCapture := getPieceAt(toFile, toRank)
		out = append(out, LegalDestination{
			File:      toFile,
			Rank:      toRank,
			IsCapture: isCapture,
		})
	}
	return out, nil
}

// formatXiangqiUCI builds an FS UCI string (ranks 1–10, no zero-padding).
func formatXiangqiUCI(fromFile, fromRank, toFile, toRank int) string {
	return fmt.Sprintf("%c%d%c%d", byte('a'+fromFile-1), fromRank, byte('a'+toFile-1), toRank)
}
