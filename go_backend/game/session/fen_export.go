package session

import "fmt"

// CurrentFEN exports the current board/session state as a FEN string.
func CurrentFEN() string {
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
