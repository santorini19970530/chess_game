// CM3070 FP code
// shogi_fen.go - shogi fen import and export for the session

package session

import (
	"fmt"
	"strings"
	"unicode"

	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// defaultShogiStartFEN is the standard Shogi start (Fairy-Stockfish compatible).
// hands sit in [] on the placement field; empty at start.
const DefaultShogiStartFEN = "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL[] w - - 0 1"

// shogiHands tracks captured pieces available to drop (relife inventory).
var shogiHands = shogiHandState{
	white: map[pieces.PieceKind]int{},
	black: map[pieces.PieceKind]int{},
}

type shogiHandState struct {
	white map[pieces.PieceKind]int
	black map[pieces.PieceKind]int
}

// resetShogiHands - resets shogi hands
func resetShogiHands() {
	shogiHands = shogiHandState{
		white: map[pieces.PieceKind]int{},
		black: map[pieces.PieceKind]int{},
	}
}

// applyShogiFENToCurrentGlobals - sets board, hands, side-to-move, and BoardFEN
func applyShogiFENToCurrentGlobals(fen string) error {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) < 2 {
		return fmt.Errorf("invalid shogi FEN: expected at least 2 fields")
	}
	placement, handText := splitShogiPlacementAndHands(parts[0])
	board, err := parseShogiFENBoard(placement)
	if err != nil {
		return err
	}
	hands, err := parseShogiHands(handText)
	if err != nil {
		return err
	}
	if err := validateShogiSetupBoard(board); err != nil {
		return err
	}
	if hands.white[pieces.King] > 0 || hands.black[pieces.King] > 0 {
		return fmt.Errorf("illegal shogi FEN: king in hand")
	}
	switch parts[1] {
	case "w":
		SetCurrentTurnColorPinned(pieces.White)
	case "b":
		SetCurrentTurnColorPinned(pieces.Black)
	default:
		return fmt.Errorf("invalid shogi FEN active color")
	}
	pieces.ChessPieces = board
	shogiHands = hands
	boardFEN = strings.TrimSpace(fen)
	lastAppliedMove = nil
	resetDrawTracking()
	recordShogiStartPosition()
	return nil
}

// splitShogiPlacementAndHands - splits shogi placement and hands
func splitShogiPlacementAndHands(raw string) (placement, hands string) {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexByte(raw, '['); i >= 0 {
		placement = raw[:i]
		end := strings.IndexByte(raw[i:], ']')
		if end >= 0 {
			hands = raw[i+1 : i+end]
		}
		return placement, hands
	}
	return raw, ""
}

// parseShogiFENBoard - parses shogi fen board
func parseShogiFENBoard(boardPart string) ([]pieces.ChessPiece, error) {
	ranks := strings.Split(boardPart, "/")
	if len(ranks) != 9 {
		return nil, fmt.Errorf("invalid shogi FEN board: expected 9 ranks")
	}
	out := make([]pieces.ChessPiece, 0, 40)
	for i, rankText := range ranks {
		rank := 9 - i
		file := 1
		for j := 0; j < len(rankText); {
			ch := rankText[j]
			if ch >= '1' && ch <= '9' {
				file += int(ch - '0')
				j++
				continue
			}
			promoted := false
			if ch == '+' {
				promoted = true
				j++
				if j >= len(rankText) {
					return nil, fmt.Errorf("invalid shogi FEN board: dangling +")
				}
				ch = rankText[j]
			}
			if file < 1 || file > 9 {
				return nil, fmt.Errorf("invalid shogi FEN board: file out of range")
			}
			kind, color, ok := shogiPieceFromChar(rune(ch), promoted)
			if !ok {
				return nil, fmt.Errorf("invalid shogi FEN piece %q", string(ch))
			}
			out = append(out, pieces.ChessPiece{
				Color: color,
				Kind:  kind,
				File:  file,
				Rank:  rank,
			})
			file++
			j++
		}
		if file != 10 {
			return nil, fmt.Errorf("invalid shogi FEN board: rank %d width", rank)
		}
	}
	return out, nil
}

