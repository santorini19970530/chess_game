// CM3070 FP code
// shogi_end.go - shogi ply-end registry and outcome context

package session

import (
	"strings"

	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

type shogiPlyFact struct {
	Side      string
	GaveCheck bool
}

var shogiPositionCounts map[string]int
var shogiPositionKeys []string
var shogiPlyFacts []shogiPlyFact

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

// buildShogiPlyEndContext - fills ply facts from the board, hands, side, and recorded repeats
func buildShogiPlyEndContext() *plyEndContext {
	side := CurrentTurnColor()
	ctx := &plyEndContext{
		GameType:      GameTypeShogi,
		SideToMove:    string(side),
		PositionKey:   shogiPositionKey(),
		PositionCount: shogiPositionCounts[shogiPositionKey()],
		PlyCount:      currentPlyCount(),
	}
	if len(shogiPlyFacts) > 0 {
		last := shogiPlyFacts[len(shogiPlyFacts)-1]
		ctx.SideJustMoved = last.Side
		ctx.GaveCheck = last.GaveCheck
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

// shogiPositionKey - board placement, hands, and side to move, ignoring fullmove
func shogiPositionKey() string {
	fen := exportShogiFEN()
	parts := strings.Fields(fen)
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	return fen
}

// resetShogiPlyTracking - clears Shogi repetition history
func resetShogiPlyTracking() {
	shogiPositionCounts = make(map[string]int)
	shogiPositionKeys = nil
	shogiPlyFacts = nil
}

// recordShogiStartPosition - counts the loaded position once so returns can reach fourfold
func recordShogiStartPosition() {
	if shogiPositionCounts == nil {
		shogiPositionCounts = make(map[string]int)
	}
	key := shogiPositionKey()
	shogiPositionKeys = append(shogiPositionKeys, key)
	shogiPositionCounts[key]++
}

// recordShogiPlyAfterMove - records check fact and the new board+hands+side key
func recordShogiPlyAfterMove(mover pieces.PieceColor) {
	gaveCheck := movement.ShogiCheckedColor() == CurrentTurnColor()
	shogiPlyFacts = append(shogiPlyFacts, shogiPlyFact{
		Side:      string(mover),
		GaveCheck: gaveCheck,
	})
	if shogiPositionCounts == nil {
		shogiPositionCounts = make(map[string]int)
	}
	key := shogiPositionKey()
	shogiPositionKeys = append(shogiPositionKeys, key)
	shogiPositionCounts[key]++
}
