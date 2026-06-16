package handlers

import (
	"testing"

	sessionpkg "go_backend/game/session"
)

func TestChooseBestLegalCandidate_PicksFirstLegal(t *testing.T) {
	legal := map[string]struct{}{
		"e2e4": {},
		"d2d4": {},
	}
	candidates := []AIPolicyCandidate{
		{Rank: 1, UCI: "g1f3"},
		{Rank: 2, UCI: "E2E4"},
		{Rank: 3, UCI: "d2d4"},
	}
	got := chooseBestLegalCandidate(candidates, legal)
	if got != "e2e4" {
		t.Fatalf("expected e2e4, got %q", got)
	}
}

func TestChooseBestLegalCandidate_ReturnsEmptyWhenNoLegal(t *testing.T) {
	legal := map[string]struct{}{
		"a2a3": {},
	}
	candidates := []AIPolicyCandidate{
		{Rank: 1, UCI: "g1f3"},
		{Rank: 2, UCI: "e2e4"},
	}
	got := chooseBestLegalCandidate(candidates, legal)
	if got != "" {
		t.Fatalf("expected empty selection, got %q", got)
	}
}

func TestToUCIMove_PromotionDefaultsToQueen(t *testing.T) {
	got := toUCIMove(5, 7, 5, 8, true)
	if got != "e7e8q" {
		t.Fatalf("expected e7e8q, got %q", got)
	}
}

func TestChooseBestLegalCandidate_RespectsProbOrderAmongLegal(t *testing.T) {
	legal := map[string]struct{}{
		"e2e4": {},
		"d2d4": {},
		"g1f3": {},
	}
	candidates := []AIPolicyCandidate{
		{Rank: 1, UCI: "g1f3", Prob: 0.9},
		{Rank: 2, UCI: "e2e4", Prob: 0.05}, // lower prob but still legal
	}
	got := chooseBestLegalCandidate(candidates, legal)
	if got != "g1f3" {
		t.Fatalf("expected g1f3 (highest prob legal), got %q", got)
	}
}

func TestLegalUCIMovesByID_EmptyWhenNoPiecesForSide(t *testing.T) {
	// This tests the helper path used by SelectAIMove when side has no pieces
	// (simulated via empty state slice)
	emptyPieces := []sessionpkg.PieceState{}
	moves, err := legalUCIMovesByID("fake", emptyPieces, "white")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(moves) != 0 {
		t.Fatalf("expected empty legal moves for side with no pieces, got %v", moves)
	}
}
