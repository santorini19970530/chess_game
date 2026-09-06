// CM3070 FP code
// runner_test.go - tests for runner

package simulation

import (
	"testing"

	session "go_backend/game/session"
)

// firstLegalMove - returns first legal move
func firstLegalMove(gameID string) (string, error) {
	snap, err := session.BuildSnapshotByID(gameID)
	if err != nil {
		return "", err
	}
	for _, p := range snap.State {
		dests, err := session.LegalMovesForSquareByID(gameID, p.File, p.Rank)
		if err != nil {
			continue
		}
		for _, d := range dests {
			uci := toUCIMove(p.File, p.Rank, d.File, d.Rank, d.RequiresPromotion)
			if uci != "" {
				return uci, nil
			}
		}
	}
	return "", nil
}

// toUCIMove - converts to uci move
func toUCIMove(ff, fr, tf, tr int, promo bool) string {
	if ff < 1 || ff > 8 || tf < 1 || tf > 8 || fr < 1 || fr > 8 || tr < 1 || tr > 8 {
		return ""
	}
	m := string('a'+byte(ff-1)) + string('0'+byte(fr)) + string('a'+byte(tf-1)) + string('0'+byte(tr))
	if promo {
		m += "q"
	}
	return m
}

// TestRunSingleAIGame_MaxPliesGuard - checks run single ai game max plies guard
func TestRunSingleAIGame_MaxPliesGuard(t *testing.T) {
	old := maxPlies
	maxPlies = 3
	defer func() { maxPlies = old }()

	game, err := session.CreateGame(session.GameModeAIVsAI, session.GameTypeChess, "white", 1, "", "beginner")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = RunSingleAIGame(game.ID, firstLegalMove)
	if err == nil {
		t.Fatalf("expected ErrMaxPliesReached, got nil")
	}
	if err != ErrMaxPliesReached {
		t.Fatalf("expected maxPlies error, got %T %v", err, err)
	}
}

// TestRunSingleAIGame_StopsWhenXiangqiAlreadyEnded - ended Result must return without picking a ply
func TestRunSingleAIGame_StopsWhenXiangqiAlreadyEnded(t *testing.T) {
	const mateFEN = "R3k3R/9/9/9/9/9/9/9/9/4K4 b - - 0 1"
	game, err := session.CreateGame(session.GameModeAIVsAI, session.GameTypeXiangqi, "white", 1, mateFEN, "beginner")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	picks := 0
	res, err := RunSingleAIGame(game.ID, func(string) (string, error) {
		picks++
		return "a4a5", nil
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if picks != 0 {
		t.Fatalf("ended game must not call Fairy-Stockfish pick, picks=%d", picks)
	}
	if res.Result == session.GameResultInProgress {
		t.Fatalf("result=%q", res.Result)
	}
}
