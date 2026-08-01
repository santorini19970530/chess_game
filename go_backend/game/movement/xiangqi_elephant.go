package movement

import pieces "go_backend/game/piece"

// xiangqiElephantStrategy - 2-step diagonal; blocked by eye; cannot cross river
type XiangqiElephantStrategy struct{}

// Name - returns the piece strategy name
func (XiangqiElephantStrategy) Name() string { return "XiangqiElephant" }

// LegalMoves - returns legal destinations for this piece from the square
func (XiangqiElephantStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}
	deltas := [][2]int{{2, 2}, {2, -2}, {-2, 2}, {-2, -2}}
	legal := make([]any, 0, 4)
	for _, d := range deltas {
		eyeFile, eyeRank := src.File+d[0]/2, src.Rank+d[1]/2
		if _, blocked := getPieceAt(eyeFile, eyeRank); blocked {
			continue
		}
		tf, tr := src.File+d[0], src.Rank+d[1]
		if !isInsideXiangqiBoard(tf, tr) {
			continue
		}
		// cannot cross river.
		if ctx.Color == pieces.White && tr > 5 {
			continue
		}
		if ctx.Color == pieces.Black && tr < 6 {
			continue
		}
		legal = appendIfEnemyOrEmpty(legal, ctx, tf, tr)
	}
	return legal
}
