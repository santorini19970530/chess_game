// CM3070 FP code
// xiangqi_soldier.go - implements the xiangqi soldier movement strategy

package movement

import pieces "go_backend/game/piece"

// xiangqiSoldierStrategy - forward one; after river also sideways. no retreat
type XiangqiSoldierStrategy struct{}

// Name - returns the piece strategy name
func (XiangqiSoldierStrategy) Name() string { return "XiangqiSoldier" }

// LegalMoves - returns legal destinations for this piece from the square
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
