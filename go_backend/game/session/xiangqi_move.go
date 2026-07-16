package session

import (
	"fmt"
	"regexp"
	"strings"

	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// Fairy-Stockfish / API UCI for Xiangqi: files a-i, ranks 1-10 (e.g. a4a5, b3b10, a10a9).
// Move legality is enforced by Go strategies, not Fairy-Stockfish.
var xiangqiUCIMovePattern = regexp.MustCompile(`^[a-i](?:10|[1-9])[a-i](?:10|[1-9])$`)

func applyXiangqiUCIMove(commandText string) (string, error) {
	move := strings.ToLower(strings.TrimSpace(commandText))
	if !xiangqiUCIMovePattern.MatchString(move) {
		return "", fmt.Errorf("invalid xiangqi move %q (expected UCI, e.g. a4a5 or h3h10)", commandText)
	}
	fromFile, fromRank, toFile, toRank, err := parseXiangqiUCISquares(move)
	if err != nil {
		return "", err
	}

	expectedColor := CurrentTurnColor()
	sourcePiece, found := getPieceAt(fromFile, fromRank)
	if !found {
		return "", fmt.Errorf("There is no piece at source square")
	}
	if sourcePiece.Color != expectedColor {
		return "", fmt.Errorf("It is not %s's turn", expectedColor)
	}
	targetPiece, destinationOccupied := getPieceAt(toFile, toRank)
	if destinationOccupied && targetPiece.Color == sourcePiece.Color {
		return "", fmt.Errorf("Cannot capture own piece")
	}
	if destinationOccupied && targetPiece.Kind == pieces.King {
		return "", fmt.Errorf("Cannot capture the king")
	}

	if err := movement.ValidateXiangqiMoveByStrategy(
		sourcePiece.Kind, fromFile, fromRank, toFile, toRank, sourcePiece.Color,
	); err != nil {
		return "", err
	}
	if movement.XiangqiWouldLeaveGeneralInCheck(sourcePiece, fromFile, fromRank, toFile, toRank) {
		return "", fmt.Errorf("illegal move: general would be in check")
	}

	capturedKind := pieces.PieceKind("")
	if destinationOccupied {
		capturedKind = targetPiece.Kind
	}
	if err := ApplyMove(fromFile, fromRank, toFile, toRank); err != nil {
		return "", err
	}
	AppendMoveHistory(move, sourcePiece.Color, sourcePiece.Kind, toFile, toRank, destinationOccupied, capturedKind)
	RecordLastMove(fromFile, fromRank, toFile, toRank, sourcePiece.Kind, sourcePiece.Color)
	SetCurrentTurnColor(OpponentColor(sourcePiece.Color))
	syncXiangqiBoardFEN()
	return move, nil
}

func parseXiangqiUCISquares(move string) (fromFile, fromRank, toFile, toRank int, err error) {
	if len(move) < 4 {
		return 0, 0, 0, 0, fmt.Errorf("invalid move")
	}
	fromFile = int(move[0] - 'a' + 1)
	i := 1
	for i < len(move) && move[i] >= '0' && move[i] <= '9' {
		i++
	}
	fromRank = atoiDec(move[1:i])
	if i >= len(move) {
		return 0, 0, 0, 0, fmt.Errorf("invalid move")
	}
	toFile = int(move[i] - 'a' + 1)
	toRank = atoiDec(move[i+1:])
	if fromFile < 1 || fromFile > 9 || toFile < 1 || toFile > 9 || fromRank < 1 || fromRank > 10 || toRank < 1 || toRank > 10 {
		return 0, 0, 0, 0, fmt.Errorf("move squares out of range")
	}
	return fromFile, fromRank, toFile, toRank, nil
}

func atoiDec(s string) int {
	n := 0
	for _, ch := range s {
		n = n*10 + int(ch-'0')
	}
	return n
}
