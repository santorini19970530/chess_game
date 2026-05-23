// CM3070 FP code
// pawn.go - implements the pawn movement strategy
// en passant to be added later

package movement

import (
	pieces "go_backend/game/piece"
)

// PawnStrategy is a placeholder for pawn move rules.
type PawnStrategy struct{}

// Name - returns the name of the pawn strategy
func (p PawnStrategy) Name() string { return "Pawn" }

// LegalMoves - simulates the movement of a pawn
func (p PawnStrategy) LegalMoves(board any, from any) []any {
	ctx, ok := board.(MovementBoard)
	if !ok {
		return nil
	}
	src, ok := from.(Square)
	if !ok {
		return nil
	}

	// set up the displacement of pawn (white positive, black negative)
	dir := 1
	startRank := 2
	if ctx.Color == pieces.Black {
		dir = -1
		startRank = 7
	}

	legal := make([]any, 0, 4)

	// basic movement is one square forward if the next square is empty
	if _, occupied := getPieceAt(src.File, src.Rank+dir); !occupied {
		legal = append(legal, Square{File: src.File, Rank: src.Rank + dir})
		// but when it is at the start rank, it can move two squares forward if the next two squares are empty,
		// so need to check if one more square is empty or not
		if src.Rank == startRank {
			if _, occupiedTwo := getPieceAt(src.File, src.Rank+2*dir); !occupiedTwo {
				legal = append(legal, Square{File: src.File, Rank: src.Rank + 2*dir})
			}
		}
	}

	// diagonal one forward movement, which is for capturing only
	for _, df := range []int{-1, 1} {
		targetFile := src.File + df
		targetRank := src.Rank + dir
		if targetFile < 1 || targetFile > 8 || targetRank < 1 || targetRank > 8 {
			continue
		}
		targetPiece, occupied := getPieceAt(targetFile, targetRank)
		if occupied && targetPiece.Color != ctx.Color {
			legal = append(legal, Square{File: targetFile, Rank: targetRank})
		}
	}

	return legal
}

// getPieceAt - gets the piece at the given file and rank
func getPieceAt(file, rank int) (pieces.ChessPiece, bool) {
	for _, p := range pieces.ChessPieces {
		if p.File == file && p.Rank == rank {
			return p, true
		}
	}

	return pieces.ChessPiece{}, false
}
