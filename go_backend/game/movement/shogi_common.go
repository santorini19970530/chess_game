package movement

import pieces "go_backend/game/piece"

func isInsideShogiBoard(file, rank int) bool {
	return file >= 1 && file <= 9 && rank >= 1 && rank <= 9
}

// ShogiInPromotionZone: White (Sente) ranks 7–9; Black (Gote) ranks 1–3.
func ShogiInPromotionZone(rank int, color pieces.PieceColor) bool {
	if color == pieces.White {
		return rank >= 7 && rank <= 9
	}
	return rank >= 1 && rank <= 3
}

// ShogiCanPromote: piece may promote if the move enters, leaves, or stays in the zone.
func ShogiCanPromote(kind pieces.PieceKind, fromRank, toRank int, color pieces.PieceColor) bool {
	if !shogiIsPromotable(kind) {
		return false
	}
	return ShogiInPromotionZone(fromRank, color) || ShogiInPromotionZone(toRank, color)
}

// ShogiMustPromote: pawn/lance on last rank; knight on last two ranks.
func ShogiMustPromote(kind pieces.PieceKind, toRank int, color pieces.PieceColor) bool {
	switch kind {
	case pieces.Pawn, pieces.Lance:
		if color == pieces.White {
			return toRank == 9
		}
		return toRank == 1
	case pieces.Knight:
		if color == pieces.White {
			return toRank >= 8
		}
		return toRank <= 2
	default:
		return false
	}
}

func shogiIsPromotable(kind pieces.PieceKind) bool {
	switch kind {
	case pieces.Pawn, pieces.Lance, pieces.Knight, pieces.Silver, pieces.Bishop, pieces.Rook:
		return true
	default:
		return false
	}
}

// ShogiPromotedKind maps an unpromoted piece to its promoted form.
func ShogiPromotedKind(kind pieces.PieceKind) (pieces.PieceKind, bool) {
	switch kind {
	case pieces.Pawn:
		return pieces.PromotedPawn, true
	case pieces.Lance:
		return pieces.PromotedLance, true
	case pieces.Knight:
		return pieces.PromotedKnight, true
	case pieces.Silver:
		return pieces.PromotedSilver, true
	case pieces.Bishop:
		return pieces.Horse, true
	case pieces.Rook:
		return pieces.Dragon, true
	default:
		return "", false
	}
}

func shogiForward(color pieces.PieceColor) int {
	if color == pieces.Black {
		return -1
	}
	return 1
}

func collectShogiSlidingMoves(ctx MovementBoard, src Square, directions [][2]int, capacity int) []any {
	legal := make([]any, 0, capacity)
	for _, dir := range directions {
		file := src.File + dir[0]
		rank := src.Rank + dir[1]
		for isInsideShogiBoard(file, rank) {
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

func appendShogiIfEnemyOrEmpty(legal []any, ctx MovementBoard, file, rank int) []any {
	if !isInsideShogiBoard(file, rank) {
		return legal
	}
	target, occupied := getPieceAt(file, rank)
	if !occupied || target.Color != ctx.Color {
		return append(legal, Square{File: file, Rank: rank})
	}
	return legal
}

func appendShogiSteps(legal []any, ctx MovementBoard, src Square, deltas [][2]int) []any {
	for _, d := range deltas {
		legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File+d[0], src.Rank+d[1])
	}
	return legal
}

// Gold-general deltas for White (Sente); Black uses flipped ranks.
func shogiGoldDeltas(color pieces.PieceColor) [][2]int {
	fwd := shogiForward(color)
	return [][2]int{
		{-1, fwd}, {0, fwd}, {1, fwd},
		{-1, 0}, {1, 0},
		{0, -fwd},
	}
}

func shogiSilverDeltas(color pieces.PieceColor) [][2]int {
	fwd := shogiForward(color)
	return [][2]int{
		{-1, fwd}, {0, fwd}, {1, fwd},
		{-1, -fwd}, {1, -fwd},
	}
}
