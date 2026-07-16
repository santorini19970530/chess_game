package movement

// XiangqiHorseStrategy — knight L-move blocked by adjacent “hump” square.
type XiangqiHorseStrategy struct{}

func (XiangqiHorseStrategy) Name() string { return "XiangqiHorse" }

func (XiangqiHorseStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}
	// delta + block square relative to src
	steps := []struct{ df, dr, bf, br int }{
		{1, 2, 0, 1}, {1, -2, 0, -1},
		{-1, 2, 0, 1}, {-1, -2, 0, -1},
		{2, 1, 1, 0}, {2, -1, 1, 0},
		{-2, 1, -1, 0}, {-2, -1, -1, 0},
	}
	legal := make([]any, 0, 8)
	for _, s := range steps {
		if _, blocked := getPieceAt(src.File+s.bf, src.Rank+s.br); blocked {
			continue
		}
		legal = appendIfEnemyOrEmpty(legal, ctx, src.File+s.df, src.Rank+s.dr)
	}
	return legal
}
