// CM3070 FP code
// history.go - records the movement history and the current state of the board

package session

import (
	"fmt"
	"strings"

	pieces "go_backend/game/piece"
)

// temporary storage for the movement history
var moveHistory []string
var moveHistoryDetailed []MoveHistoryEntry
var currentTurnOverride *pieces.PieceColor
var currentTurnPinned bool
var halfmoveClock int
var positionCounts map[string]int

type LastMove struct {
	FromFile       int
	FromRank       int
	ToFile         int
	ToRank         int
	PieceKind      pieces.PieceKind
	Color          pieces.PieceColor
	PawnDoubleStep bool
}

type MoveHistoryEntry struct {
	Side              string `json:"side"`
	PieceKind         string `json:"pieceKind"`
	To                string `json:"to"`
	Command           string `json:"command"`
	IsCapture         bool   `json:"isCapture"`
	CapturedPieceKind string `json:"capturedPieceKind,omitempty"`
}

var lastAppliedMove *LastMove

// boardFEN is set for Xiangqi (and later Shogi); when non-empty, CurrentFEN returns it.
var boardFEN string

var whiteKingMoved bool
var blackKingMoved bool
var whiteRookAMoved bool
var whiteRookHMoved bool
var blackRookAMoved bool
var blackRookHMoved bool

// append the movement command to the history string
func AppendMoveHistory(command string, color pieces.PieceColor, pieceKind pieces.PieceKind, toFile, toRank int, isCapture bool, capturedKind pieces.PieceKind) {
	sideLabel := "White"
	if color == pieces.Black {
		sideLabel = "Black"
	}
	capturedKindText := ""
	if isCapture {
		capturedKindText = string(capturedKind)
	}
	moveHistory = append(moveHistory, fmt.Sprintf("%s: %s", sideLabel, command))
	moveHistoryDetailed = append(moveHistoryDetailed, MoveHistoryEntry{
		Side:              sideLabel,
		PieceKind:         string(pieceKind),
		To:                fmt.Sprintf("%c%d", byte('a'+toFile-1), toRank),
		Command:           command,
		IsCapture:         isCapture,
		CapturedPieceKind: capturedKindText,
	})
}

// get the movement history
func GetMoveHistory() []string {
	out := make([]string, len(moveHistory))
	copy(out, moveHistory)

	return out
}

func GetMoveHistoryDetailed() []MoveHistoryEntry {
	if len(moveHistory) == 0 {
		return nil
	}
	out := make([]MoveHistoryEntry, len(moveHistoryDetailed))
	copy(out, moveHistoryDetailed)
	return out
}

// CurrentTurnColor returns whose move should be next.
// White starts at move 1, then alternates each successful move.
func CurrentTurnColor() pieces.PieceColor {
	if currentTurnOverride != nil {
		if !currentTurnPinned && len(moveHistory) == 0 && lastAppliedMove == nil {
			// If tests or manual setup clear runtime history, fall back to standard start turn.
			currentTurnOverride = nil
		}
	}
	if currentTurnOverride != nil {
		return *currentTurnOverride
	}
	if len(moveHistory)%2 == 0 {
		return pieces.White
	}
	return pieces.Black
}

// CurrentTurnLabel returns a frontend-friendly turn label.
func CurrentTurnLabel() string {
	if CurrentTurnColor() == pieces.Black {
		return "Black"
	}
	return "White"
}

func RecordLastMove(fromFile, fromRank, toFile, toRank int, kind pieces.PieceKind, color pieces.PieceColor) {
	lastAppliedMove = &LastMove{
		FromFile:       fromFile,
		FromRank:       fromRank,
		ToFile:         toFile,
		ToRank:         toRank,
		PieceKind:      kind,
		Color:          color,
		PawnDoubleStep: kind == pieces.Pawn && fromFile == toFile && absInt(toRank-fromRank) == 2,
	}
}

func GetLastMove() *LastMove {
	if lastAppliedMove == nil {
		return nil
	}
	copied := *lastAppliedMove
	return &copied
}

