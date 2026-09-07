// CM3070 FP code
// preview_move.go - restore-safe child fen derivation for what-if preview

package session

import (
	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// PreviewChildResult - child position snapshot for preview-move (live session unchanged)
type PreviewChildResult struct {
	Command     string
	FEN         string
	State       []PieceState
	Captured    CapturedSummary
	CurrentTurn string
	CheckedSide string
}

// PreviewChildFENByCommandByID - returns normalized command and child fen without committing session, clock, or history
func PreviewChildFENByCommandByID(gameID, commandText string) (string, string, error) {
	out, err := PreviewChildPositionByCommandByID(gameID, commandText)
	if err != nil {
		return "", "", err
	}
	return out.Command, out.FEN, nil
}

// PreviewChildPositionByCommandByID - applies a candidate under lock, snapshots child board, then restores live state
func PreviewChildPositionByCommandByID(gameID, commandText string) (PreviewChildResult, error) {
	game, err := lockRuntimeStateByID(gameID)
	if err != nil {
		return PreviewChildResult{}, err
	}
	defer unlockRuntimeStateByID(game)

	if err := rejectIfGameOverLocked(game); err != nil {
		return PreviewChildResult{}, err
	}

	sessionSnap := cloneGameSession(game.Session)
	stateSnap := cloneRuntimeState(game.State)
	handsSnap := cloneShogiHandState(shogiHands)
	defer func() {
		game.Session = sessionSnap
		game.State = stateSnap
		shogiHands = handsSnap
		game.bindToGlobals()
	}()

	var normalized string
	switch game.Session.Type {
	case GameTypeXiangqi:
		normalized, err = applyXiangqiUCIMove(commandText)
	case GameTypeShogi:
		normalized, err = applyShogiUCIMove(commandText)
	default:
		normalized, err = applyMoveByCommandCurrentLoaded(commandText)
	}
	if err != nil {
		return PreviewChildResult{}, err
	}

	captured := GetCapturedSummary()
	checked := CheckedSideLabel()
	switch game.Session.Type {
	case GameTypeXiangqi:
		checked = string(movement.XiangqiCheckedColor())
		captured = GetXiangqiCapturedSummary()
	case GameTypeShogi:
		checked = string(movement.ShogiCheckedColor())
		captured = shogiHandsSummary()
	}

	return PreviewChildResult{
		Command:     normalized,
		FEN:         CurrentFEN(),
		State:       GetBoardState(),
		Captured:    captured,
		CurrentTurn: CurrentTurnLabel(),
		CheckedSide: checked,
	}, nil
}

// cloneGameSession - deep-copies session including clock pointer fields
func cloneGameSession(in GameSession) GameSession {
	out := in
	if in.Clock != nil {
		clk := *in.Clock
		out.Clock = &clk
	}
	return out
}

// cloneRuntimeState - deep-copies runtime board/history fields used across unlock cycles
func cloneRuntimeState(in RuntimeState) RuntimeState {
	out := in
	out.Pieces = append([]pieces.ChessPiece(nil), in.Pieces...)
	out.MoveHistory = append([]string(nil), in.MoveHistory...)
	out.MoveHistoryDetailed = append([]MoveHistoryEntry(nil), in.MoveHistoryDetailed...)
	out.PositionCounts = copyStringIntMap(in.PositionCounts)
	if in.CurrentTurnOverride != nil {
		c := *in.CurrentTurnOverride
		out.CurrentTurnOverride = &c
	}
	if in.LastAppliedMove != nil {
		mv := *in.LastAppliedMove
		out.LastAppliedMove = &mv
	}
	out.XiangqiPositionCounts = copyStringIntMap(in.XiangqiPositionCounts)
	out.XiangqiPositionKeys = append([]string(nil), in.XiangqiPositionKeys...)
	out.XiangqiPlyFacts = copyXiangqiPlyFacts(in.XiangqiPlyFacts)
	out.ShogiPositionCounts = copyStringIntMap(in.ShogiPositionCounts)
	out.ShogiPositionKeys = append([]string(nil), in.ShogiPositionKeys...)
	out.ShogiPlyFacts = copyShogiPlyFacts(in.ShogiPlyFacts)
	return out
}

// cloneShogiHandState - copies white/black hand maps so preview capture/drop can roll back
func cloneShogiHandState(in shogiHandState) shogiHandState {
	return shogiHandState{
		white: copyPieceKindIntMap(in.white),
		black: copyPieceKindIntMap(in.black),
	}
}

// copyPieceKindIntMap - returns a shallow copy of a piece-kind count map
func copyPieceKindIntMap(in map[pieces.PieceKind]int) map[pieces.PieceKind]int {
	if in == nil {
		return map[pieces.PieceKind]int{}
	}
	out := make(map[pieces.PieceKind]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
