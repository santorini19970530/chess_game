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

// TestRunSingleAIGame_MaxPliesGuard - ply limit ends as a named draw, not a runner error
func TestRunSingleAIGame_MaxPliesGuard(t *testing.T) {
	old := session.DefaultMaxPlies
	session.DefaultMaxPlies = 3
	defer func() { session.DefaultMaxPlies = old }()

	game, err := session.CreateGame(session.GameModeAIVsAI, session.GameTypeChess, "white", 1, "", "beginner")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	res, err := RunSingleAIGame(game.ID, firstLegalMove)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Result != session.GameResultDraw {
		t.Fatalf("result=%q want draw", res.Result)
	}
	ended, err := session.GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ended.Outcome.Status != "draw_max_plies" {
		t.Fatalf("status=%q", ended.Outcome.Status)
	}
	if res.MoveCount != 3 {
		t.Fatalf("moves=%d want 3", res.MoveCount)
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

// TestRunSingleAIGame_ShogiImpasseDeclaresBeforePick - match path: winning FESA 5.3 is declared, not a ply
func TestRunSingleAIGame_ShogiImpasseDeclaresBeforePick(t *testing.T) {
	const fen = "9/G3K4/PPPPPPPPP/9/9/9/9/9/4k4[RRBB] w - - 0 1"
	game, err := session.CreateGame(session.GameModeAIVsAI, session.GameTypeShogi, "white", 1, fen, "beginner")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	picks := 0
	res, err := RunSingleAIGame(game.ID, func(string) (string, error) {
		picks++
		return "e8e9", nil
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if picks != 0 {
		t.Fatalf("impasse must not pick a ply, picks=%d", picks)
	}
	if res.Result != session.GameResultWhiteWin {
		t.Fatalf("result=%q", res.Result)
	}
	ended, err := session.GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ended.Outcome.Status != "impasse" {
		t.Fatalf("status=%q want impasse", ended.Outcome.Status)
	}
}