func RecordPieceMoveForCastling(kind pieces.PieceKind, color pieces.PieceColor, fromFile, fromRank int) {
	switch kind {
	case pieces.King:
		if color == pieces.White {
			whiteKingMoved = true
		} else if color == pieces.Black {
			blackKingMoved = true
		}
	case pieces.Rook:
		if color == pieces.White && fromRank == 1 {
			if fromFile == 1 {
				whiteRookAMoved = true
			}
			if fromFile == 8 {
				whiteRookHMoved = true
			}
		}
		if color == pieces.Black && fromRank == 8 {
			if fromFile == 1 {
				blackRookAMoved = true
			}
			if fromFile == 8 {
				blackRookHMoved = true
			}
		}
	}
}

func CanCastleByState(color pieces.PieceColor, kingSide bool) bool {
	if color == pieces.White {
		if kingSide {
			return !whiteKingMoved && !whiteRookHMoved
		}
		return !whiteKingMoved && !whiteRookAMoved
	}
	if kingSide {
		return !blackKingMoved && !blackRookHMoved
	}
	return !blackKingMoved && !blackRookAMoved
}

func resetCastlingState() {
	whiteKingMoved = false
	blackKingMoved = false
	whiteRookAMoved = false
	whiteRookHMoved = false
	blackRookAMoved = false
	blackRookHMoved = false
}

func SetCurrentTurnColor(color pieces.PieceColor) {
	c := color
	currentTurnOverride = &c
	currentTurnPinned = false
}

func SetCurrentTurnColorPinned(color pieces.PieceColor) {
	c := color
	currentTurnOverride = &c
	currentTurnPinned = true
}

func AdvanceTurnColor() {
	SetCurrentTurnColor(OpponentColor(CurrentTurnColor()))
}

func resetTurnOverride() {
	currentTurnOverride = nil
	currentTurnPinned = false
}

func SetCastlingStateFromFEN(rights string) {
	resetCastlingState()
	whiteKingMoved = true
	blackKingMoved = true
	whiteRookAMoved = true
	whiteRookHMoved = true
	blackRookAMoved = true
	blackRookHMoved = true

	for _, ch := range rights {
		switch ch {
		case 'K':
			whiteKingMoved = false
			whiteRookHMoved = false
		case 'Q':
			whiteKingMoved = false
			whiteRookAMoved = false
		case 'k':
			blackKingMoved = false
			blackRookHMoved = false
		case 'q':
			blackKingMoved = false
			blackRookAMoved = false
		}
	}
}

func OpponentColor(color pieces.PieceColor) pieces.PieceColor {
	if color == pieces.White {
		return pieces.Black
	}
	return pieces.White
}

func resetDrawTracking() {
	halfmoveClock = 0
	positionCounts = make(map[string]int)
	recordCurrentPosition()
}

func recordDrawStateAfterMove(movedKind pieces.PieceKind, capture bool) {
	if movedKind == pieces.Pawn || capture {
		halfmoveClock = 0
	} else {
		halfmoveClock++
	}
	recordCurrentPosition()
}

func GetHalfmoveClock() int {
	return halfmoveClock
}

func GetCurrentPositionRepetitionCount() int {
	key := currentPositionKey()
	return positionCounts[key]
}

func currentPositionKey() string {
	return fmt.Sprintf("%s %s %s %s",
		boardToKey(),
		string(CurrentTurnColor()),
		castlingRightsKey(),
		enPassantTargetKey(),
	)
}

func recordCurrentPosition() {
	if positionCounts == nil {
		positionCounts = make(map[string]int)
	}
	key := currentPositionKey()
	positionCounts[key]++
}

func boardToKey() string {
	var out strings.Builder
	for rank := 8; rank >= 1; rank-- {
		empty := 0
		for file := 1; file <= 8; file++ {
			p, found := getPieceAt(file, rank)
			if !found {
				empty++
				continue
			}
			if empty > 0 {
				out.WriteString(fmt.Sprintf("%d", empty))
				empty = 0
			}
			out.WriteRune(pieceToFENRune(p))
		}
		if empty > 0 {
			out.WriteString(fmt.Sprintf("%d", empty))
		}
		if rank > 1 {
			out.WriteRune('/')
		}
	}
	return out.String()
}

