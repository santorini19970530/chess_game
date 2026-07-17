package session

import pieces "go_backend/game/piece"

func shogiHandMap(color pieces.PieceColor) map[pieces.PieceKind]int {
	if color == pieces.Black {
		return shogiHands.black
	}
	return shogiHands.white
}

func shogiAddToHand(color pieces.PieceColor, kind pieces.PieceKind) {
	base := shogiUnpromoteForHand(kind)
	if base == pieces.King || base == "" {
		return
	}
	m := shogiHandMap(color)
	m[base]++
}

func shogiTakeFromHand(color pieces.PieceColor, kind pieces.PieceKind) bool {
	m := shogiHandMap(color)
	if m[kind] <= 0 {
		return false
	}
	m[kind]--
	if m[kind] == 0 {
		delete(m, kind)
	}
	return true
}

func shogiHandCount(color pieces.PieceColor, kind pieces.PieceKind) int {
	return shogiHandMap(color)[kind]
}

// Captured promoted pieces return to hand as their unpromoted form.
func shogiUnpromoteForHand(kind pieces.PieceKind) pieces.PieceKind {
	switch kind {
	case pieces.PromotedPawn:
		return pieces.Pawn
	case pieces.PromotedLance:
		return pieces.Lance
	case pieces.PromotedKnight:
		return pieces.Knight
	case pieces.PromotedSilver:
		return pieces.Silver
	case pieces.Horse:
		return pieces.Bishop
	case pieces.Dragon:
		return pieces.Rook
	case pieces.Pawn, pieces.Lance, pieces.Knight, pieces.Silver, pieces.Gold, pieces.Bishop, pieces.Rook:
		return kind
	default:
		return ""
	}
}

func shogiHandsSummary() CapturedSummary {
	toMap := func(src map[pieces.PieceKind]int) map[string]int {
		out := map[string]int{}
		for k, n := range src {
			if n > 0 {
				out[string(k)] = n
			}
		}
		return out
	}
	return CapturedSummary{
		White: toMap(shogiHands.white),
		Black: toMap(shogiHands.black),
	}
}

// shogiHasUnpromotedPawnOnFile reports nifu risk for a pawn drop on file.
func shogiHasUnpromotedPawnOnFile(color pieces.PieceColor, file int) bool {
	for _, p := range pieces.ChessPieces {
		if p.Color == color && p.Kind == pieces.Pawn && p.File == file {
			return true
		}
	}
	return false
}
