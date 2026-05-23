// CM3070 FP code
// history.go - records the movement history and the current state of the board

package session

import (
	"fmt"

	pieces "go_backend/game/piece"
)

// temporary storage for the movement history
var moveHistory []string

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

// temporary storage for the current state of the board
type PieceState struct {
	Color   string `json:"color"`
	Kind    string `json:"kind"`
	ImgFile string `json:"imgFile"`
	File    int    `json:"file"`
	Rank    int    `json:"rank"`
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
