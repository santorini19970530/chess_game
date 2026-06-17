package simulation

import (
	"testing"

	session "go_backend/game/session"
)

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