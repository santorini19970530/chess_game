// CM3070 FP code
// king.go - implements the king movement strategy

package movement

// KingStrategy handles king movement rules (castling excluded for now)
type KingStrategy struct{}

// Name - returns the name of the king strategy
func (k KingStrategy) Name() string { return "King" }

// LegalMoves - simulates king one-square movement in all directions
func (k KingStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}

	// king can move one square in any directions
	legal := make([]any, 0, 8)
	for df := -1; df <= 1; df++ {
		for dr := -1; dr <= 1; dr++ {
			if df == 0 && dr == 0 {
				continue
			}

			targetFile := src.File + df
			targetRank := src.Rank + dr
			if targetFile < 1 || targetFile > 8 || targetRank < 1 || targetRank > 8 {
				continue
			}

			targetPiece, occupied := getPieceAt(targetFile, targetRank)
			if !occupied || targetPiece.Color != ctx.Color {
				legal = append(legal, Square{File: targetFile, Rank: targetRank})
			}
		}
	}

	return legal
}