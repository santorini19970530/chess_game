package command

import (
	"fmt"
	pieces "go_backend/game/piece"
	"log"
	"regexp"
	"strings"
)

var commandFormatPattern = regexp.MustCompile(`^(?:[a-h][1-8][a-h][1-8][qrbn]?|[prnbqk][a-h][1-8][a-h][1-8])$`)
var sanPattern = regexp.MustCompile(`^([pkqrbn])?([a-h1-8]{0,2})(x)?([a-h][1-8])(?:=?([qrbn]))?$`)

type castleMove struct {
	fromFile int
	fromRank int
	toFile   int
	toRank   int
}

func ParseCommand(command string) (ParsedCommand, error) {
	return ParseCommandForColor(command, "")
}

func ParseCommandForColor(command string, expectedColor pieces.PieceColor) (ParsedCommand, error) {
	command = strings.ToLower(strings.TrimSpace(command))
	if !commandFormatPattern.MatchString(command) {
		parsedSAN, err := parseSANCommand(command, expectedColor)
		if err != nil {
			return ParsedCommand{}, err
		}
		return parsedSAN, nil
	}

	parsed := ParsedCommand{
		Raw:    command,
		Format: "uci",
	}

	if command[1] >= '1' && command[1] <= '8' {
		parsed.FromFile = command[0]
		parsed.FromRank = int(command[1] - '0')
		parsed.ToFile = command[2]
		parsed.ToRank = int(command[3] - '0')
		if len(command) == 5 {
			parsed.Promotion = string(command[4])
		}
	} else {
		parsed.Format = "piece-prefixed"
		parsed.PieceCode = string(command[0])
		parsed.FromFile = command[1]
		parsed.FromRank = int(command[2] - '0')
		parsed.ToFile = command[3]
		parsed.ToRank = int(command[4] - '0')
	}

	parsed.Normalized = fmt.Sprintf(
		"%c%d%c%d%s",
		parsed.FromFile,
		parsed.FromRank,
		parsed.ToFile,
		parsed.ToRank,
		parsed.Promotion,
	)

	return parsed, nil
}

func parseSANCommand(command string, expectedColor pieces.PieceColor) (ParsedCommand, error) {
	san := strings.TrimRight(command, "+#!?")

	if san == "o-o" || san == "0-0" {
		return parseCastleSAN(command, false, expectedColor)
	}
	if san == "o-o-o" || san == "0-0-0" {
		return parseCastleSAN(command, true, expectedColor)
	}

	matches := sanPattern.FindStringSubmatch(san)
	if matches == nil {
		return ParsedCommand{}, fmt.Errorf("invalid san")
	}

	pieceLetter := matches[1] // optional
	disamb := matches[2]
	isCapture := matches[3] == "x"
	toSquare := matches[4]
	promotion := matches[5]

	targetFile := int(toSquare[0]-'a') + 1
	targetRank := int(toSquare[1] - '0')

	targetKind := pieces.Pawn
	if pieceLetter != "" {
		targetKind = kindFromSANPiece(pieceLetter[0])
	}

	fileHint := 0
	rankHint := 0
	if len(disamb) == 1 {
		if disamb[0] >= 'a' && disamb[0] <= 'h' {
			fileHint = int(disamb[0]-'a') + 1
		} else if disamb[0] >= '1' && disamb[0] <= '8' {
			rankHint = int(disamb[0] - '0')
		}
	} else if len(disamb) == 2 {
		if disamb[0] >= 'a' && disamb[0] <= 'h' {
			fileHint = int(disamb[0]-'a') + 1
		}
		if disamb[1] >= '1' && disamb[1] <= '8' {
			rankHint = int(disamb[1] - '0')
		}
	}

	fromFile, fromRank, candidateCount := findSANCandidate(
		targetKind,
		targetFile,
		targetRank,
		isCapture,
		fileHint,
		rankHint,
		expectedColor,
	)

	if candidateCount == 0 {
		return ParsedCommand{}, fmt.Errorf("no san source piece found")
	}
	if candidateCount > 1 {
		return ParsedCommand{}, fmt.Errorf("ambiguous san move")
	}

	parsed := ParsedCommand{
		Raw:       command,
		Format:    "san",
		PieceCode: pieceCodeFromKind(targetKind),
		FromFile:  byte('a' + fromFile - 1),
		FromRank:  fromRank,
		ToFile:    byte('a' + targetFile - 1),
		ToRank:    targetRank,
		Promotion: promotion,
	}
	parsed.Normalized = fmt.Sprintf("%c%d%c%d%s", parsed.FromFile, parsed.FromRank, parsed.ToFile, parsed.ToRank, parsed.Promotion)
	return parsed, nil
}

