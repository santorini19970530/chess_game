package session

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

// Board: a1a2, a8a9+ (optional promote). Drop: P*e5 (piece in hand → empty square).
var (
	shogiBoardMovePattern = regexp.MustCompile(`^[a-i][1-9][a-i][1-9]\+?$`)
	shogiDropMovePattern  = regexp.MustCompile(`^[plnsgbrPLNSGBR][*@][a-iA-I][1-9]$`)
)

func applyShogiUCIMove(commandText string) (string, error) {
	raw := strings.TrimSpace(commandText)
	if shogiDropMovePattern.MatchString(raw) {
		return applyShogiDrop(raw)
	}
	move := strings.ToLower(raw)
	if !shogiBoardMovePattern.MatchString(move) {
		return "", fmt.Errorf("invalid shogi move %q (board e.g. c3c4 / c3c4+, drop e.g. P*e5)", commandText)
	}
	return applyShogiBoardMove(move)
}

func applyShogiBoardMove(move string) (string, error) {
	promote := strings.HasSuffix(move, "+")
	core := strings.TrimSuffix(move, "+")
	fromFile := int(core[0] - 'a' + 1)
	fromRank := int(core[1] - '0')
	toFile := int(core[2] - 'a' + 1)
	toRank := int(core[3] - '0')

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

	if err := movement.ValidateShogiMoveByStrategy(
		sourcePiece.Kind, fromFile, fromRank, toFile, toRank, sourcePiece.Color,
	); err != nil {
		return "", err
	}
	if movement.ShogiWouldLeaveKingInCheck(sourcePiece, fromFile, fromRank, toFile, toRank) {
		return "", fmt.Errorf("illegal move: king would be in check")
	}

	canPromote := movement.ShogiCanPromote(sourcePiece.Kind, fromRank, toRank, sourcePiece.Color)
	mustPromote := movement.ShogiMustPromote(sourcePiece.Kind, toRank, sourcePiece.Color)
	if promote && !canPromote {
		return "", fmt.Errorf("illegal promotion")
	}
	doPromote := mustPromote || (promote && canPromote)

	capturedKind := pieces.PieceKind("")
	if destinationOccupied {
		capturedKind = targetPiece.Kind
	}
	if err := ApplyMove(fromFile, fromRank, toFile, toRank); err != nil {
		return "", err
	}
	if destinationOccupied {
		shogiAddToHand(sourcePiece.Color, capturedKind)
	}
	if doPromote {
		if err := shogiPromotePieceAt(toFile, toRank); err != nil {
			return "", err
		}
		move = core + "+"
	} else {
		move = core
	}

	AppendMoveHistory(move, sourcePiece.Color, sourcePiece.Kind, toFile, toRank, destinationOccupied, shogiUnpromoteForHand(capturedKind))
	RecordLastMove(fromFile, fromRank, toFile, toRank, sourcePiece.Kind, sourcePiece.Color)
	SetCurrentTurnColor(OpponentColor(sourcePiece.Color))
	syncShogiBoardFEN()
	return move, nil
}

func applyShogiDrop(raw string) (string, error) {
	piece := strings.ToUpper(raw[:1])
	sq := strings.ToLower(raw[2:])
	move := piece + "*" + sq
	kind, ok := shogiDropKindFromChar(rune(piece[0]))
	if !ok {
		return "", fmt.Errorf("invalid drop piece")
	}
	toFile := int(sq[0] - 'a' + 1)
	toRank := int(sq[1] - '0')

	color := CurrentTurnColor()
	if !shogiTakeFromHand(color, kind) {
		return "", fmt.Errorf("no %s in hand to drop", kind)
	}
	if err := validateShogiDrop(kind, color, toFile, toRank); err != nil {
		shogiAddToHand(color, kind)
		return "", err
	}

	pieces.ChessPieces = append(pieces.ChessPieces, pieces.ChessPiece{
		Color: color,
		Kind:  kind,
		File:  toFile,
		Rank:  toRank,
	})
	log.Printf("move applied: %s %s", strings.ToLower(string(color)), move)
	AppendMoveHistory(move, color, kind, toFile, toRank, false, "")
	RecordLastMove(0, 0, toFile, toRank, kind, color)
	SetCurrentTurnColor(OpponentColor(color))
	syncShogiBoardFEN()
	return move, nil
}

func validateShogiDrop(kind pieces.PieceKind, color pieces.PieceColor, file, rank int) error {
	if kind == pieces.King {
		return fmt.Errorf("cannot drop king")
	}
	if _, occupied := getPieceAt(file, rank); occupied {
		return fmt.Errorf("cannot drop on occupied square")
	}
	if movement.ShogiMustPromote(kind, rank, color) {
		// pawn/lance/knight cannot drop where they would have no legal move
		return fmt.Errorf("illegal drop square for %s", kind)
	}
	if kind == pieces.Pawn && shogiHasUnpromotedPawnOnFile(color, file) {
		return fmt.Errorf("nifu: two unpromoted pawns on the same file")
	}
	// MVP: uchifuzume (pawn-drop mate) not enforced yet.
	if movement.ShogiWouldLeaveKingInCheckAfterDrop(kind, color, file, rank) {
		return fmt.Errorf("illegal drop: king would be in check")
	}
	return nil
}

func shogiPromotePieceAt(file, rank int) error {
	for i := range pieces.ChessPieces {
		p := &pieces.ChessPieces[i]
		if p.File != file || p.Rank != rank {
			continue
		}
		promoted, ok := movement.ShogiPromotedKind(p.Kind)
		if !ok {
			return fmt.Errorf("piece cannot promote")
		}
		p.Kind = promoted
		return nil
	}
	return fmt.Errorf("piece to promote not found")
}

func shogiDropKindFromChar(ch rune) (pieces.PieceKind, bool) {
	switch ch {
	case 'P':
		return pieces.Pawn, true
	case 'L':
		return pieces.Lance, true
	case 'N':
		return pieces.Knight, true
	case 'S':
		return pieces.Silver, true
	case 'G':
		return pieces.Gold, true
	case 'B':
		return pieces.Bishop, true
	case 'R':
		return pieces.Rook, true
	default:
		return "", false
	}
}

func shogiDropChar(kind pieces.PieceKind) (byte, bool) {
	switch kind {
	case pieces.Pawn:
		return 'P', true
	case pieces.Lance:
		return 'L', true
	case pieces.Knight:
		return 'N', true
	case pieces.Silver:
		return 'S', true
	case pieces.Gold:
		return 'G', true
	case pieces.Bishop:
		return 'B', true
	case pieces.Rook:
		return 'R', true
	default:
		return 0, false
	}
}
