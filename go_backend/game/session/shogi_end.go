// CM3070 FP code
// shogi_end.go - shogi ply-end registry and outcome context

package session

import (
	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// shogiGameEndStrategies - ordered Shogi ply-end strategies; first ended wins
func shogiGameEndStrategies() []GameEndStrategy {
	return []GameEndStrategy{
		ShogiMateOrStalemateStrategy{},
		maxPlyStrategy(),
	}
}

// ensureShogiGameEndStrategies - registers the Shogi ply-end chain
func ensureShogiGameEndStrategies() {
	registerGameEndStrategies(GameTypeShogi, shogiGameEndStrategies()...)
}

// EvaluateShogiGameOutcome - runs the Shogi GameEndStrategy chain for the current ply
func EvaluateShogiGameOutcome() GameOutcome {
	ensureShogiGameEndStrategies()
	ctx := buildShogiPlyEndContext()
	ended, out := runGameEndStrategies(ctx)
	if ended {
		return out
	}
	if ctx.InCheck {
		return GameOutcome{
			Status:      "check",
			CheckedSide: ctx.SideToMove,
			LegalMoves:  ctx.LegalMoves,
			Message:     sideLabelFromText(ctx.SideToMove) + " is in check.",
		}
	}
	return GameOutcome{Status: "in_progress", LegalMoves: ctx.LegalMoves}
}

// buildShogiPlyEndContext - fills ply facts from the board plus move history length
func buildShogiPlyEndContext() *plyEndContext {
	side := CurrentTurnColor()
	ctx := &plyEndContext{
		GameType:   GameTypeShogi,
		SideToMove: string(side),
		PlyCount:   currentPlyCount(),
	}
	whiteOK, blackOK := false, false
	for _, p := range pieces.ChessPieces {
		if p.Kind != pieces.King {
			continue
		}
		if p.Color == pieces.White {
			whiteOK = true
		} else {
			blackOK = true
		}
	}
	if !whiteOK {
		ctx.LegalMoves = 0
		ctx.InCheck = true
		ctx.SideToMove = "white"
		return ctx
	}
	if !blackOK {
		ctx.LegalMoves = 0
		ctx.InCheck = true
		ctx.SideToMove = "black"
		return ctx
	}
	legal, err := shogiAllLegalUCIMoves()
	if err == nil {
		ctx.LegalMoves = len(legal)
	}
	ctx.InCheck = movement.ShogiCheckedColor() == side
	return ctx
}
