package movement

// xiangqiGeneralStrategy - one orthogonal step inside the palace
type XiangqiGeneralStrategy struct{}

// Name - returns the piece strategy name
func (XiangqiGeneralStrategy) Name() string { return "XiangqiGeneral" }

// LegalMoves - returns legal destinations for this piece from the square
func (XiangqiGeneralStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}
	legal := make([]any, 0, 4)
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		tf, tr := src.File+d[0], src.Rank+d[1]
		if !inXiangqiPalace(tf, tr, ctx.Color) {
			continue
		}
		legal = appendIfEnemyOrEmpty(legal, ctx, tf, tr)
	}
	return legal
}
