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
	sourcePiece, found := getPieceAt(fromFile, parsed.FromRank)
	if !found {
		return "", fmt.Errorf("There is no piece at source square")
	}

	moveColor, err := engine.ValidateMove(fromFile, parsed.FromRank, toFile, parsed.ToRank, parsed.PieceCode)
	enPassant := false
	if err != nil {
		_, destinationOccupied := getPieceAt(toFile, parsed.ToRank)
		adjacentPawn, adjacentPawnFound := getPieceAt(toFile, parsed.FromRank)
		lastMove := toEngineLastMove(GetLastMove())
		if sourcePiece.Kind == pieces.Pawn && engine.CanEnPassant(
			sourcePiece,
			fromFile, parsed.FromRank,
			toFile, parsed.ToRank,
			destinationOccupied,
			lastMove,
			adjacentPawn,
			adjacentPawnFound,
		) {
			moveColor = sourcePiece.Color
			enPassant = true
		} else {
			return "", err
		}
	}
	if moveColor != expectedColor {
		return "", fmt.Errorf("wrong turn: expected %s to move", expectedColor)
	}

	if enPassant {
		if err := ApplyEnPassantMove(fromFile, parsed.FromRank, toFile, parsed.ToRank); err != nil {
			return "", err
		}
	} else {
		if err := ApplyMove(fromFile, parsed.FromRank, toFile, parsed.ToRank); err != nil {
			return "", err
		}
	}
	AppendMoveHistory(parsed.Normalized, moveColor)
	RecordLastMove(fromFile, parsed.FromRank, toFile, parsed.ToRank, sourcePiece.Kind, moveColor)
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

func ApplyEnPassantMove(fromFile, fromRank, toFile, toRank int) error {
	sourceIdx := -1
	capturedIdx := -1
	capturedRank := fromRank

	for i := range pieces.ChessPieces {
		p := &pieces.ChessPieces[i]
		if p.File == fromFile && p.Rank == fromRank {
			sourceIdx = i
		}
		if p.File == toFile && p.Rank == capturedRank {
			capturedIdx = i
		}
	}

	if sourceIdx == -1 {
		return fmt.Errorf("There is no piece at source square")
	}
	if capturedIdx == -1 {
		return fmt.Errorf("No en passant target pawn found")
	}

	sourcePiece := &pieces.ChessPieces[sourceIdx]
	pieces.ChessPieces = append(pieces.ChessPieces[:capturedIdx], pieces.ChessPieces[capturedIdx+1:]...)
	if capturedIdx < sourceIdx {
		sourceIdx--
	}
	sourcePiece = &pieces.ChessPieces[sourceIdx]
	sourcePiece.Move(toFile, toRank)
	log.Printf("en passant applied: %d%d -> %d%d", fromFile, fromRank, toFile, toRank)
	return nil
}

func getPieceAt(file, rank int) (pieces.ChessPiece, bool) {
	for _, p := range pieces.ChessPieces {
		if p.File == file && p.Rank == rank {
			return p, true
		}
	}
	return pieces.ChessPiece{}, false
}

func toEngineLastMove(mv *LastMove) *engine.LastMoveInfo {
	if mv == nil {
		return nil
	}
	return &engine.LastMoveInfo{
		FromFile:       mv.FromFile,
		FromRank:       mv.FromRank,
		ToFile:         mv.ToFile,
		ToRank:         mv.ToRank,
		PieceKind:      mv.PieceKind,
		Color:          mv.Color,
		PawnDoubleStep: mv.PawnDoubleStep,
	}
}
