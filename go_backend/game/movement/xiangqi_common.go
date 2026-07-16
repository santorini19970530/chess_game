package movement

import pieces "go_backend/game/piece"

func isInsideXiangqiBoard(file, rank int) bool {
	return file >= 1 && file <= 9 && rank >= 1 && rank <= 10
}

func inXiangqiPalace(file, rank int, color pieces.PieceColor) bool {
	if file < 4 || file > 6 {
		return false
	}
	if color == pieces.White {
		return rank >= 1 && rank <= 3
	}
	return rank >= 8 && rank <= 10
}

// White river is between rank 5 and 6; white has crossed when rank >= 6.
func xiangqiSoldierCrossedRiver(rank int, color pieces.PieceColor) bool {
	if color == pieces.White {
		return rank >= 6
	}
	return rank <= 5
}

func collectXiangqiSlidingMoves(ctx MovementBoard, src Square, directions [][2]int, capacity int) []any {
	legal := make([]any, 0, capacity)
	for _, dir := range directions {
		file := src.File + dir[0]
		rank := src.Rank + dir[1]
		for isInsideXiangqiBoard(file, rank) {
			targetPiece, occupied := getPieceAt(file, rank)
			if !occupied {
				legal = append(legal, Square{File: file, Rank: rank})
			} else {
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

func appendIfEnemyOrEmpty(legal []any, ctx MovementBoard, file, rank int) []any {
	if !isInsideXiangqiBoard(file, rank) {
		return legal
	}
	target, occupied := getPieceAt(file, rank)
	if !occupied || target.Color != ctx.Color {
		return append(legal, Square{File: file, Rank: rank})
	}
	return legal
}
