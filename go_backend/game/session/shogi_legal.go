package session

import (
	"fmt"

	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

func shogiLegalDestinationsForSquare(file, rank int) ([]LegalDestination, error) {
	sourcePiece, found := getPieceAt(file, rank)
	if !found {
		return []LegalDestination{}, nil
	}
	if sourcePiece.Color != CurrentTurnColor() {
		return []LegalDestination{}, nil
	}
	squares := movement.ShogiLegalSquares(sourcePiece.Kind, sourcePiece.Color, file, rank)
	out := make([]LegalDestination, 0, len(squares))
	for _, sq := range squares {
		if movement.ShogiWouldLeaveKingInCheck(sourcePiece, file, rank, sq.File, sq.Rank) {
			continue
		}
		_, isCapture := getPieceAt(sq.File, sq.Rank)
		must := movement.ShogiMustPromote(sourcePiece.Kind, sq.Rank, sourcePiece.Color)
		can := movement.ShogiCanPromote(sourcePiece.Kind, rank, sq.Rank, sourcePiece.Color)
		out = append(out, LegalDestination{
			File:              sq.File,
			Rank:              sq.Rank,
			IsCapture:         isCapture,
			RequiresPromotion: must,
			CanPromote:        can,
		})
	}
	return out, nil
}

func shogiAllLegalUCIMoves() ([]string, error) {
	side := CurrentTurnColor()
	out := make([]string, 0, 128)
	for _, p := range pieces.ChessPieces {
		if p.Color != side {
			continue
		}
		dests, err := shogiLegalDestinationsForSquare(p.File, p.Rank)
		if err != nil {
			return nil, err
		}
		for _, d := range dests {
			core := formatShogiBoardUCI(p.File, p.Rank, d.File, d.Rank)
			can := movement.ShogiCanPromote(p.Kind, p.Rank, d.Rank, p.Color)
			must := d.RequiresPromotion
			if must {
				out = append(out, core+"+")
				continue
			}
			out = append(out, core)
			if can {
				out = append(out, core+"+")
			}
		}
	}
	out = append(out, shogiAllLegalDrops(side)...)
	return out, nil
}

func shogiAllLegalDrops(side pieces.PieceColor) []string {
	hand := shogiHandMap(side)
	if len(hand) == 0 {
		return nil
	}
	out := make([]string, 0, 64)
	for kind, n := range hand {
		if n <= 0 {
			continue
		}
		ch, ok := shogiDropChar(kind)
		if !ok {
			continue
		}
		for _, d := range shogiLegalDropDestinations(kind, side) {
			dropCh := ch
			if dropCh >= 'A' && dropCh <= 'Z' {
				dropCh += 'a' - 'A'
			}
			// Lowercase so handlers.normalizeUCI matches engine candidates.
			out = append(out, fmt.Sprintf("%c*%c%d", dropCh, byte('a'+d.File-1), d.Rank))
		}
	}
	return out
}

func shogiLegalDropDestinations(kind pieces.PieceKind, side pieces.PieceColor) []LegalDestination {
	out := make([]LegalDestination, 0, 32)
	for file := 1; file <= 9; file++ {
		for rank := 1; rank <= 9; rank++ {
			if err := validateShogiDrop(kind, side, file, rank); err != nil {
				continue
			}
			out = append(out, LegalDestination{File: file, Rank: rank})
		}
	}
	return out
}

func formatShogiBoardUCI(fromFile, fromRank, toFile, toRank int) string {
	return fmt.Sprintf("%c%d%c%d", byte('a'+fromFile-1), fromRank, byte('a'+toFile-1), toRank)
}
