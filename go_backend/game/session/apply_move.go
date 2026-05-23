// CM3070 FP code
// apply_move.go - applies the movement command to the board

package session

import (
	"fmt"
	"log"

	"go_backend/game/command"
	pieces "go_backend/game/piece"
)

func ApplyMoveByCommand(commandText string) error {
	parsed, err := command.ParseCommand(commandText)
	if err != nil {
		return err
	}

	fromFile := int(parsed.FromFile - 'a' + 1)
	toFile := int(parsed.ToFile - 'a' + 1)

	sourceIdx := -1
	for i := range pieces.ChessPieces {
		p := &pieces.ChessPieces[i]
		if p.File == fromFile && p.Rank == parsed.FromRank {
			sourceIdx = i
			break
		}
	}
	if sourceIdx == -1 {
		return fmt.Errorf("There is no piece at source square")
	}
	moveColor := pieces.ChessPieces[sourceIdx].Color

	if err := ApplyMove(fromFile, parsed.FromRank, toFile, parsed.ToRank, parsed.PieceCode); err != nil {
		return err
	}
	AppendMoveHistory(commandText, moveColor)
	return nil
}

func ApplyMove(fromFile, fromRank, toFile, toRank int, pieceCode string) error {
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
		return fmt.Errorf("There is no piece at source square")
	}

	sourcePiece := &pieces.ChessPieces[sourceIdx]
	if pieceCode != "" {
		expectedKind, ok := command.CommandPieceMap[pieceCode]
		if ok && sourcePiece.Kind != expectedKind {
			return fmt.Errorf("Piece code does not match source piece")
		}
	}

	if targetIdx != -1 {
		targetPiece := pieces.ChessPieces[targetIdx]
		if targetPiece.Color == sourcePiece.Color {
			return fmt.Errorf("Cannot capture own piece")
		}

		pieces.ChessPieces = append(pieces.ChessPieces[:targetIdx], pieces.ChessPieces[targetIdx+1:]...)
		if targetIdx < sourceIdx {
			sourceIdx--
		}
		sourcePiece = &pieces.ChessPieces[sourceIdx]
	}

	sourcePiece.Move(toFile, toRank)
	log.Printf("move applied: %d%d -> %d%d", fromFile, fromRank, toFile, toRank)
	return nil
}
