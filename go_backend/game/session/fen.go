// CM3070 FP code
// fen.go - implements FEN parsing rules

package session

import (
	"fmt"
	"strings"

	pieces "go_backend/game/piece"
)

// ApplyFEN sets board state, side to move, and castling rights from a FEN string.
func ApplyFEN(fen string) error {
	game, err := lockActiveRuntimeState()
	if err != nil {
		return err
	}
	defer unlockActiveRuntimeState(game)
	return applyFENToCurrentGlobals(fen)
}

func applyFENToCurrentGlobals(fen string) error {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) < 4 {
		return fmt.Errorf("invalid FEN: expected at least 4 fields")
	}

	boardPart := parts[0]
	activeColor := parts[1]
	castlingRights := parts[2]

	board, err := parseFENBoard(boardPart)
	if err != nil {
		return err
	}
	pieces.ChessPieces = board

	switch activeColor {
	case "w":
		SetCurrentTurnColorPinned(pieces.White)
	case "b":
		SetCurrentTurnColorPinned(pieces.Black)
	default:
		return fmt.Errorf("invalid FEN active color")
	}

	if castlingRights == "-" {
		SetCastlingStateFromFEN("")
	} else {
		SetCastlingStateFromFEN(castlingRights)
	}
	lastAppliedMove = nil
	resetDrawTracking()
	if len(parts) >= 5 {
		halfmove, err := parseInt(parts[4])
		if err != nil {
			return fmt.Errorf("invalid FEN halfmove clock")
		}
		halfmoveClock = halfmove
	}
	return nil
}

func parseFENBoard(boardPart string) ([]pieces.ChessPiece, error) {
	ranks := strings.Split(boardPart, "/")
	if len(ranks) != 8 {
		return nil, fmt.Errorf("invalid FEN board: expected 8 ranks")
	}
	out := make([]pieces.ChessPiece, 0, 32)
	for i, rankText := range ranks {
		rank := 8 - i
		file := 1
		for _, ch := range rankText {
			if ch >= '1' && ch <= '8' {
				file += int(ch - '0')
				continue
			}
			if file < 1 || file > 8 {
				return nil, fmt.Errorf("invalid FEN board: file out of range")
			}
			piece, err := fenCharToPiece(ch, file, rank)
			if err != nil {
				return nil, err
			}
			out = append(out, piece)
			file++
		}
		if file != 9 {
			return nil, fmt.Errorf("invalid FEN board: rank does not sum to 8")
		}
	}
	return out, nil
}

func fenCharToPiece(ch rune, file, rank int) (pieces.ChessPiece, error) {
	color := pieces.White
	if ch >= 'a' && ch <= 'z' {
		color = pieces.Black
	}

	lower := ch
	if ch >= 'A' && ch <= 'Z' {
		lower = ch - 'A' + 'a'
	}

	var kind pieces.PieceKind
	switch lower {
	case 'p':
		kind = pieces.Pawn
	case 'r':
		kind = pieces.Rook
	case 'n':
		kind = pieces.Knight
	case 'b':
		kind = pieces.Bishop
	case 'q':
		kind = pieces.Queen
	case 'k':
		kind = pieces.King
	default:
		return pieces.ChessPiece{}, fmt.Errorf("invalid FEN piece: %q", string(ch))
	}

	tone := "light"
	if color == pieces.Black {
		tone = "dark"
	}

	return pieces.ChessPiece{
		Color:   color,
		Kind:    kind,
		ImgFile: fmt.Sprintf("pic/chess_pic/%s_%s.png", string(kind), tone),
		File:    file,
		Rank:    rank,
	}, nil
}

func parseInt(v string) (int, error) {
	n := 0
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid integer")
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}
