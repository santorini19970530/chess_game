// CM3070 FP code
// runner.go - runs a single ai-vs-ai game loop

package simulation

import (
	session "go_backend/game/session"
)

// MoveSelector - picks the next ai move for one ply
type MoveSelector func(gameID string) (string, error)

// Result - summary of one finished ai-vs-ai game
type Result struct {
	Result          session.GameResult
	Winner          string
	MoveCount       int
	HistoryDetailed []session.MoveHistoryEntry
}

// RunSingleAIGame - plays one ai-vs-ai game until GameEndStrategy sets Result
func RunSingleAIGame(gameID string, pick MoveSelector) (Result, error) {
	limit := session.DefaultMaxPlies
	for i := 0; i < limit; i++ {
		g, err := session.RefreshGameSessionOutcomeByID(gameID)
		if err != nil {
			return Result{}, err
		}
		if g.Result != session.GameResultInProgress {
			return endedSimulationResult(gameID, g)
		}
		move, err := pick(gameID)
		if err != nil || move == "" {
			return Result{}, err
		}
		if _, err := session.ApplyMoveByCommandByID(gameID, move); err != nil {
			return Result{}, err
		}
	}
	g, err := session.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		return Result{}, err
	}
	if g.Result != session.GameResultInProgress {
		return endedSimulationResult(gameID, g)
	}
	return Result{}, ErrMaxPliesReached
}

// endedSimulationResult - snapshots an already-ended session as a simulation result
func endedSimulationResult(gameID string, g session.GameSession) (Result, error) {
	snap, _ := session.BuildSnapshotByID(gameID)
	return Result{
		Result:          g.Result,
		Winner:          g.Outcome.Winner,
		MoveCount:       len(snap.History),
		HistoryDetailed: snap.HistoryDetailed,
	}, nil
}

// ErrMaxPliesReached - backup if the ply-limit strategy did not set Result
var ErrMaxPliesReached = &maxPliesError{}

type maxPliesError struct{}

func (e *maxPliesError) Error() string { return "max plies reached" }

// sessionMoveHistoryLen - returns the length of the move history for the game
func sessionMoveHistoryLen(gameID string) int {
	h, _ := session.MoveHistoryByID(gameID)
	return len(h)
}
