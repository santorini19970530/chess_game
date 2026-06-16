package handlers

import "testing"

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
