package session

import (
	"fmt"
	"regexp"
	"strings"

	"go_backend/game/engine"
	pieces "go_backend/game/piece"
)

// Fairy-Stockfish Xiangqi UCI: files a-i, ranks 1-10 (e.g. a4a5, b3b10, a10a9).
var xiangqiUCIMovePattern = regexp.MustCompile(`^[a-i](?:10|[1-9])[a-i](?:10|[1-9])$`)

func applyXiangqiUCIMove(commandText string) (string, error) {
	move := strings.ToLower(strings.TrimSpace(commandText))
	if !xiangqiUCIMovePattern.MatchString(move) {
		return "", fmt.Errorf("invalid xiangqi move %q (expected FS UCI, e.g. a4a5 or h3h10)", commandText)
	}
	fen := boardFEN
	if fen == "" {
		fen = DefaultXiangqiStartFEN
	}

	legal, err := xiangqiAllLegalUCIMoves()
	if err != nil {
		return "", fmt.Errorf("list legal moves: %w", err)
	}
	ok := false
	for _, mv := range legal {
		if mv == move {
			ok = true
			break
		}
	}
	if !ok {
		return "", fmt.Errorf("illegal move: %s", move)
	}

	fs, err := engine.RulesEngine()
	if err != nil {
		return "", fmt.Errorf("fairy-stockfish unavailable: %w", err)
	}
	newFEN, err := fs.FENAfterMove(fen, move)
	if err != nil {
		return "", fmt.Errorf("apply move on engine: %w", err)
	}

	fromFile, fromRank, toFile, toRank, err := parseXiangqiUCISquares(move)
	if err != nil {
		return "", err
	}
	sourcePiece, found := getPieceAt(fromFile, fromRank)
	if !found {
		return "", fmt.Errorf("There is no piece at source square")
	}
	_, destOccupied := getPieceAt(toFile, toRank)
	capturedKind := pieces.PieceKind("")
	if destOccupied {
		if target, ok := getPieceAt(toFile, toRank); ok {
			capturedKind = target.Kind
		}
	}

	if err := applyXiangqiFENToCurrentGlobals(newFEN); err != nil {
		return "", err
	}
	AppendMoveHistory(move, sourcePiece.Color, sourcePiece.Kind, toFile, toRank, destOccupied, capturedKind)
	RecordLastMove(fromFile, fromRank, toFile, toRank, sourcePiece.Kind, sourcePiece.Color)
	return move, nil
}

func parseXiangqiUCISquares(move string) (fromFile, fromRank, toFile, toRank int, err error) {
	// [file][rank][file][rank] with ranks 1-10
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
