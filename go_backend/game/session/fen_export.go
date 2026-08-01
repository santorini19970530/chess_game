// CM3070 FP code
// fen_export.go - exports session position as fen text

package session

import "fmt"

// CurrentFEN - exports the current board/session state as a FEN string
func CurrentFEN() string {
	if boardFEN != "" {
		return boardFEN
	}
	activeColor := "w"
	if CurrentTurnColor() == "black" {
		activeColor = "b"
	}

	fullmoveNumber := len(moveHistory)/2 + 1
	return fmt.Sprintf(
		"%s %s %s %s %d %d",
		boardToKey(),
		activeColor,
		castlingRightsKey(),
		enPassantTargetKey(),
		halfmoveClock,
		fullmoveNumber,
	)
}
