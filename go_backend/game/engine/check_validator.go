// CM3070 FP code
// check_validator.go - implements check validation rules

package engine

import pieces "go_backend/game/piece"

// IsInCheck reports whether the specified side's king is under attack.
func IsInCheck(color pieces.PieceColor) bool {
	return isInCheckOnBoard(pieces.ChessPieces, color)
}

// IsSquareAttackedBy reports whether an attacker color attacks the square.
func IsSquareAttackedBy(file, rank int, attackerColor pieces.PieceColor) bool {
	return isSquareAttacked(pieces.ChessPieces, file, rank, attackerColor)
}

// CheckedColor returns the side currently in check, if any.
func CheckedColor() pieces.PieceColor {
	if IsInCheck(pieces.White) {
		return pieces.White
	}
	if IsInCheck(pieces.Black) {
		return pieces.Black
	}
	return ""
}

// WouldLeaveKingInCheck reports whether applying a move would leave the mover in check.
func WouldLeaveKingInCheck(
	source pieces.ChessPiece,
	fromFile, fromRank, toFile, toRank int,
	enPassant, castling bool,
	requiresPromotion bool,
	promotionKind pieces.PieceKind,
) bool {
	boardAfter := simulateBoardAfterMove(
		pieces.ChessPieces,
		source,
		fromFile,
		fromRank,
		toFile,
		toRank,
		enPassant,
		castling,
		requiresPromotion,
		promotionKind,
	)
	return isInCheckOnBoard(boardAfter, source.Color)
}

func simulateBoardAfterMove(
	board []pieces.ChessPiece,
	source pieces.ChessPiece,
	fromFile, fromRank, toFile, toRank int,
	enPassant, castling bool,
	requiresPromotion bool,
	promotionKind pieces.PieceKind,
) []pieces.ChessPiece {
	cloned := append([]pieces.ChessPiece(nil), board...)
	sourceIdx := indexOfPiece(cloned, fromFile, fromRank)
	if sourceIdx == -1 {
		return cloned
	}

	if castling {
		cloned[sourceIdx].Move(toFile, toRank)
		rookFromFile := 1
		rookToFile := 4
		if toFile == 7 {
			rookFromFile = 8
			rookToFile = 6
		}
		rookIdx := indexOfPiece(cloned, rookFromFile, fromRank)
		if rookIdx != -1 {
			cloned[rookIdx].Move(rookToFile, fromRank)
		}
		return cloned
	}

	capturedIdx := -1
	if enPassant {
		capturedIdx = indexOfPiece(cloned, toFile, fromRank)
	} else {
		capturedIdx = indexOfPiece(cloned, toFile, toRank)
	}
	if capturedIdx != -1 {
		cloned = append(cloned[:capturedIdx], cloned[capturedIdx+1:]...)
		if capturedIdx < sourceIdx {
			sourceIdx--
		}
	}

	cloned[sourceIdx].Move(toFile, toRank)
	if requiresPromotion {
		cloned[sourceIdx].Kind = promotionKind
	}
	return cloned
}

func isInCheckOnBoard(board []pieces.ChessPiece, color pieces.PieceColor) bool {
	kingFile, kingRank, found := findKing(board, color)
	if !found {
		return false
	}
	attackerColor := pieces.White
	if color == pieces.White {
		attackerColor = pieces.Black
	}
	return isSquareAttacked(board, kingFile, kingRank, attackerColor)
}

func findKing(board []pieces.ChessPiece, color pieces.PieceColor) (int, int, bool) {
	for _, p := range board {
		if p.Color == color && p.Kind == pieces.King {
			return p.File, p.Rank, true
		}
	}
	return 0, 0, false
}

func isSquareAttacked(board []pieces.ChessPiece, targetFile, targetRank int, attackerColor pieces.PieceColor) bool {
	for _, p := range board {
		if p.Color != attackerColor {
			continue
		}
		if pieceAttacksSquare(board, p, targetFile, targetRank) {
			return true
		}
	}
	return false
}

func pieceAttacksSquare(board []pieces.ChessPiece, piece pieces.ChessPiece, targetFile, targetRank int) bool {
	df := targetFile - piece.File
	dr := targetRank - piece.Rank
	absFile := absCheck(df)
	absRank := absCheck(dr)

	switch piece.Kind {
	case pieces.Pawn:
		dir := 1
		if piece.Color == pieces.Black {
			dir = -1
		}
		return dr == dir && absFile == 1
	case pieces.Knight:
		return (absFile == 1 && absRank == 2) || (absFile == 2 && absRank == 1)
	case pieces.Bishop:
		return absFile == absRank && isPathClear(board, piece.File, piece.Rank, targetFile, targetRank)
	case pieces.Rook:
		straight := (df == 0 && dr != 0) || (dr == 0 && df != 0)
		return straight && isPathClear(board, piece.File, piece.Rank, targetFile, targetRank)
	case pieces.Queen:
		diagonal := absFile == absRank
		straight := (df == 0 && dr != 0) || (dr == 0 && df != 0)
		return (diagonal || straight) && isPathClear(board, piece.File, piece.Rank, targetFile, targetRank)
	case pieces.King:
		return absFile <= 1 && absRank <= 1 && !(absFile == 0 && absRank == 0)
	default:
		return false
	}
}

func isPathClear(board []pieces.ChessPiece, fromFile, fromRank, toFile, toRank int) bool {
	stepFile := signInt(toFile - fromFile)
	stepRank := signInt(toRank - fromRank)
	file := fromFile + stepFile
	rank := fromRank + stepRank
	for file != toFile || rank != toRank {
		if _, occupied := pieceAt(board, file, rank); occupied {
			return false
		}
		file += stepFile
		rank += stepRank
	}
	return true
}

func indexOfPiece(board []pieces.ChessPiece, file, rank int) int {
	for i, p := range board {
		if p.File == file && p.Rank == rank {
			return i
		}
	}
	return -1
}

func pieceAt(board []pieces.ChessPiece, file, rank int) (pieces.ChessPiece, bool) {
	for _, p := range board {
		if p.File == file && p.Rank == rank {
			return p, true
		}
	}
	return pieces.ChessPiece{}, false
}

func signInt(v int) int {
	if v < 0 {
		return -1
	}
	if v > 0 {
		return 1
	}
	return 0
}

func absCheck(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
