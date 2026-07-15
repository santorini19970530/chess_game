package session

import (
	"fmt"
	"strings"
	"unicode"

	pieces "go_backend/game/piece"
)

// applyXiangqiFENToCurrentGlobals sets BoardFEN, piece list, and side-to-move from a Xiangqi FEN.
func applyXiangqiFENToCurrentGlobals(fen string) error {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) < 2 {
		return fmt.Errorf("invalid xiangqi FEN: expected at least 2 fields")
	}
	board, err := parseXiangqiFENBoard(parts[0])
	if err != nil {
		return err
	}
	switch parts[1] {
	case "w":
		SetCurrentTurnColorPinned(pieces.White)
	case "b":
		SetCurrentTurnColorPinned(pieces.Black)
	default:
		return fmt.Errorf("invalid xiangqi FEN active color")
	}
	pieces.ChessPieces = board
	boardFEN = strings.TrimSpace(fen)
	lastAppliedMove = nil
	resetDrawTracking()
	return nil
}

func parseXiangqiFENBoard(boardPart string) ([]pieces.ChessPiece, error) {
	ranks := strings.Split(boardPart, "/")
	if len(ranks) != 10 {
		return nil, fmt.Errorf("invalid xiangqi FEN board: expected 10 ranks")
	}
	out := make([]pieces.ChessPiece, 0, 32)
	for i, rankText := range ranks {
		rank := 10 - i
		file := 1
		for _, ch := range rankText {
			if ch >= '1' && ch <= '9' {
				file += int(ch - '0')
				continue
			}
			if file < 1 || file > 9 {
				return nil, fmt.Errorf("invalid xiangqi FEN board: file out of range")
			}
			kind, color, ok := xiangqiPieceFromChar(ch)
			if !ok {
				return nil, fmt.Errorf("invalid xiangqi FEN piece %q", string(ch))
			}
			out = append(out, pieces.ChessPiece{
				Color: color,
				Kind:  kind,
				File:  file,
				Rank:  rank,
			})
			file++
		}
		if file != 10 {
			return nil, fmt.Errorf("invalid xiangqi FEN board: rank %d width", rank)
		}
	}
	return out, nil
}

func xiangqiPieceFromChar(ch rune) (pieces.PieceKind, pieces.PieceColor, bool) {
	color := pieces.White
	if unicode.IsLower(ch) {
		color = pieces.Black
	}
	switch unicode.ToLower(ch) {
	case 'r':
		return pieces.Rook, color, true
	case 'n':
		return pieces.Knight, color, true
	case 'b', 'e':
		return pieces.Elephant, color, true
	case 'a':
		return pieces.Advisor, color, true
	case 'k':
		return pieces.King, color, true
	case 'c':
		return pieces.Cannon, color, true
	case 'p':
		return pieces.Pawn, color, true
	default:
		return "", "", false
	}
}