// validateShogiSetupBoard - rejects a parsed shogi fen that is already an illegal setup
func validateShogiSetupBoard(board []pieces.ChessPiece) error {
	seen := map[[2]int]bool{}
	whitePawn, blackPawn := [10]int{}, [10]int{}
	wk, bk := 0, 0
	for _, p := range board {
		key := [2]int{p.File, p.Rank}
		if seen[key] {
			return fmt.Errorf("illegal shogi FEN: two pieces on %s", shogiSetupSquare(p.File, p.Rank))
		}
		seen[key] = true
		switch p.Kind {
		case pieces.King:
			if p.Color == pieces.White {
				wk++
			} else {
				bk++
			}
		case pieces.Pawn:
			if p.Color == pieces.White {
				whitePawn[p.File]++
			} else {
				blackPawn[p.File]++
			}
		}
		if movement.ShogiMustPromote(p.Kind, p.Rank, p.Color) {
			return fmt.Errorf(
				"illegal shogi FEN: %s %s on %s has no legal move",
				shogiSetupSide(p.Color), p.Kind, shogiSetupSquare(p.File, p.Rank),
			)
		}
	}
	if wk > 1 || bk > 1 {
		return fmt.Errorf("illegal shogi FEN: white has %d king(s), black has %d", wk, bk)
	}
	for file := 1; file <= 9; file++ {
		if whitePawn[file] > 1 {
			return fmt.Errorf("nifu: two unpromoted white pawns on file %s", shogiSetupFile(file))
		}
		if blackPawn[file] > 1 {
			return fmt.Errorf("nifu: two unpromoted black pawns on file %s", shogiSetupFile(file))
		}
	}
	return nil
}

// shogiSetupFile - uci file letter plus diagram file 1–9 (fen file 1 = a = 9)
func shogiSetupFile(file int) string {
	return fmt.Sprintf("%c (%d)", 'a'+file-1, 10-file)
}

// shogiSetupSquare - uci square plus diagram file and rank
func shogiSetupSquare(file, rank int) string {
	return fmt.Sprintf("%c%d (file %d rank %d)", 'a'+file-1, rank, 10-file, rank)
}

// shogiSetupSide - white/black word for setup errors
func shogiSetupSide(color pieces.PieceColor) string {
	if color == pieces.White {
		return "white"
	}
	return "black"
}

// shogiPieceFromChar - performs shogi piece from char
func shogiPieceFromChar(ch rune, promoted bool) (pieces.PieceKind, pieces.PieceColor, bool) {
	color := pieces.White
	if unicode.IsLower(ch) {
		color = pieces.Black
	}
	base := unicode.ToLower(ch)
	if promoted {
		switch base {
		case 'p':
			return pieces.PromotedPawn, color, true
		case 'l':
			return pieces.PromotedLance, color, true
		case 'n':
			return pieces.PromotedKnight, color, true
		case 's':
			return pieces.PromotedSilver, color, true
		case 'b':
			return pieces.Horse, color, true
		case 'r':
			return pieces.Dragon, color, true
		default:
			return "", "", false
		}
	}
	switch base {
	case 'p':
		return pieces.Pawn, color, true
	case 'l':
		return pieces.Lance, color, true
	case 'n':
		return pieces.Knight, color, true
	case 's':
		return pieces.Silver, color, true
	case 'g':
		return pieces.Gold, color, true
	case 'b':
		return pieces.Bishop, color, true
	case 'r':
		return pieces.Rook, color, true
	case 'k':
		return pieces.King, color, true
	default:
		return "", "", false
	}
}

