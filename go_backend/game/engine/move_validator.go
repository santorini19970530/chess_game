// CM3070 FP code
// move_validator.go - validates the movements of pieces

package engine

import (
	"fmt"

	"go_backend/game/command"
	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// ValidateMove checks command-piece consistency and basic board legality
func ValidateMove(fromFile, fromRank, toFile, toRank int, pieceCode string) (pieces.PieceColor, error) {
	sourceIdx := -1
	targetIdx := -1

	for i := range pieces.ChessPieces {
		p := &pieces.ChessPieces[i]
		if p.File == fromFile && p.Rank == fromRank {
			sourceIdx = i
		}
		if p.File == toFile && p.Rank == toRank {
			targetIdx = i
		}
	}

	if sourceIdx == -1 {
		return "", fmt.Errorf("There is no piece at source square")
	}

	sourcePiece := pieces.ChessPieces[sourceIdx]
	if pieceCode != "" {
		expectedKind, ok := command.CommandPieceMap[pieceCode]
		if ok && sourcePiece.Kind != expectedKind {
			return "", fmt.Errorf("Piece code does not match source piece")
		}
	}

	if targetIdx != -1 {
		targetPiece := pieces.ChessPieces[targetIdx]
		if targetPiece.Color == sourcePiece.Color {
			return "", fmt.Errorf("Cannot capture own piece")
		}
	}

	// Validate by PieceMovementStrategy registry dispatch.
	if err := movement.ValidateMoveByStrategy(
		sourcePiece.Kind,
		fromFile,
		fromRank,
		toFile,
		toRank,
		sourcePiece.Color,
	); err != nil {
		return "", err
	}

	return sourcePiece.Color, nil
}
