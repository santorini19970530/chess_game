// CM3070 FP code
// xiangqi_end.go - xiangqi ply-end registry and outcome context

package session

import (
	"fmt"
	"strings"

	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

const xiangqiNoCapturePlies = 60

type xiangqiPlyFact struct {
	Side      string
	GaveCheck bool
	ChaseKeys []string
}

var xiangqiIdlePly int
var xiangqiPositionCounts map[string]int
var xiangqiPositionKeys []string
var xiangqiPlyFacts []xiangqiPlyFact

// xiangqiGameEndStrategies - ordered Xiangqi ply-end strategies; first ended wins
func xiangqiGameEndStrategies() []GameEndStrategy {
	return []GameEndStrategy{
		XiangqiMateOrStalemateStrategy{},
		XiangqiPerpetualCheckStrategy{},
		XiangqiPerpetualChaseStrategy{},
		XiangqiMutualRepetitionStrategy{},
		XiangqiNoCaptureStrategy{},
		maxPlyStrategy(),
	}
}

// ensureXiangqiGameEndStrategies - registers the Xiangqi ply-end chain
func ensureXiangqiGameEndStrategies() {
	registerGameEndStrategies(GameTypeXiangqi, xiangqiGameEndStrategies()...)
}

// EvaluateXiangqiGameOutcome - runs the Xiangqi GameEndStrategy chain for the current ply
func EvaluateXiangqiGameOutcome() GameOutcome {
	ensureXiangqiGameEndStrategies()
	ctx := buildXiangqiPlyEndContext()
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

// buildXiangqiPlyEndContext - fills ply facts from the board plus recorded idle/repeat state
func buildXiangqiPlyEndContext() *plyEndContext {
	side := CurrentTurnColor()
	ctx := &plyEndContext{
		GameType:      GameTypeXiangqi,
		IdlePly:       xiangqiIdlePly,
		SideToMove:    string(side),
		PositionKey:   xiangqiPositionKey(),
		PositionCount: xiangqiPositionCounts[xiangqiPositionKey()],
		PlyCount:      currentPlyCount(),
	}
	if len(xiangqiPlyFacts) > 0 {
		last := xiangqiPlyFacts[len(xiangqiPlyFacts)-1]
		ctx.SideJustMoved = last.Side
		ctx.GaveCheck = last.GaveCheck
	}
	whiteOK, blackOK := xiangqiGeneralsPresent()
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
	legal, err := xiangqiAllLegalUCIMoves()
	if err == nil {
		ctx.LegalMoves = len(legal)
	}
	ctx.InCheck = movement.XiangqiCheckedColor() == side
	classifyXiangqiRepeatCycle(ctx)
	return ctx
}

// xiangqiGeneralsPresent - reports whether both generals are on the board
func xiangqiGeneralsPresent() (whiteOK, blackOK bool) {
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
	return whiteOK, blackOK
}

// xiangqiPositionKey - board placement plus side to move, ignoring fullmove
func xiangqiPositionKey() string {
	fen := exportXiangqiFEN()
	parts := strings.Fields(fen)
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	return fen
}

// resetXiangqiPlyTracking - clears Xiangqi idle and repetition history
func resetXiangqiPlyTracking() {
	xiangqiIdlePly = 0
	xiangqiPositionCounts = make(map[string]int)
	xiangqiPositionKeys = nil
	xiangqiPlyFacts = nil
}

// recordXiangqiStartPosition - counts the loaded position once so returns can reach threefold
func recordXiangqiStartPosition() {
	if xiangqiPositionCounts == nil {
		xiangqiPositionCounts = make(map[string]int)
	}
	key := xiangqiPositionKey()
	xiangqiPositionKeys = append(xiangqiPositionKeys, key)
	xiangqiPositionCounts[key]++
}

// recordXiangqiPlyAfterMove - records idle ply, check/chase facts, and the new position key
func recordXiangqiPlyAfterMove(mover pieces.PieceColor, capture bool, chaseKeys []string) {
	if capture {
		xiangqiIdlePly = 0
	} else {
		xiangqiIdlePly++
	}
	gaveCheck := movement.XiangqiCheckedColor() == CurrentTurnColor()
	xiangqiPlyFacts = append(xiangqiPlyFacts, xiangqiPlyFact{
		Side:      string(mover),
		GaveCheck: gaveCheck,
		ChaseKeys: append([]string(nil), chaseKeys...),
	})
	if xiangqiPositionCounts == nil {
		xiangqiPositionCounts = make(map[string]int)
	}
	key := xiangqiPositionKey()
	xiangqiPositionKeys = append(xiangqiPositionKeys, key)
	xiangqiPositionCounts[key]++
}

// classifyXiangqiRepeatCycle - sets check/chase/mutual fields when this position is at least threefold
func classifyXiangqiRepeatCycle(ctx *plyEndContext) {
	if ctx.PositionCount < 3 {
		return
	}
	ctx.CycleRepeat = true
	n := len(xiangqiPositionKeys)
	if n == 0 {
		return
	}
	key := xiangqiPositionKeys[n-1]
	prev := -1
	for i := n - 2; i >= 0; i-- {
		if xiangqiPositionKeys[i] == key {
			prev = i
			break
		}
	}
	if prev < 0 || prev > len(xiangqiPlyFacts) {
		return
	}
	facts := xiangqiPlyFacts[prev:]
	whiteFacts, blackFacts := splitXiangqiFactsBySide(facts)
	whiteAllCheck := xiangqiAllGaveCheck(whiteFacts)
	blackAllCheck := xiangqiAllGaveCheck(blackFacts)
	if whiteAllCheck && blackAllCheck {
		return
	}
	if whiteAllCheck {
		ctx.PerpetualCheckLoser = "white"
	} else if blackAllCheck {
		ctx.PerpetualCheckLoser = "black"
	}
	if side, kingOnly := xiangqiCycleChase(whiteFacts); side != "" {
		if kingOnly {
			ctx.ChaseIsKingOrSoldierOnly = true
		} else {
			ctx.PerpetualChaseLoser = "white"
		}
	}
	if side, kingOnly := xiangqiCycleChase(blackFacts); side != "" {
		if kingOnly {
			ctx.ChaseIsKingOrSoldierOnly = true
		} else if ctx.PerpetualChaseLoser == "" {
			ctx.PerpetualChaseLoser = "black"
		} else {
			ctx.PerpetualChaseLoser = ""
		}
	}
}

// splitXiangqiFactsBySide - splits cycle plies into white and black
func splitXiangqiFactsBySide(facts []xiangqiPlyFact) (white, black []xiangqiPlyFact) {
	for _, f := range facts {
		if f.Side == "black" {
			black = append(black, f)
		} else {
			white = append(white, f)
		}
	}
	return white, black
}

// xiangqiAllGaveCheck - reports whether every ply in the list gave check
func xiangqiAllGaveCheck(facts []xiangqiPlyFact) bool {
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

// xiangqiCycleChase - one shared chase target across every ply, or king/soldier-only
func xiangqiCycleChase(facts []xiangqiPlyFact) (side string, kingOrSoldier bool) {
	if len(facts) == 0 {
		return "", false
	}
	inter := facts[0].ChaseKeys
	if len(inter) == 0 {
		return "", false
	}
	for _, f := range facts[1:] {
		inter = intersectStrings(inter, f.ChaseKeys)
		if len(inter) == 0 {
			return "", false
		}
	}
	if len(inter) != 1 {
		return "", false
	}
	kind := strings.Split(inter[0], ":")[0]
	if kind == string(pieces.King) || kind == string(pieces.Pawn) {
		return facts[0].Side, true
	}
	return facts[0].Side, false
}

// intersectStrings - returns strings present in both lists
func intersectStrings(a, b []string) []string {
	seen := make(map[string]bool, len(b))
	for _, s := range b {
		seen[s] = true
	}
	out := make([]string, 0, len(a))
	for _, s := range a {
		if seen[s] {
			out = append(out, s)
		}
	}
	return out
}

// xiangqiUnprotectedAttackedKeys - enemy pieces attacked by attacker and not protected
func xiangqiUnprotectedAttackedKeys(attacker pieces.PieceColor) []string {
	victim := OpponentColor(attacker)
	out := make([]string, 0, 4)
	for _, p := range pieces.ChessPieces {
		if p.Color != victim {
			continue
		}
		if !movement.XiangqiSquareAttacked(p.File, p.Rank, attacker) {
			continue
		}
		if movement.XiangqiSquareAttacked(p.File, p.Rank, victim) {
			continue
		}
		out = append(out, fmt.Sprintf("%s:%d:%d", p.Kind, p.File, p.Rank))
	}
	return out
}

// newStringKeys - returns keys in after that were not in before
func newStringKeys(before, after []string) []string {
	seen := make(map[string]bool, len(before))
	for _, k := range before {
		seen[k] = true
	}
	out := make([]string, 0, len(after))
	for _, k := range after {
		if !seen[k] {
			out = append(out, k)
		}
	}
	return out
}

// xiangqiLossOutcome - win for the opponent of loser under a named WXF status
func xiangqiLossOutcome(status, loser, message string, legalMoves int) GameOutcome {
	winner := "black"
	if loser == "black" {
		winner = "white"
	}
	return GameOutcome{
		Status:     status,
		Winner:     winner,
		Loser:      loser,
		LegalMoves: legalMoves,
		Message:    message,
	}
}
