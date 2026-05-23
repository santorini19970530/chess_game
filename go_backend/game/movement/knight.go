// CM3070 FP code
// knight.go - implements the knight movement strategy

package movement

// KnightStrategy handles knight movement rules
type KnightStrategy struct{}

// Name - returns the name of the knight strategy
func (k KnightStrategy) Name() string { return "Knight" }

// LegalMoves - simulates knight L-shape jumps
func (k KnightStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}

	// knight can move in L-shape
	legal := make([]any, 0, 8)
	deltas := [][2]int{
		{1, 2}, {2, 1}, {2, -1}, {1, -2},
		{-1, -2}, {-2, -1}, {-2, 1}, {-1, 2},
	}

	for _, d := range deltas {
		targetFile := src.File + d[0]
		targetRank := src.Rank + d[1]
		if !isInsideBoard(targetFile, targetRank) {
			continue
		}

		targetPiece, occupied := getPieceAt(targetFile, targetRank)
		if !occupied || targetPiece.Color != ctx.Color {
			legal = append(legal, Square{File: targetFile, Rank: targetRank})
		}
	}

	return legal
}