func parseCastleSAN(raw string, queenSide bool, expectedColor pieces.PieceColor) (ParsedCommand, error) {
	moves := []castleMove{
		{fromFile: 5, fromRank: 1, toFile: 7, toRank: 1}, // white king side
		{fromFile: 5, fromRank: 8, toFile: 7, toRank: 8}, // black king side
	}
	if queenSide {
		moves = []castleMove{
			{fromFile: 5, fromRank: 1, toFile: 3, toRank: 1}, // white queen side
			{fromFile: 5, fromRank: 8, toFile: 3, toRank: 8}, // black queen side
		}
	}

	selected, candidateCount := findCastleCandidate(moves, expectedColor)

	if candidateCount == 0 {
		return ParsedCommand{}, fmt.Errorf("cannot castle in current position")
	}
	if candidateCount > 1 {
		return ParsedCommand{}, fmt.Errorf("ambiguous castling side")
	}

	parsed := ParsedCommand{
		Raw:       raw,
		Format:    "san",
		PieceCode: "k",
		FromFile:  byte('a' + selected.fromFile - 1),
		FromRank:  selected.fromRank,
		ToFile:    byte('a' + selected.toFile - 1),
		ToRank:    selected.toRank,
	}
	parsed.Normalized = fmt.Sprintf("%c%d%c%d", parsed.FromFile, parsed.FromRank, parsed.ToFile, parsed.ToRank)
	return parsed, nil
}

func kindFromSANPiece(ch byte) pieces.PieceKind {
	switch ch {
	case 'p':
		return pieces.Pawn
	case 'k':
		return pieces.King
	case 'q':
		return pieces.Queen
	case 'r':
		return pieces.Rook
	case 'b':
		return pieces.Bishop
	case 'n':
		return pieces.Knight
	default:
		return pieces.Pawn
	}
}

func pieceCodeFromKind(kind pieces.PieceKind) string {
	switch kind {
	case pieces.King:
		return "k"
	case pieces.Queen:
		return "q"
	case pieces.Rook:
		return "r"
	case pieces.Bishop:
		return "b"
	case pieces.Knight:
		return "n"
	default:
		return "p"
	}
}

func getPieceAt(file, rank int) (pieces.ChessPiece, bool) {
	for _, p := range pieces.ChessPieces {
		if p.File == file && p.Rank == rank {
			return p, true
		}
	}
	return pieces.ChessPiece{}, false
}

