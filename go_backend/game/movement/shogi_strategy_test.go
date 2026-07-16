package movement

import (
	"testing"

	pieces "go_backend/game/piece"
)

func TestShogiPawnForwardOnly(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Pawn, File: 5, Rank: 3},
	}
	legal := ShogiLegalSquares(pieces.Pawn, pieces.White, 5, 3)
	if len(legal) != 1 || legal[0].File != 5 || legal[0].Rank != 4 {
		t.Fatalf("pawn should step to e4, got %+v", legal)
	}
}

func TestShogiKnightJumps(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Knight, File: 2, Rank: 1},
		{Color: pieces.White, Kind: pieces.Pawn, File: 2, Rank: 2}, // does not block knight
	}
	legal := ShogiLegalSquares(pieces.Knight, pieces.White, 2, 1)
	want := map[[2]int]bool{{1, 3}: true, {3, 3}: true}
	if len(legal) != 2 {
		t.Fatalf("knight jumps expected 2, got %+v", legal)
	}
	for _, sq := range legal {
		if !want[[2]int{sq.File, sq.Rank}] {
			t.Fatalf("unexpected knight square %+v in %+v", sq, legal)
		}
	}
}

func TestShogiLanceBlocked(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Lance, File: 1, Rank: 1},
		{Color: pieces.White, Kind: pieces.Pawn, File: 1, Rank: 3},
	}
	legal := ShogiLegalSquares(pieces.Lance, pieces.White, 1, 1)
	for _, sq := range legal {
		if sq.Rank >= 3 {
			t.Fatalf("lance should stop before own pawn, got %+v", legal)
		}
	}
	if len(legal) != 1 || legal[0].Rank != 2 {
		t.Fatalf("lance should only reach a2, got %+v", legal)
	}
}

func TestShogiSilverVsGold(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Silver, File: 5, Rank: 5},
	}
	silver := ShogiLegalSquares(pieces.Silver, pieces.White, 5, 5)
	hasSide := false
	hasBack := false
	for _, sq := range silver {
		if sq.File == 4 && sq.Rank == 5 {
			hasSide = true
		}
		if sq.File == 5 && sq.Rank == 4 {
			hasBack = true
		}
	}
	if hasSide || hasBack {
		t.Fatalf("silver must not move sideways/straight back, got %+v", silver)
	}

	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Gold, File: 5, Rank: 5},
	}
	gold := ShogiLegalSquares(pieces.Gold, pieces.White, 5, 5)
	hasSide, hasBack = false, false
	hasBackDiag := false
	for _, sq := range gold {
		if sq.File == 4 && sq.Rank == 5 {
			hasSide = true
		}
		if sq.File == 5 && sq.Rank == 4 {
			hasBack = true
		}
		if sq.File == 4 && sq.Rank == 4 {
			hasBackDiag = true
		}
	}
	if !hasSide || !hasBack {
		t.Fatalf("gold should move side and back, got %+v", gold)
	}
	if hasBackDiag {
		t.Fatalf("gold must not move back-diagonal, got %+v", gold)
	}
}

func TestShogiPromotedMovesAsGold(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.PromotedPawn, File: 5, Rank: 8},
	}
	tokin := ShogiLegalSquares(pieces.PromotedPawn, pieces.White, 5, 8)
	gold := ShogiLegalSquares(pieces.Gold, pieces.White, 5, 8)
	if len(tokin) != len(gold) {
		t.Fatalf("tokin len=%d gold len=%d", len(tokin), len(gold))
	}
}

func TestShogiDragonHasDiagonalStep(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Dragon, File: 5, Rank: 5},
	}
	legal := ShogiLegalSquares(pieces.Dragon, pieces.White, 5, 5)
	found := false
	for _, sq := range legal {
		if sq.File == 6 && sq.Rank == 6 {
			found = true
		}
	}
	if !found {
		t.Fatalf("dragon should have diagonal king-step, got %+v", legal)
	}
}

func TestShogiPromotionHelpers(t *testing.T) {
	if !ShogiInPromotionZone(7, pieces.White) || ShogiInPromotionZone(6, pieces.White) {
		t.Fatal("white promotion zone ranks 7-9")
	}
	if !ShogiCanPromote(pieces.Pawn, 6, 7, pieces.White) {
		t.Fatal("pawn entering zone may promote")
	}
	if !ShogiMustPromote(pieces.Pawn, 9, pieces.White) {
		t.Fatal("pawn to rank 9 must promote")
	}
	if !ShogiMustPromote(pieces.Knight, 8, pieces.White) {
		t.Fatal("knight to rank 8 must promote")
	}
	if kind, ok := ShogiPromotedKind(pieces.Rook); !ok || kind != pieces.Dragon {
		t.Fatalf("rook promotes to dragon, got %q ok=%v", kind, ok)
	}
}

func TestShogiCheckFilter(t *testing.T) {
	// White king e1; black rook on e9 open file — white pawn e2 cannot move off-file? 
	// Simpler: white king e5, black gold attacks e6; white gold on e4 moving away leaves king in check from black rook on e9.
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.King, File: 5, Rank: 5},
		{Color: pieces.White, Kind: pieces.Gold, File: 5, Rank: 6}, // blocks rook
		{Color: pieces.Black, Kind: pieces.Rook, File: 5, Rank: 9},
	}
	src := pieces.ChessPiece{Color: pieces.White, Kind: pieces.Gold, File: 5, Rank: 6}
	if !ShogiWouldLeaveKingInCheck(src, 5, 6, 4, 6) {
		t.Fatal("moving blocking gold sideways should leave king in check")
	}
	if ShogiWouldLeaveKingInCheck(src, 5, 6, 5, 7) {
		t.Fatal("moving blocker up the file should stay safe")
	}
}

func TestShogiValidateRook(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Rook, File: 2, Rank: 2},
	}
	if err := ValidateShogiMoveByStrategy(pieces.Rook, 2, 2, 2, 8, pieces.White); err != nil {
		t.Fatalf("rook b2b8 should be legal: %v", err)
	}
	if err := ValidateShogiMoveByStrategy(pieces.Rook, 2, 2, 3, 3, pieces.White); err == nil {
		t.Fatal("rook diagonal should be illegal")
	}
}
