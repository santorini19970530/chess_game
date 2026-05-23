package session

import (
	pieces "go_backend/game/piece"
	"testing"
)

var initialChessPieces = append([]pieces.ChessPiece(nil), pieces.ChessPieces...)

func resetChessPieces() {
	pieces.ChessPieces = append([]pieces.ChessPiece(nil), initialChessPieces...)
	moveHistory = nil
}

func pieceAt(file, rank int) (pieces.ChessPiece, bool) {
	for _, p := range pieces.ChessPieces {
		if p.File == file && p.Rank == rank {
			return p, true
		}
	}
	return pieces.ChessPiece{}, false
}

func TestApplyMoveByCommand_BlackPawnDoubleStep_UCI(t *testing.T) {
	resetChessPieces()

	normalized, err := ApplyMoveByCommand("e7e5")
	if err != nil {
		t.Fatalf("expected black pawn double step to succeed, got error: %v", err)
	}
	if normalized != "e7e5" {
		t.Fatalf("expected normalized move e7e5, got %q", normalized)
	}

	blackPawn, ok := pieceAt(5, 5)
	if !ok {
		t.Fatalf("expected piece on e5 after move")
	}
	if blackPawn.Kind != pieces.Pawn || blackPawn.Color != pieces.Black {
		t.Fatalf("expected black pawn on e5, got kind=%v color=%v", blackPawn.Kind, blackPawn.Color)
	}
}

func TestApplyMoveByCommand_BlackPawnDoubleStep_SANAfterWhiteMove(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("e4"); err != nil {
		t.Fatalf("expected white SAN move e4 to succeed, got error: %v", err)
	}

	normalized, err := ApplyMoveByCommand("e5")
	if err != nil {
		t.Fatalf("expected black SAN move e5 to succeed, got error: %v", err)
	}
	if normalized != "e7e5" {
		t.Fatalf("expected normalized move e7e5, got %q", normalized)
	}

	blackPawn, ok := pieceAt(5, 5)
	if !ok {
		t.Fatalf("expected piece on e5 after move")
	}
	if blackPawn.Kind != pieces.Pawn || blackPawn.Color != pieces.Black {
		t.Fatalf("expected black pawn on e5, got kind=%v color=%v", blackPawn.Kind, blackPawn.Color)
	}
}

func TestApplyMoveByCommand_BlackPawnDoubleStep_SANFromInitialPosition(t *testing.T) {
	resetChessPieces()

	normalized, err := ApplyMoveByCommand("g5")
	if err != nil {
		t.Fatalf("expected SAN g5 to resolve to black pawn start double step, got error: %v", err)
	}
	if normalized != "g7g5" {
		t.Fatalf("expected normalized move g7g5, got %q", normalized)
	}

	blackPawn, ok := pieceAt(7, 5)
	if !ok {
		t.Fatalf("expected piece on g5 after move")
	}
	if blackPawn.Kind != pieces.Pawn || blackPawn.Color != pieces.Black {
		t.Fatalf("expected black pawn on g5, got kind=%v color=%v", blackPawn.Kind, blackPawn.Color)
	}
}

func TestApplyMoveByCommand_QueenStrategy(t *testing.T) {
	resetChessPieces()

	// Open d-file so white queen can move from d1 to d3.
	if _, err := ApplyMoveByCommand("d2d4"); err != nil {
		t.Fatalf("expected setup move d2d4 to succeed, got error: %v", err)
	}

	if _, err := ApplyMoveByCommand("d1d3"); err != nil {
		t.Fatalf("expected queen move d1d3 to succeed, got error: %v", err)
	}
	queen, ok := pieceAt(4, 3)
	if !ok || queen.Kind != pieces.Queen || queen.Color != pieces.White {
		t.Fatalf("expected white queen on d3 after move")
	}

	// Queen cannot move in knight pattern.
	if _, err := ApplyMoveByCommand("d3e5"); err == nil {
		t.Fatalf("expected d3e5 to fail for queen movement")
	}
}

func TestApplyMoveByCommand_KnightStrategy(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("b1c3"); err != nil {
		t.Fatalf("expected knight move b1c3 to succeed, got error: %v", err)
	}
	knight, ok := pieceAt(3, 3)
	if !ok || knight.Kind != pieces.Knight || knight.Color != pieces.White {
		t.Fatalf("expected white knight on c3 after move")
	}

	if _, err := ApplyMoveByCommand("c3c5"); err == nil {
		t.Fatalf("expected c3c5 to fail for knight movement")
	}
}
