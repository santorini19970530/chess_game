// CM3070 FP code
// chess_end_material.go - chess insufficient-material ply-end strategy

package session

// ChessInsufficientMaterialStrategy - king vs king (and the current sparse cases) is a draw
type ChessInsufficientMaterialStrategy struct{}

func (ChessInsufficientMaterialStrategy) Name() string { return "chess_insufficient_material" }

// AfterPly - ends as a draw when material cannot mate and a legal move remains
func (ChessInsufficientMaterialStrategy) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if ctx == nil || ctx.LegalMoves == 0 || !ctx.InsufficientMaterial {
		return false, GameOutcome{}
	}
	return true, GameOutcome{
		Status:     "draw_insufficient_material",
		LegalMoves: ctx.LegalMoves,
		Message:    "Draw by insufficient material.",
	}
}
