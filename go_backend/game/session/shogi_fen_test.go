package session

import (
	"strings"
	"testing"
)

const nifuNHKFen = "ln1g5/1ks1g3l/1p1pp1n2/p1pGs2rp/1P1N1ppp1/P1SB1P2P/1S1p1bPP1/LKG6/4R2NL[pp] w"

const nifuNHKErr = "nifu: two unpromoted black pawns on file d (6)"

// TestValidateShogiSetupBoard_Nifu - two unpromoted pawns on one file are illegal fen
func TestValidateShogiSetupBoard_Nifu(t *testing.T) {
	placement, _ := splitShogiPlacementAndHands(nifuNHKFen)
	board, err := parseShogiFENBoard(placement)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = validateShogiSetupBoard(board)
	if err == nil || err.Error() != nifuNHKErr {
		t.Fatalf("err=%v want %q", err, nifuNHKErr)
	}
}

// TestValidateShogiSetupBoard_StartOK - start placement is not nifu
func TestValidateShogiSetupBoard_StartOK(t *testing.T) {
	placement, _ := splitShogiPlacementAndHands(DefaultShogiStartFEN)
	board, err := parseShogiFENBoard(placement)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := validateShogiSetupBoard(board); err != nil {
		t.Fatalf("start: %v", err)
	}
}

// TestApplyShogiFEN_NifuRejected - apply does not install a nifu fen
func TestApplyShogiFEN_NifuRejected(t *testing.T) {
	if err := applyShogiFENToCurrentGlobals(nifuNHKFen); err == nil || err.Error() != nifuNHKErr {
		t.Fatalf("err=%v want %q", err, nifuNHKErr)
	}
}

// TestApplyShogiFEN_DeadPawnRejected - unpromoted pawn on the last rank has no move
func TestApplyShogiFEN_DeadPawnRejected(t *testing.T) {
	fen := "4k4/9/9/9/9/9/9/9/4K3p[] w - - 0 1"
	err := applyShogiFENToCurrentGlobals(fen)
	if err == nil || !strings.Contains(err.Error(), "no legal move") || !strings.Contains(err.Error(), "i1") {
		t.Fatalf("err=%v", err)
	}
}

// TestApplyShogiFEN_TwoKingsRejected - two kings of one side cannot be a setup
func TestApplyShogiFEN_TwoKingsRejected(t *testing.T) {
	fen := "3kk4/9/9/9/9/9/9/9/4K4[] w - - 0 1"
	err := applyShogiFENToCurrentGlobals(fen)
	if err == nil || !strings.Contains(err.Error(), "black has 2") {
		t.Fatalf("err=%v", err)
	}
}

// TestApplyShogiFEN_KingInHandRejected - king cannot sit in the hand field
func TestApplyShogiFEN_KingInHandRejected(t *testing.T) {
	fen := "4k4/9/9/9/9/9/9/9/4K4[K] w - - 0 1"
	err := applyShogiFENToCurrentGlobals(fen)
	if err == nil || !strings.Contains(err.Error(), "king in hand") {
		t.Fatalf("err=%v", err)
	}
}