func pieceToFENRune(p pieces.ChessPiece) rune {
	var ch rune
	switch p.Kind {
	case pieces.Pawn:
		ch = 'p'
	case pieces.Rook:
		ch = 'r'
	case pieces.Knight:
		ch = 'n'
	case pieces.Bishop:
		ch = 'b'
	case pieces.Queen:
		ch = 'q'
	case pieces.King:
		ch = 'k'
	default:
		ch = 'x'
	}
	if p.Color == pieces.White {
		return ch - ('a' - 'A')
	}
	return ch
}

func castlingRightsKey() string {
	rights := ""
	if CanCastleByState(pieces.White, true) {
		rights += "K"
	}
	if CanCastleByState(pieces.White, false) {
		rights += "Q"
	}
	if CanCastleByState(pieces.Black, true) {
		rights += "k"
	}
	if CanCastleByState(pieces.Black, false) {
		rights += "q"
	}
	if rights == "" {
		return "-"
	}
	return rights
}

func enPassantTargetKey() string {
	mv := GetLastMove()
	if mv == nil || !mv.PawnDoubleStep {
		return "-"
	}
	targetRank := (mv.FromRank + mv.ToRank) / 2
	fileChar := byte('a' + mv.ToFile - 1)
	return fmt.Sprintf("%c%d", fileChar, targetRank)
}

// temporary storage for the current state of the board
type PieceState struct {
	Color string `json:"color"`
	Kind  string `json:"kind"`
	File  int    `json:"file"`
	Rank  int    `json:"rank"`
}

type CapturedSummary struct {
	White map[string]int `json:"white"`
	Black map[string]int `json:"black"`
}

// get the current state of the board
func GetBoardState() []PieceState {
	state := make([]PieceState, 0, len(pieces.ChessPieces))
	for _, p := range pieces.ChessPieces {
		state = append(state, PieceState{
			Color: string(p.Color),
			Kind:  string(p.Kind),
			File:  p.File,
			Rank:  p.Rank,
		})
	}

	return state
}

// GetCapturedSummary computes captured-piece counts for each side
// from current board state against standard Chess initial piece counts.
func GetCapturedSummary() CapturedSummary {
	return capturedSummaryAgainstInitial(map[string]int{
		"pawn": 8, "rook": 2, "knight": 2, "bishop": 2, "queen": 1, "king": 1,
	})
}

// GetXiangqiCapturedSummary uses Xiangqi starting counts (not Chess queens/bishops).
func GetXiangqiCapturedSummary() CapturedSummary {
	return capturedSummaryAgainstInitial(map[string]int{
		"pawn": 5, "rook": 2, "knight": 2, "elephant": 2, "advisor": 2, "cannon": 2, "king": 1,
	})
}

func capturedSummaryAgainstInitial(initial map[string]int) CapturedSummary {
	liveWhite := map[string]int{}
	liveBlack := map[string]int{}
	for kind := range initial {
		liveWhite[kind] = 0
		liveBlack[kind] = 0
	}
	for _, p := range pieces.ChessPieces {
		kind := string(p.Kind)
		if _, ok := initial[kind]; !ok {
			continue
		}
		if p.Color == pieces.White {
			liveWhite[kind]++
		} else if p.Color == pieces.Black {
			liveBlack[kind]++
		}
	}

	whiteCaptured := map[string]int{}
	blackCaptured := map[string]int{}
	for kind, total := range initial {
		w := total - liveBlack[kind] // white captures black
		b := total - liveWhite[kind] // black captures white
		if w < 0 {
			w = 0
		}
		if b < 0 {
			b = 0
		}
		whiteCaptured[kind] = w
		blackCaptured[kind] = b
	}

	return CapturedSummary{
		White: whiteCaptured,
		Black: blackCaptured,
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
