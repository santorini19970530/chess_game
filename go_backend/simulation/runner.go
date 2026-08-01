// CM3070 FP code
// runner.go - runs a single ai-vs-ai game loop

package simulation

import (
	session "go_backend/game/session"
)

// maximum number of plies to simulate in a single game
var maxPlies = 600

// function to select a move for the game
type MoveSelector func(gameID string) (string, error)

// result of a single game simulation
type Result struct {
	Result          session.GameResult
	Winner          string
	MoveCount       int
	HistoryDetailed []session.MoveHistoryEntry
}

// RunSingleAIGame - plays one ai-vs-ai game to completion using the move picker function
func RunSingleAIGame(gameID string, pick MoveSelector) (Result, error) {
	for i := 0; i < maxPlies; i++ {
		g, err := session.RefreshGameSessionOutcomeByID(gameID)
		if err != nil {
			return Result{}, err
		}
		if g.Result != session.GameResultInProgress {
			snap, _ := session.BuildSnapshotByID(gameID)
			return Result{
				Result:          g.Result,
				Winner:          g.Outcome.Winner,
				MoveCount:       len(snap.History),
				HistoryDetailed: snap.HistoryDetailed,
			}, nil
		}
		move, err := pick(gameID)
		if err != nil || move == "" {
			return Result{}, err
		}
		if _, err := session.ApplyMoveByCommandByID(gameID, move); err != nil {
			return Result{}, err
		}
	}
	return Result{}, ErrMaxPliesReached
}

// error when the maximum number of plies is reached
var ErrMaxPliesReached = &maxPliesError{}

// error when the maximum number of plies is reached
type maxPliesError struct{}

// Error - returns the error message
func (e *maxPliesError) Error() string { return "max plies reached" }

// sessionMoveHistoryLen - returns the length of the move history for the game
func sessionMoveHistoryLen(gameID string) int {
	h, _ := session.MoveHistoryByID(gameID)
	return len(h)
}
