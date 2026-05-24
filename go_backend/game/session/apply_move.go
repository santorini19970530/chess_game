// CM3070 FP code
// apply_move.go - applies the movement command to the board

package session

import (
	"fmt"
	"log"

	"go_backend/game/command"
	"go_backend/game/engine"
	pieces "go_backend/game/piece"
)

func ApplyMoveByCommand(commandText string) (string, error) {
	expectedColor := CurrentTurnColor()
	parsed, err := command.ParseCommandForColor(commandText, expectedColor)
	if err != nil {
		return "", err
	}

	fromFile := int(parsed.FromFile - 'a' + 1)
	toFile := int(parsed.ToFile - 'a' + 1)

	moveColor, err := engine.ValidateMove(fromFile, parsed.FromRank, toFile, parsed.ToRank, parsed.PieceCode)
	if err != nil {
		return "", err
	}
	if moveColor != expectedColor {
		return "", fmt.Errorf("wrong turn: expected %s to move", expectedColor)
	}

	if err := ApplyMove(fromFile, parsed.FromRank, toFile, parsed.ToRank); err != nil {
		return "", err
	}
	AppendMoveHistory(parsed.Normalized, moveColor)
	return parsed.Normalized, nil
}

func ApplyMove(fromFile, fromRank, toFile, toRank int) error {
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
	if targetIdx != -1 {
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
