// CM3070 FP code
// common_rules.go - implements the common rules for sliding pieces

package movement

// isInsideBoard checks whether a square is inside the board.
func isInsideBoard(file, rank int) bool {
	return file >= 1 && file <= 8 && rank >= 1 && rank <= 8
}

// collectSlidingMoves walks rays for sliding pieces (rook/bishop/queen).
func collectSlidingMoves(ctx MovementBoard, src Square, directions [][2]int, capacity int) []any {
	legal := make([]any, 0, capacity)

	for _, dir := range directions {
		file := src.File + dir[0]
		rank := src.Rank + dir[1]
		for isInsideBoard(file, rank) {
			targetPiece, occupied := getPieceAt(file, rank)
			if !occupied {
				legal = append(legal, Square{File: file, Rank: rank})
			} else {
				// Can capture opponent piece, but cannot move through any piece.
				if targetPiece.Color != ctx.Color {
					legal = append(legal, Square{File: file, Rank: rank})
				}
				break
			}
			file += dir[0]
			rank += dir[1]
		}
	}

	return legal
}