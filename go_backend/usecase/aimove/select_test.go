// CM3070 FP code
// select_test.go - tests for select

package aimove

import "testing"

// TestChooseBestLegalCandidate_PicksFirstLegal - picks the first policy candidate present in the legal set
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
