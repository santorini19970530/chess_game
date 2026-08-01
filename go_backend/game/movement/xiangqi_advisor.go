// CM3070 FP code
// xiangqi_advisor.go - implements the xiangqi advisor movement strategy

package movement

// xiangqiAdvisorStrategy - one diagonal step inside the palace
type XiangqiAdvisorStrategy struct{}

// Name - returns the piece strategy name
func (XiangqiAdvisorStrategy) Name() string { return "XiangqiAdvisor" }

// LegalMoves - returns legal destinations for this piece from the square
func (XiangqiAdvisorStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}
	legal := make([]any, 0, 4)
	for _, d := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		tf, tr := src.File+d[0], src.Rank+d[1]
		if !inXiangqiPalace(tf, tr, ctx.Color) {
			continue
		}
		legal = appendIfEnemyOrEmpty(legal, ctx, tf, tr)
	}
	return legal
}
