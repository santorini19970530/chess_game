package session

import (
	"strings"
	"testing"
)

const chessStartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// TestApplyFEN_StartOK - start chess fen is a legal setup
func TestApplyFEN_StartOK(t *testing.T) {
	if err := applyFENToCurrentGlobals(chessStartFEN); err != nil {
		t.Fatalf("start: %v", err)
	}
}

// TestApplyFEN_PawnOnEighthRejected - a pawn cannot sit on rank 1 or 8
func TestApplyFEN_PawnOnEighthRejected(t *testing.T) {
	err := applyFENToCurrentGlobals("4P3/4k3/8/8/8/8/8/4K3 w - - 0 1")
	if err == nil || !strings.Contains(err.Error(), "pawn on e8") {
		t.Fatalf("err=%v", err)
	}
}

// TestApplyFEN_TwoKingsRejected - two kings of one side cannot be a setup
func TestApplyFEN_TwoKingsRejected(t *testing.T) {
	err := applyFENToCurrentGlobals("3kk3/8/8/8/8/8/8/4K3 w - - 0 1")
	if err == nil || !strings.Contains(err.Error(), "black has 2") {
		t.Fatalf("err=%v", err)
	}
}

// TestApplyXiangqiFEN_StartOK - start xiangqi fen is a legal setup
func TestApplyXiangqiFEN_StartOK(t *testing.T) {
	if err := applyXiangqiFENToCurrentGlobals(DefaultXiangqiStartFEN); err != nil {
		t.Fatalf("start: %v", err)
	}
}

// TestApplyXiangqiFEN_GeneralOutsidePalaceRejected - general must stay in the palace
func TestApplyXiangqiFEN_GeneralOutsidePalaceRejected(t *testing.T) {
	fen := "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/KNBA1ABNR w - - 0 1"
	err := applyXiangqiFENToCurrentGlobals(fen)
	if err == nil || !strings.Contains(err.Error(), "general outside palace") || !strings.Contains(err.Error(), "a1") {
		t.Fatalf("err=%v", err)
	}
}

// TestApplyXiangqiFEN_AdvisorOutsidePalaceRejected - advisor must stay in the palace
func TestApplyXiangqiFEN_AdvisorOutsidePalaceRejected(t *testing.T) {
	fen := "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/ANB1KABNR w - - 0 1"
	err := applyXiangqiFENToCurrentGlobals(fen)
	if err == nil || !strings.Contains(err.Error(), "advisor outside palace") || !strings.Contains(err.Error(), "a1") {
		t.Fatalf("err=%v", err)
	}
}

// TestApplyXiangqiFEN_ElephantCrossedRiverRejected - elephant cannot sit across the river
func TestApplyXiangqiFEN_ElephantCrossedRiverRejected(t *testing.T) {
	fen := "rnbakabnr/9/1c5c1/p1p1p1p1p/2B6/9/P1P1P1P1P/1C5C1/9/RN1AKABNR w - - 0 1"
	err := applyXiangqiFENToCurrentGlobals(fen)
	if err == nil || !strings.Contains(err.Error(), "elephant crossed the river") || !strings.Contains(err.Error(), "c6") {
		t.Fatalf("err=%v", err)
	}
}

// TestApplyXiangqiFEN_FlyingGeneralsRejected - two generals on one empty file cannot be a setup
func TestApplyXiangqiFEN_FlyingGeneralsRejected(t *testing.T) {
	fen := "4k4/9/1pc5b/p5p2/2n6/R8/8n/1C3A1CN/3P2N2/4K3R w"
	err := applyXiangqiFENToCurrentGlobals(fen)
	if err == nil || !strings.Contains(err.Error(), "flying generals") || !strings.Contains(err.Error(), "file e") {
		t.Fatalf("err=%v", err)
	}
}
