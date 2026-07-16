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

func xiangqiCharFromPiece(kind pieces.PieceKind, color pieces.PieceColor) (byte, bool) {
	var ch byte
	switch kind {
	case pieces.Rook:
		ch = 'r'
	case pieces.Knight:
		ch = 'n'
	case pieces.Elephant:
		ch = 'b'
	case pieces.Advisor:
		ch = 'a'
	case pieces.King:
		ch = 'k'
	case pieces.Cannon:
		ch = 'c'
	case pieces.Pawn:
		ch = 'p'
	default:
		return 0, false
	}
	if color == pieces.White {
		ch = byte(unicode.ToUpper(rune(ch)))
	}
	return ch, true
}

// exportXiangqiFEN builds FEN from current pieces + side to move.
func exportXiangqiFEN() string {
	type cell struct {
		kind  pieces.PieceKind
		color pieces.PieceColor
		ok    bool
	}
	grid := [11][10]cell{} // 1-indexed ranks/files
	for _, p := range pieces.ChessPieces {
		if p.File < 1 || p.File > 9 || p.Rank < 1 || p.Rank > 10 {
			continue
		}
		grid[p.Rank][p.File] = cell{kind: p.Kind, color: p.Color, ok: true}
	}
	var placement strings.Builder
	for rank := 10; rank >= 1; rank-- {
		if rank < 10 {
			placement.WriteByte('/')
		}
		empty := 0
		for file := 1; file <= 9; file++ {
			c := grid[rank][file]
			if !c.ok {
				empty++
				continue
			}
			if empty > 0 {
				placement.WriteByte(byte('0' + empty))
				empty = 0
			}
			ch, ok := xiangqiCharFromPiece(c.kind, c.color)
			if ok {
				placement.WriteByte(ch)
			}
		}
		if empty > 0 {
			placement.WriteByte(byte('0' + empty))
		}
	}
	active := "w"
	if CurrentTurnColor() == pieces.Black {
		active = "b"
	}
	fullmove := len(moveHistory)/2 + 1
	return fmt.Sprintf("%s %s - - 0 %d", placement.String(), active, fullmove)
}

func syncXiangqiBoardFEN() {
	boardFEN = exportXiangqiFEN()
}