// shogiCharFromPiece - performs shogi char from piece
func shogiCharFromPiece(kind pieces.PieceKind, color pieces.PieceColor) (string, bool) {
	var base byte
	promoted := false
	switch kind {
	case pieces.Pawn:
		base = 'p'
	case pieces.Lance:
		base = 'l'
	case pieces.Knight:
		base = 'n'
	case pieces.Silver:
		base = 's'
	case pieces.Gold:
		base = 'g'
	case pieces.Bishop:
		base = 'b'
	case pieces.Rook:
		base = 'r'
	case pieces.King:
		base = 'k'
	case pieces.PromotedPawn:
		base, promoted = 'p', true
	case pieces.PromotedLance:
		base, promoted = 'l', true
	case pieces.PromotedKnight:
		base, promoted = 'n', true
	case pieces.PromotedSilver:
		base, promoted = 's', true
	case pieces.Horse:
		base, promoted = 'b', true
	case pieces.Dragon:
		base, promoted = 'r', true
	default:
		return "", false
	}
	if color == pieces.White {
		base = byte(unicode.ToUpper(rune(base)))
	}
	if promoted {
		return "+" + string(base), true
	}
	return string(base), true
}

// parseShogiHands - parses shogi hands
func parseShogiHands(handText string) (shogiHandState, error) {
	out := shogiHandState{
		white: map[pieces.PieceKind]int{},
		black: map[pieces.PieceKind]int{},
	}
	for _, ch := range handText {
		if ch == ' ' {
			continue
		}
		kind, color, ok := shogiPieceFromChar(ch, false)
		if !ok {
			return out, fmt.Errorf("invalid shogi hand piece %q", string(ch))
		}
		if color == pieces.White {
			out.white[kind]++
		} else {
			out.black[kind]++
		}
	}
	return out, nil
}

// exportShogiHands - exports shogi hands
func exportShogiHands() string {
	order := []pieces.PieceKind{
		pieces.Rook, pieces.Bishop, pieces.Gold, pieces.Silver,
		pieces.Knight, pieces.Lance, pieces.Pawn,
	}
	var b strings.Builder
	for _, kind := range order {
		for i := 0; i < shogiHands.white[kind]; i++ {
			s, ok := shogiCharFromPiece(kind, pieces.White)
			if ok {
				b.WriteString(s)
			}
		}
	}
	for _, kind := range order {
		for i := 0; i < shogiHands.black[kind]; i++ {
			s, ok := shogiCharFromPiece(kind, pieces.Black)
			if ok {
				b.WriteString(s)
			}
		}
	}
	return b.String()
}

// exportShogiFEN - builds FEN from current pieces, hands, and side to move
func exportShogiFEN() string {
	type cell struct {
		kind  pieces.PieceKind
		color pieces.PieceColor
		ok    bool
	}
	grid := [10][10]cell{} // 1-indexed
	for _, p := range pieces.ChessPieces {
		if p.File < 1 || p.File > 9 || p.Rank < 1 || p.Rank > 9 {
			continue
		}
		grid[p.Rank][p.File] = cell{kind: p.Kind, color: p.Color, ok: true}
	}
	var placement strings.Builder
	for rank := 9; rank >= 1; rank-- {
		if rank < 9 {
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
			s, ok := shogiCharFromPiece(c.kind, c.color)
			if ok {
				placement.WriteString(s)
			}
		}
		if empty > 0 {
			placement.WriteByte(byte('0' + empty))
		}
	}
	placement.WriteByte('[')
	placement.WriteString(exportShogiHands())
	placement.WriteByte(']')
	active := "w"
	if CurrentTurnColor() == pieces.Black {
		active = "b"
	}
	fullmove := len(moveHistory)/2 + 1
	return fmt.Sprintf("%s %s - - 0 %d", placement.String(), active, fullmove)
}

// syncShogiBoardFEN - syncs shogi board fen
func syncShogiBoardFEN() {
	boardFEN = exportShogiFEN()
}

// looksLikeShogiFEN - shogi boards have 9 ranks (8 '/' separators in the placement field)
func looksLikeShogiFEN(fen string) bool {
	parts := strings.Fields(strings.TrimSpace(fen))
	if len(parts) == 0 {
		return false
	}
	placement, _ := splitShogiPlacementAndHands(parts[0])
	return strings.Count(placement, "/") == 8
}
