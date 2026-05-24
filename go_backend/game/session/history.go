// CM3070 FP code
// history.go - records the movement history and the current state of the board

package session

import (
	"fmt"

	pieces "go_backend/game/piece"
)

// temporary storage for the movement history
var moveHistory []string

type LastMove struct {
	FromFile       int
	FromRank       int
	ToFile         int
	ToRank         int
	PieceKind      pieces.PieceKind
	Color          pieces.PieceColor
	PawnDoubleStep bool
}

var lastAppliedMove *LastMove

// append the movement command to the history string
func AppendMoveHistory(command string, color pieces.PieceColor) {
	sideLabel := "White"
	if color == pieces.Black {
		sideLabel = "Black"
	}
	moveHistory = append(moveHistory, fmt.Sprintf("%s: %s", sideLabel, command))
}

// get the movement history
func GetMoveHistory() []string {
	out := make([]string, len(moveHistory))
	copy(out, moveHistory)

	return out
}

// CurrentTurnColor returns whose move should be next.
// White starts at move 1, then alternates each successful move.
func CurrentTurnColor() pieces.PieceColor {
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

// temporary storage for the current state of the board
type PieceState struct {
	Color   string `json:"color"`
	Kind    string `json:"kind"`
	ImgFile string `json:"imgFile"`
	File    int    `json:"file"`
	Rank    int    `json:"rank"`
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
			Color:   string(p.Color),
			Kind:    string(p.Kind),
			ImgFile: p.ImgFile,
			File:    p.File,
			Rank:    p.Rank,
		})
	}

	return state
}

// GetCapturedSummary computes captured-piece counts for each side
// from current board state against standard initial piece counts.
func GetCapturedSummary() CapturedSummary {
	initial := map[string]int{
		"pawn":   8,
		"rook":   2,
		"knight": 2,
		"bishop": 2,
		"queen":  1,
		"king":   1,
	}
	liveWhite := map[string]int{
		"pawn": 0, "rook": 0, "knight": 0, "bishop": 0, "queen": 0, "king": 0,
	}
	liveBlack := map[string]int{
		"pawn": 0, "rook": 0, "knight": 0, "bishop": 0, "queen": 0, "king": 0,
	}
	for _, p := range pieces.ChessPieces {
		kind := string(p.Kind)
		if p.Color == pieces.White {
			liveWhite[kind]++
		} else if p.Color == pieces.Black {
			liveBlack[kind]++
		}
	}

	whiteCaptured := map[string]int{
		"pawn": 0, "rook": 0, "knight": 0, "bishop": 0, "queen": 0, "king": 0,
	}
	blackCaptured := map[string]int{
		"pawn": 0, "rook": 0, "knight": 0, "bishop": 0, "queen": 0, "king": 0,
	}
	for kind, total := range initial {
		whiteCaptured[kind] = total - liveBlack[kind] // white captures black
		blackCaptured[kind] = total - liveWhite[kind] // black captures white
		if whiteCaptured[kind] < 0 {
			whiteCaptured[kind] = 0
		}
		if blackCaptured[kind] < 0 {
			blackCaptured[kind] = 0
		}
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
