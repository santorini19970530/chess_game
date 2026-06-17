package simulation

import (
	session "go_backend/game/session"
)

var maxPlies = 600

type MoveSelector func(gameID string) (string, error)

type Result struct {
	Result    session.GameResult
	Winner    string
	MoveCount int
}

func RunSingleAIGame(gameID string, pick MoveSelector) (Result, error) {
	for i := 0; i < maxPlies; i++ {
		g, err := session.RefreshGameSessionOutcomeByID(gameID)
		if err != nil {
			return Result{}, err
		}
		if g.Result != session.GameResultInProgress {
			return Result{
				Result:    g.Result,
				Winner:    g.Outcome.Winner,
				MoveCount: sessionMoveHistoryLen(gameID),
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

var ErrMaxPliesReached = &maxPliesError{}

type maxPliesError struct{}

func (e *maxPliesError) Error() string { return "max plies reached" }

func sessionMoveHistoryLen(gameID string) int {
	h, _ := session.MoveHistoryByID(gameID)
	return len(h)
}
