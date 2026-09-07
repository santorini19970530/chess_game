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
		ShogiContinuousCheckStrategy{},
		ShogiSennichiteStrategy{},
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
	classifyShogiRepeatCycle(ctx)
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

// classifyShogiRepeatCycle - sets check-loop / sennichite fields when this position is at least fourfold
func classifyShogiRepeatCycle(ctx *plyEndContext) {
	if ctx.PositionCount < 4 {
		return
	}
	ctx.CycleRepeat = true
	n := len(shogiPositionKeys)
	if n == 0 {
		return
	}
	key := shogiPositionKeys[n-1]
	prev := -1
	for i := n - 2; i >= 0; i-- {
		if shogiPositionKeys[i] == key {
			prev = i
			break
		}
	}
	if prev < 0 || prev > len(shogiPlyFacts) {
		return
	}
	facts := shogiPlyFacts[prev:]
	whiteFacts, blackFacts := splitShogiFactsBySide(facts)
	whiteAllCheck := shogiAllGaveCheck(whiteFacts)
	blackAllCheck := shogiAllGaveCheck(blackFacts)
	if whiteAllCheck && blackAllCheck {
		return
	}
	if whiteAllCheck {
		ctx.PerpetualCheckLoser = "white"
	} else if blackAllCheck {
		ctx.PerpetualCheckLoser = "black"
	}
}

// splitShogiFactsBySide - splits cycle plies into white and black
func splitShogiFactsBySide(facts []shogiPlyFact) (white, black []shogiPlyFact) {
	for _, f := range facts {
		if f.Side == "black" {
			black = append(black, f)
		} else {
			white = append(white, f)
		}
	}
	return white, black
}

// shogiAllGaveCheck - reports whether every ply in the list gave check
func shogiAllGaveCheck(facts []shogiPlyFact) bool {
	if len(facts) == 0 {
		return false
	}
	for _, f := range facts {
		if !f.GaveCheck {
			return false
		}
	}
	return true
}
