package movement

import pieces "go_backend/game/piece"

// XiangqiWouldLeaveGeneralInCheck reports whether applying from→to leaves mover's general
// attacked or creates a flying-general face-off.
func XiangqiWouldLeaveGeneralInCheck(source pieces.ChessPiece, fromFile, fromRank, toFile, toRank int) bool {
	after := simulateXiangqiMove(pieces.ChessPieces, fromFile, fromRank, toFile, toRank)
	return xiangqiGeneralInCheckOnBoard(after, source.Color) || xiangqiFlyingGeneral(after)
}

func simulateXiangqiMove(board []pieces.ChessPiece, fromFile, fromRank, toFile, toRank int) []pieces.ChessPiece {
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

func xiangqiGeneralInCheckOnBoard(board []pieces.ChessPiece, color pieces.PieceColor) bool {
	gf, gr, ok := findGeneral(board, color)
	if !ok {
		return true
	}
	attacker := pieces.White
	if color == pieces.White {
		attacker = pieces.Black
	}
	return xiangqiSquareAttackedOnBoard(board, gf, gr, attacker)
}

func findGeneral(board []pieces.ChessPiece, color pieces.PieceColor) (int, int, bool) {
	for _, p := range board {
		if p.Kind == pieces.King && p.Color == color {
			return p.File, p.Rank, true
		}
	}
	return 0, 0, false
}

// XiangqiCheckedColor returns the side in check under Xiangqi rules, if any.
func XiangqiCheckedColor() pieces.PieceColor {
	if xiangqiGeneralInCheckOnBoard(pieces.ChessPieces, pieces.White) {
		return pieces.White
	}
	if xiangqiGeneralInCheckOnBoard(pieces.ChessPieces, pieces.Black) {
		return pieces.Black
	}
	return ""
}

// Flying general: same file, no pieces between the two kings.
func xiangqiFlyingGeneral(board []pieces.ChessPiece) bool {
	wf, wr, wok := findGeneral(board, pieces.White)
	bf, br, bok := findGeneral(board, pieces.Black)
	if !wok || !bok || wf != bf {
		return false
	}
	lo, hi := wr, br
	if lo > hi {
		lo, hi = hi, lo
	}
	for _, p := range board {
		if p.File == wf && p.Rank > lo && p.Rank < hi {
			return false
		}
	}
	return true
}

func xiangqiSquareAttackedOnBoard(board []pieces.ChessPiece, file, rank int, attacker pieces.PieceColor) bool {
	saved := pieces.ChessPieces
	pieces.ChessPieces = board
	defer func() { pieces.ChessPieces = saved }()

	for _, p := range board {
		if p.Color != attacker {
			continue
		}
		for _, sq := range XiangqiLegalSquares(p.Kind, p.Color, p.File, p.Rank) {
			if sq.File == file && sq.Rank == rank {
				return true
			}
		}
	}
	return false
}
