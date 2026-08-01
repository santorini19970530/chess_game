package movement

// shogi pawn strategy
type ShogiPawnStrategy struct{}

// Name - returns the piece strategy name
func (ShogiPawnStrategy) Name() string { return "ShogiPawn" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiPawnStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return appendShogiIfEnemyOrEmpty(nil, ctx, src.File, src.Rank+shogiForward(ctx.Color))
}

// shogi lance strategy
type ShogiLanceStrategy struct{}

// Name - returns the piece strategy name
func (ShogiLanceStrategy) Name() string { return "ShogiLance" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiLanceStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return collectShogiSlidingMoves(ctx, src, [][2]int{{0, shogiForward(ctx.Color)}}, 8)
}

// shogi knight strategy
type ShogiKnightStrategy struct{}

// Name - returns the piece strategy name
func (ShogiKnightStrategy) Name() string { return "ShogiKnight" }

// LegalMoves - returns legal destinations for this piece from the square
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

// shogi silver strategy
type ShogiSilverStrategy struct{}

// Name - returns the piece strategy name
func (ShogiSilverStrategy) Name() string { return "ShogiSilver" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiSilverStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return appendShogiSteps(nil, ctx, src, shogiSilverDeltas(ctx.Color))
}

// shogi gold strategy
type ShogiGoldStrategy struct{}

// Name - returns the piece strategy name
func (ShogiGoldStrategy) Name() string { return "ShogiGold" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiGoldStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return appendShogiSteps(nil, ctx, src, shogiGoldDeltas(ctx.Color))
}

// shogi king strategy
type ShogiKingStrategy struct{}

// Name - returns the piece strategy name
func (ShogiKingStrategy) Name() string { return "ShogiKing" }

// LegalMoves - returns legal destinations for this piece from the square
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

// shogi bishop strategy
type ShogiBishopStrategy struct{}

// Name - returns the piece strategy name
func (ShogiBishopStrategy) Name() string { return "ShogiBishop" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiBishopStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return collectShogiSlidingMoves(ctx, src, [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}, 16)
}

// shogi rook strategy
type ShogiRookStrategy struct{}

// Name - returns the piece strategy name
func (ShogiRookStrategy) Name() string { return "ShogiRook" }

// LegalMoves - returns legal destinations for this piece from the square
func (ShogiRookStrategy) LegalMoves(board any, from any) []any {
	ctx, src, ok := shogiBoardFrom(board, from)
	if !ok {
		return nil
	}
	return collectShogiSlidingMoves(ctx, src, [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}, 16)
}

// shogi horse strategy
type ShogiHorseStrategy struct{}

// Name - returns the piece strategy name
func (ShogiHorseStrategy) Name() string { return "ShogiHorse" }

// LegalMoves - returns legal destinations for this piece from the square
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

// shogi dragon strategy
type ShogiDragonStrategy struct{}

// Name - returns the piece strategy name
func (ShogiDragonStrategy) Name() string { return "ShogiDragon" }

// LegalMoves - returns legal destinations for this piece from the square
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

// shogiBoardFrom - performs shogi board from
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
