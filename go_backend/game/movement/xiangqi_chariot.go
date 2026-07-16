package movement

// XiangqiChariotStrategy — rook-like on 9×10.
type XiangqiChariotStrategy struct{}

func (XiangqiChariotStrategy) Name() string { return "XiangqiChariot" }

func (XiangqiChariotStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}
	directions := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	return collectXiangqiSlidingMoves(ctx, src, directions, 17)
}