func canPieceReach(p pieces.ChessPiece, toFile, toRank int, isCapture bool) bool {
	df := toFile - p.File
	dr := toRank - p.Rank

	targetPiece, targetOccupied := getPieceAt(toFile, toRank)
	if isCapture && (!targetOccupied || targetPiece.Color == p.Color) {
		return false
	}
	if !isCapture && targetOccupied {
		return false
	}

	switch p.Kind {
	case pieces.Knight:
		absDf := df
		if absDf < 0 {
			absDf = -absDf
		}
		absDr := dr
		if absDr < 0 {
			absDr = -absDr
		}
		return (absDf == 1 && absDr == 2) || (absDf == 2 && absDr == 1)
	case pieces.Bishop:
		if abs(df) != abs(dr) {
			return false
		}
		return pathClear(p.File, p.Rank, toFile, toRank)
	case pieces.Rook:
		if df != 0 && dr != 0 {
			return false
		}
		return pathClear(p.File, p.Rank, toFile, toRank)
	case pieces.Queen:
		if df == 0 || dr == 0 || abs(df) == abs(dr) {
			return pathClear(p.File, p.Rank, toFile, toRank)
		}
		return false
	case pieces.King:
		return abs(df) <= 1 && abs(dr) <= 1
	case pieces.Pawn:
		dir := 1
		startRank := 2
		if p.Color == pieces.Black {
			dir = -1
			startRank = 7
		}
		if isCapture {
			return dr == dir && abs(df) == 1
		}
		if df != 0 {
			return false
		}
		if dr == dir {
			return !targetOccupied
		}
		if p.Rank == startRank && dr == 2*dir {
			midRank := p.Rank + dir
			_, midOccupied := getPieceAt(p.File, midRank)
			return !midOccupied && !targetOccupied
		}
		return false
	default:
		return false
	}
}

func pathClear(fromFile, fromRank, toFile, toRank int) bool {
	stepFile := sign(toFile - fromFile)
	stepRank := sign(toRank - fromRank)

	f := fromFile + stepFile
	r := fromRank + stepRank
	for f != toFile || r != toRank {
		if _, occupied := getPieceAt(f, r); occupied {
			return false
		}
		f += stepFile
		r += stepRank
	}
	return true
}

func sign(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func findSANCandidate(
	targetKind pieces.PieceKind,
	targetFile, targetRank int,
	isCapture bool,
	fileHint, rankHint int,
	expectedColor pieces.PieceColor,
) (int, int, int) {
	var fromFile, fromRank int
	candidateCount := 0

	for _, p := range pieces.ChessPieces {
		if expectedColor != "" && p.Color != expectedColor {
			continue
		}
		if p.Kind != targetKind {
			continue
		}
		if fileHint != 0 && p.File != fileHint {
			continue
		}
		if rankHint != 0 && p.Rank != rankHint {
			continue
		}
		if !canPieceReach(p, targetFile, targetRank, isCapture) {
			continue
		}
		candidateCount++
		fromFile = p.File
		fromRank = p.Rank
	}

	return fromFile, fromRank, candidateCount
}

func findCastleCandidate(moves []castleMove, expectedColor pieces.PieceColor) (castleMove, int) {
	candidateCount := 0
	var selected castleMove

	for _, mv := range moves {
		piece, found := getPieceAt(mv.fromFile, mv.fromRank)
		if !found || piece.Kind != pieces.King {
			continue
		}
		if expectedColor != "" && piece.Color != expectedColor {
			continue
		}
		if _, occupied := getPieceAt(mv.toFile, mv.toRank); occupied {
			continue
		}
		candidateCount++
		selected = mv
	}

	return selected, candidateCount
}

func ParseAndLogCommand(command string) error {
	return ParseAndLogCommandForColor(command, "")
}

func ParseAndLogCommandForColor(command string, expectedColor pieces.PieceColor) error {
	parsed, err := ParseCommandForColor(command, expectedColor)
	if err != nil {
		return err
	}

	fromFileInt := int(parsed.FromFile - 'a' + 1)
	toFileInt := int(parsed.ToFile - 'a' + 1)

	log.Printf(
		"command parsed: raw=%q format=%s piece=%q from=%c%d(file=%d,rank=%d) to=%c%d(file=%d,rank=%d) promotion=%q",
		parsed.Raw,
		parsed.Format,
		parsed.PieceCode,
		parsed.FromFile,
		parsed.FromRank,
		fromFileInt,
		parsed.FromRank,
		parsed.ToFile,
		parsed.ToRank,
		toFileInt,
		parsed.ToRank,
		parsed.Promotion,
	)
	log.Printf("command normalized: %q", parsed.Normalized)

	return nil
}
