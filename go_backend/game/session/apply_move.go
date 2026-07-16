// CM3070 FP code
// apply_move.go - applies the movement command to the board

package session

import (
	"fmt"
	"log"
	"strings"
	"time"

	"go_backend/game/command"
	"go_backend/game/engine"
	pieces "go_backend/game/piece"
)

func ApplyMoveByCommand(commandText string) (string, error) {
	game, err := lockActiveRuntimeState()
	if err != nil {
		return "", err
	}
	defer unlockActiveRuntimeState(game)
	if game.Session.Type == GameTypeXiangqi {
		normalized, err := applyXiangqiUCIMove(commandText)
		if err != nil {
			return "", err
		}
		game.Session.Outcome = GameOutcome{Status: "in_progress", Message: "in progress"}
		game.Session.Result = GameResultInProgress
		game.Session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return normalized, nil
	}
	return applyMoveByCommandCurrentLoaded(commandText)
}

func applyMoveByCommandCurrentLoaded(commandText string) (string, error) {
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
	targetPiece, destinationOccupied := getPieceAt(toFile, parsed.ToRank)
	capturedKind := pieces.PieceKind("")
	if destinationOccupied {
		capturedKind = targetPiece.Kind
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
				capturedKind = pieces.Pawn
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
	if castling && castlingViolatesCheckRules(moveColor, parsed.FromRank, toFile) {
		return "", fmt.Errorf("illegal move: castling through check is not allowed")
	}
	if engine.WouldLeaveKingInCheck(
		sourcePiece,
		fromFile, parsed.FromRank,
		toFile, parsed.ToRank,
		enPassant, castling,
		requiresPromotion, promotionKind,
	) {
		return "", fmt.Errorf("illegal move: king would remain in check")
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
	displayPieceKind := sourcePiece.Kind
	if requiresPromotion {
		displayPieceKind = promotionKind
	}
	captureOccurred := enPassant || destinationOccupied
	AppendMoveHistory(parsed.Normalized, moveColor, displayPieceKind, toFile, parsed.ToRank, captureOccurred, capturedKind)
	SetCurrentTurnColor(OpponentColor(moveColor))
	RecordLastMove(fromFile, parsed.FromRank, toFile, parsed.ToRank, sourcePiece.Kind, moveColor)
	recordDrawStateAfterMove(sourcePiece.Kind, captureOccurred)
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
	log.Printf("move applied: %s %s", strings.ToLower(string(sourcePiece.Color)), toUCI(fromFile, fromRank, toFile, toRank))
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
	log.Printf("move applied: %s %s", strings.ToLower(string(sourcePiece.Color)), toUCI(fromFile, fromRank, toFile, toRank))
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
	log.Printf("move applied: %s %s (castle)", strings.ToLower(string(pieces.ChessPieces[kingIdx].Color)), toUCI(fromFile, fromRank, toFile, toRank))
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

func castlingViolatesCheckRules(color pieces.PieceColor, rank, toFile int) bool {
	if engine.IsInCheck(color) {
		return true
	}
	opponent := pieces.White
	if color == pieces.White {
		opponent = pieces.Black
	}
	throughFile := 6
	if toFile == 3 {
		throughFile = 4
	}
	if engine.IsSquareAttackedBy(throughFile, rank, opponent) {
		return true
	}
	if engine.IsSquareAttackedBy(toFile, rank, opponent) {
		return true
	}
	return false
}

// toUCI converts internal file/rank to UCI square notation (chess a-h/1-8, xiangqi a-i/1-10).
func toUCI(fromFile, fromRank, toFile, toRank int) string {
	maxFile, maxRank := 8, 8
	if fromFile > 8 || toFile > 8 || fromRank > 8 || toRank > 8 {
		maxFile, maxRank = 9, 10
	}
	if fromFile < 1 || fromFile > maxFile || toFile < 1 || toFile > maxFile ||
		fromRank < 1 || fromRank > maxRank || toRank < 1 || toRank > maxRank {
		return fmt.Sprintf("%d%d->%d%d", fromFile, fromRank, toFile, toRank)
	}
	return fmt.Sprintf("%c%d%c%d",
		'a'+byte(fromFile-1), fromRank,
		'a'+byte(toFile-1), toRank)
}
