package movement

import pieces "go_backend/game/piece"

// ShogiWouldLeaveKingInCheck reports whether applying from→to leaves the mover's king attacked.
func ShogiWouldLeaveKingInCheck(source pieces.ChessPiece, fromFile, fromRank, toFile, toRank int) bool {
	after := simulateShogiMove(pieces.ChessPieces, fromFile, fromRank, toFile, toRank)
	return shogiKingInCheckOnBoard(after, source.Color)
}

func simulateShogiMove(board []pieces.ChessPiece, fromFile, fromRank, toFile, toRank int) []pieces.ChessPiece {
	cloned := append([]pieces.ChessPiece(nil), board...)
	srcIdx := -1
	tgtIdx := -1
	for i := range cloned {
		if cloned[i].File == fromFile && cloned[i].Rank == fromRank {
			srcIdx = i
		}
		if cloned[i].File == toFile && cloned[i].Rank == toRank {
			tgtIdx = i
		}
	}
	if srcIdx == -1 {
		return cloned
	}
	if tgtIdx != -1 {
		cloned = append(cloned[:tgtIdx], cloned[tgtIdx+1:]...)
		if tgtIdx < srcIdx {
			srcIdx--
		}
	}
	cloned[srcIdx].File = toFile
	cloned[srcIdx].Rank = toRank
	return cloned
}

func shogiKingInCheckOnBoard(board []pieces.ChessPiece, color pieces.PieceColor) bool {
	kf, kr, ok := findShogiKing(board, color)
	if !ok {
		return true
	}
	attacker := pieces.White
	if color == pieces.White {
		attacker = pieces.Black
	}
	return shogiSquareAttackedOnBoard(board, kf, kr, attacker)
}

func findShogiKing(board []pieces.ChessPiece, color pieces.PieceColor) (int, int, bool) {
	for _, p := range board {
		if p.Kind == pieces.King && p.Color == color {
			return p.File, p.Rank, true
		}
	}
	return 0, 0, false
}

// ShogiCheckedColor returns the side in check under Shogi rules, if any.
func ShogiCheckedColor() pieces.PieceColor {
	if shogiKingInCheckOnBoard(pieces.ChessPieces, pieces.White) {
		return pieces.White
	}
	if shogiKingInCheckOnBoard(pieces.ChessPieces, pieces.Black) {
		return pieces.Black
	}
	return ""
}

func shogiSquareAttackedOnBoard(board []pieces.ChessPiece, file, rank int, attacker pieces.PieceColor) bool {
	saved := pieces.ChessPieces
	pieces.ChessPieces = board
	defer func() { pieces.ChessPieces = saved }()

	for _, p := range board {
		if p.Color != attacker {
			continue
		}
		for _, sq := range ShogiLegalSquares(p.Kind, p.Color, p.File, p.Rank) {
			if sq.File == file && sq.Rank == rank {
				return true
			}
		}
	}
	return false
}

// ShogiWouldLeaveKingInCheckAfterDrop reports whether dropping kind at file/rank leaves mover in check.
func ShogiWouldLeaveKingInCheckAfterDrop(kind pieces.PieceKind, color pieces.PieceColor, file, rank int) bool {
	after := append([]pieces.ChessPiece(nil), pieces.ChessPieces...)
	after = append(after, pieces.ChessPiece{Color: color, Kind: kind, File: file, Rank: rank})
	return shogiKingInCheckOnBoard(after, color)
}
