package movement

import pieces "go_backend/game/piece"

// XiangqiSoldierStrategy — forward one; after river also sideways. No retreat.
type XiangqiSoldierStrategy struct{}

func (XiangqiSoldierStrategy) Name() string { return "XiangqiSoldier" }

func (XiangqiSoldierStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}
	dir := 1
	if ctx.Color == pieces.Black {
		dir = -1
	}
	legal := make([]any, 0, 3)
	legal = appendIfEnemyOrEmpty(legal, ctx, src.File, src.Rank+dir)
	if xiangqiSoldierCrossedRiver(src.Rank, ctx.Color) {
		legal = appendIfEnemyOrEmpty(legal, ctx, src.File-1, src.Rank)
		legal = appendIfEnemyOrEmpty(legal, ctx, src.File+1, src.Rank)
	}
	return legal
}
