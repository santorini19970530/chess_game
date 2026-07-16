package movement

// --- Pawn ---

type ShogiPawnStrategy struct{}

func (ShogiPawnStrategy) Name() string { return "ShogiPawn" }

func (ShogiPawnStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return appendShogiIfEnemyOrEmpty(nil, ctx, src.File, src.Rank+shogiForward(ctx.Color))
}

// --- Lance ---

type ShogiLanceStrategy struct{}

func (ShogiLanceStrategy) Name() string { return "ShogiLance" }

func (ShogiLanceStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return collectShogiSlidingMoves(ctx, src, [][2]int{{0, shogiForward(ctx.Color)}}, 8)
}

// --- Knight ---

type ShogiKnightStrategy struct{}

func (ShogiKnightStrategy) Name() string { return "ShogiKnight" }

func (ShogiKnightStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	fwd := shogiForward(ctx.Color)
	legal := make([]any, 0, 2)
	legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File-1, src.Rank+2*fwd)
	legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File+1, src.Rank+2*fwd)
	return legal
}

// --- Silver ---

type ShogiSilverStrategy struct{}

func (ShogiSilverStrategy) Name() string { return "ShogiSilver" }

func (ShogiSilverStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return appendShogiSteps(nil, ctx, src, shogiSilverDeltas(ctx.Color))
}

// --- Gold (+ promoted pawn/lance/knight/silver) ---

type ShogiGoldStrategy struct{}

func (ShogiGoldStrategy) Name() string { return "ShogiGold" }

func (ShogiGoldStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return appendShogiSteps(nil, ctx, src, shogiGoldDeltas(ctx.Color))
}

// --- King ---

type ShogiKingStrategy struct{}

func (ShogiKingStrategy) Name() string { return "ShogiKing" }

func (ShogiKingStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	legal := make([]any, 0, 8)
	for df := -1; df <= 1; df++ {
		for dr := -1; dr <= 1; dr++ {
			if df == 0 && dr == 0 {
				continue
			}
			legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File+df, src.Rank+dr)
		}
	}
	return legal
}

// --- Bishop ---

type ShogiBishopStrategy struct{}

func (ShogiBishopStrategy) Name() string { return "ShogiBishop" }

func (ShogiBishopStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return collectShogiSlidingMoves(ctx, src, [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}, 16)
}

// --- Rook ---

type ShogiRookStrategy struct{}

func (ShogiRookStrategy) Name() string { return "ShogiRook" }

func (ShogiRookStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return collectShogiSlidingMoves(ctx, src, [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}, 16)
}

// --- Horse (promoted bishop): bishop + orthogonal king steps ---

type ShogiHorseStrategy struct{}

func (ShogiHorseStrategy) Name() string { return "ShogiHorse" }

func (ShogiHorseStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	legal := collectShogiSlidingMoves(ctx, src, [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}, 20)
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File+d[0], src.Rank+d[1])
	}
	return legal
}

// --- Dragon (promoted rook): rook + diagonal king steps ---

type ShogiDragonStrategy struct{}

func (ShogiDragonStrategy) Name() string { return "ShogiDragon" }

func (ShogiDragonStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	legal := collectShogiSlidingMoves(ctx, src, [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}, 20)
	for _, d := range [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}} {
		legal = appendShogiIfEnemyOrEmpty(legal, ctx, src.File+d[0], src.Rank+d[1])
	}
	return legal
}

func shogiBoardFrom(board any, from any) (MovementBoard, Square, bool) {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return MovementBoard{}, Square{}, false
	}
	src, ok := from.(Square)
	if !ok {
		return MovementBoard{}, Square{}, false
	}
	return ctx, src, true
}
