package session

import (
	pieces "go_backend/game/piece"
	"testing"
)

var initialChessPieces = append([]pieces.ChessPiece(nil), pieces.ChessPieces...)

func resetChessPieces() {
	pieces.ChessPieces = append([]pieces.ChessPiece(nil), initialChessPieces...)
	moveHistory = nil
	lastAppliedMove = nil
}

func pieceAt(file, rank int) (pieces.ChessPiece, bool) {
	for _, p := range pieces.ChessPieces {
		if p.File == file && p.Rank == rank {
			return p, true
		}
	}
	return pieces.ChessPiece{}, false
}

func TestApplyMoveByCommand_BlackPawnDoubleStep_UCIRejectedOnWhiteTurn(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("e7e5"); err == nil {
		t.Fatalf("expected black pawn double step to fail on white turn")
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

func TestApplyMoveByCommand_BlackPawnDoubleStep_SANFromInitialPositionRejected(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("g5"); err == nil {
		t.Fatalf("expected SAN g5 to fail on white turn")
	}
}

func TestApplyMoveByCommand_QueenStrategy(t *testing.T) {
	resetChessPieces()

	// Open d-file so white queen can move from d1 to d3.
	if _, err := ApplyMoveByCommand("d2d4"); err != nil {
		t.Fatalf("expected setup move d2d4 to succeed, got error: %v", err)
	}
	if _, err := ApplyMoveByCommand("a7a6"); err != nil {
		t.Fatalf("expected black reply a7a6 to succeed, got error: %v", err)
	}

	if _, err := ApplyMoveByCommand("d1d3"); err != nil {
		t.Fatalf("expected queen move d1d3 to succeed, got error: %v", err)
	}
	queen, ok := pieceAt(4, 3)
	if !ok || queen.Kind != pieces.Queen || queen.Color != pieces.White {
		t.Fatalf("expected white queen on d3 after move")
	}
	if _, err := ApplyMoveByCommand("a6a5"); err != nil {
		t.Fatalf("expected black reply a6a5 to succeed, got error: %v", err)
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
	if _, err := ApplyMoveByCommand("a7a6"); err != nil {
		t.Fatalf("expected black reply a7a6 to succeed, got error: %v", err)
	}

	if _, err := ApplyMoveByCommand("c3c5"); err == nil {
		t.Fatalf("expected c3c5 to fail for knight movement")
	}
}

func TestApplyMoveByCommand_SANQueenCapture_Qxd3(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Queen, File: 4, Rank: 5}, // Qd5
		{Color: pieces.Black, Kind: pieces.Pawn, File: 4, Rank: 3},  // pd3
	}
	moveHistory = nil // white to move

	normalized, err := ApplyMoveByCommand("Qxd3")
	if err != nil {
		t.Fatalf("expected SAN queen capture Qxd3 to succeed, got error: %v", err)
	}
	if normalized != "d5d3" {
		t.Fatalf("expected normalized move d5d3, got %q", normalized)
	}

	queen, ok := pieceAt(4, 3)
	if !ok || queen.Kind != pieces.Queen || queen.Color != pieces.White {
		t.Fatalf("expected white queen on d3 after capture")
	}
}

func TestApplyMoveByCommand_SANPawnCapture_Pxd4(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.Black, Kind: pieces.Pawn, File: 5, Rank: 5}, // pe5
		{Color: pieces.White, Kind: pieces.Pawn, File: 4, Rank: 4}, // pd4
	}
	moveHistory = []string{"White: e2e4"} // black to move

	normalized, err := ApplyMoveByCommand("Pxd4")
	if err != nil {
		t.Fatalf("expected SAN pawn capture Pxd4 to succeed, got error: %v", err)
	}
	if normalized != "e5d4" {
		t.Fatalf("expected normalized move e5d4, got %q", normalized)
	}

	pawn, ok := pieceAt(4, 4)
	if !ok || pawn.Kind != pieces.Pawn || pawn.Color != pieces.Black {
		t.Fatalf("expected black pawn on d4 after capture")
	}
}

func TestApplyMoveByCommand_EnPassant_UCI(t *testing.T) {
	resetChessPieces()

	if _, err := ApplyMoveByCommand("e2e4"); err != nil {
		t.Fatalf("expected e2e4 to succeed, got error: %v", err)
	}
	if _, err := ApplyMoveByCommand("a7a6"); err != nil {
		t.Fatalf("expected a7a6 to succeed, got error: %v", err)
	}
	if _, err := ApplyMoveByCommand("e4e5"); err != nil {
		t.Fatalf("expected e4e5 to succeed, got error: %v", err)
	}
	if _, err := ApplyMoveByCommand("d7d5"); err != nil {
		t.Fatalf("expected d7d5 to succeed, got error: %v", err)
	}

	normalized, err := ApplyMoveByCommand("e5d6")
	if err != nil {
		t.Fatalf("expected en passant e5d6 to succeed, got error: %v", err)
	}
	if normalized != "e5d6" {
		t.Fatalf("expected normalized move e5d6, got %q", normalized)
	}

	whitePawn, ok := pieceAt(4, 6)
	if !ok || whitePawn.Kind != pieces.Pawn || whitePawn.Color != pieces.White {
		t.Fatalf("expected white pawn on d6 after en passant")
	}
	if _, exists := pieceAt(4, 5); exists {
		t.Fatalf("expected captured black pawn at d5 to be removed")
	}
}
