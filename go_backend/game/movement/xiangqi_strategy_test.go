package movement

import (
	"testing"

	pieces "go_backend/game/piece"
)

func TestXiangqiHorseBlockedByHump(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Knight, File: 2, Rank: 1}, // b1
		{Color: pieces.White, Kind: pieces.Pawn, File: 2, Rank: 2},    // b2 blocks b1→a3/c3
	}
	legal := XiangqiLegalSquares(pieces.Knight, pieces.White, 2, 1)
	for _, sq := range legal {
		if (sq.File == 1 && sq.Rank == 3) || (sq.File == 3 && sq.Rank == 3) {
			t.Fatalf("horse should be blocked toward rank 3, got %+v", legal)
		}
	}
}

func TestXiangqiSoldierNoDoubleStep(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Pawn, File: 1, Rank: 4},
	}
	legal := XiangqiLegalSquares(pieces.Pawn, pieces.White, 1, 4)
	if len(legal) != 1 || legal[0].File != 1 || legal[0].Rank != 5 {
		t.Fatalf("soldier should only step to a5, got %+v", legal)
	}
}

func TestXiangqiValidateChariot(t *testing.T) {
	pieces.ChessPieces = []pieces.ChessPiece{
		{Color: pieces.White, Kind: pieces.Rook, File: 1, Rank: 1},
	}
	if err := ValidateXiangqiMoveByStrategy(pieces.Rook, 1, 1, 1, 5, pieces.White); err != nil {
		t.Fatalf("chariot a1a5 should be legal: %v", err)
	}
	if err := ValidateXiangqiMoveByStrategy(pieces.Rook, 1, 1, 2, 2, pieces.White); err == nil {
		t.Fatal("chariot diagonal should be illegal")
	}
}
