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
	castling := false
	if err != nil {
		kingSide := toFile == 7
		queenSide := toFile == 3
		if sourcePiece.Kind == pieces.King && (kingSide || queenSide) && engine.CanCastle(
			sourcePiece,
			fromFile, parsed.FromRank,
			toFile, parsed.ToRank,
			CanCastleByState(sourcePiece.Color, kingSide),
		) {
			moveColor = sourcePiece.Color
			castling = true
		} else {
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
	}
	if moveColor != expectedColor {
		return "", fmt.Errorf("wrong turn: expected %s to move", expectedColor)
	}
	requiresPromotion, promotionKind, err := resolvePromotion(sourcePiece, parsed.ToRank, parsed.Promotion)
	if err != nil {
		return "", err
	}

	if enPassant {
		if err := ApplyEnPassantMove(fromFile, parsed.FromRank, toFile, parsed.ToRank); err != nil {
			return "", err
		}
	} else if castling {
		if err := ApplyCastlingMove(fromFile, parsed.FromRank, toFile, parsed.ToRank); err != nil {
			return "", err
		}
	} else {
		if err := ApplyMove(fromFile, parsed.FromRank, toFile, parsed.ToRank); err != nil {
			return "", err
		}
	}
	if requiresPromotion {
		if err := ApplyPromotion(toFile, parsed.ToRank, moveColor, promotionKind); err != nil {
			return "", err
		}
	}
	RecordPieceMoveForCastling(sourcePiece.Kind, moveColor, fromFile, parsed.FromRank)
	if castling {
		rookFromFile := 1
		if toFile == 7 {
			rookFromFile = 8
		}
		RecordPieceMoveForCastling(pieces.Rook, moveColor, rookFromFile, parsed.FromRank)
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

func ApplyCastlingMove(fromFile, fromRank, toFile, toRank int) error {
	kingIdx := -1
	rookIdx := -1
	rookFromFile := 1
	rookToFile := 4
	if toFile == 7 {
		rookFromFile = 8
		rookToFile = 6
	}

	for i := range pieces.ChessPieces {
		p := &pieces.ChessPieces[i]
		if p.File == fromFile && p.Rank == fromRank {
			kingIdx = i
		}
		if p.File == rookFromFile && p.Rank == fromRank {
			rookIdx = i
		}
	}
	if kingIdx == -1 {
		return fmt.Errorf("king not found for castling")
	}
	if rookIdx == -1 {
		return fmt.Errorf("rook not found for castling")
	}

	pieces.ChessPieces[kingIdx].Move(toFile, toRank)
	pieces.ChessPieces[rookIdx].Move(rookToFile, fromRank)
	log.Printf("castling applied: king %d%d -> %d%d, rook %d%d -> %d%d",
		fromFile, fromRank, toFile, toRank, rookFromFile, fromRank, rookToFile, fromRank)
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

func resolvePromotion(source pieces.ChessPiece, toRank int, promotion string) (bool, pieces.PieceKind, error) {
	reachesLastRank := (source.Color == pieces.White && toRank == 8) || (source.Color == pieces.Black && toRank == 1)
	if source.Kind != pieces.Pawn {
		if promotion != "" {
			return false, "", fmt.Errorf("promotion only allowed for pawn")
		}
		return false, "", nil
	}

	if reachesLastRank && promotion == "" {
		return false, "", fmt.Errorf("promotion piece required (q/r/b/n)")
	}
	if !reachesLastRank && promotion != "" {
		return false, "", fmt.Errorf("promotion only allowed when pawn reaches last rank")
	}
	if !reachesLastRank {
		return false, "", nil
	}

	switch promotion {
	case "q":
		return true, pieces.Queen, nil
	case "r":
		return true, pieces.Rook, nil
	case "b":
		return true, pieces.Bishop, nil
	case "n":
		return true, pieces.Knight, nil
	default:
		return false, "", fmt.Errorf("invalid promotion piece (use q/r/b/n)")
	}
}

func ApplyPromotion(file, rank int, color pieces.PieceColor, promotedKind pieces.PieceKind) error {
	for i := range pieces.ChessPieces {
		p := &pieces.ChessPieces[i]
		if p.File == file && p.Rank == rank && p.Color == color {
			p.Kind = promotedKind
			p.ImgFile = promotedPieceImage(promotedKind, color)
			return nil
		}
	}
	return fmt.Errorf("promotion target piece not found")
}

func promotedPieceImage(kind pieces.PieceKind, color pieces.PieceColor) string {
	tone := "light"
	if color == pieces.Black {
		tone = "dark"
	}
	return fmt.Sprintf("pic/chess_pic/%s_%s.png", string(kind), tone)
}
