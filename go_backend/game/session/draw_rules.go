// CM3070 FP code
// draw_rules.go - implements draw rules

package session

import pieces "go_backend/game/piece"

// isFiftyMoveDraw - reports whether fifty move draw
func isFiftyMoveDraw() bool {
	return GetHalfmoveClock() >= 100
}

// isThreefoldRepetitionDraw - reports whether threefold repetition draw
func isThreefoldRepetitionDraw() bool {
	return GetCurrentPositionRepetitionCount() >= 3
}

// isInsufficientMaterialDraw - reports whether insufficient material draw
func isInsufficientMaterialDraw() bool {
	nonKings := make([]pieces.ChessPiece, 0, len(pieces.ChessPieces))
	for _, p := range pieces.ChessPieces {
		if p.Kind != pieces.King {
			nonKings = append(nonKings, p)
		}
	}
	if len(nonKings) == 0 {
		return true
	}
	if len(nonKings) == 1 {
		return nonKings[0].Kind == pieces.Bishop || nonKings[0].Kind == pieces.Knight
	}
	if len(nonKings) == 2 {
		a := nonKings[0]
		b := nonKings[1]
		if a.Kind == pieces.Bishop && b.Kind == pieces.Bishop {
			return squareColor(a.File, a.Rank) == squareColor(b.File, b.Rank)
		}
	}
	return false
}

// squareColor - returns square color
func squareColor(file, rank int) int {
	return (file + rank) % 2
}
