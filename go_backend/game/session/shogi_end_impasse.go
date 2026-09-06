// CM3070 FP code
// shogi_end_impasse.go - shogi FESA 5.3 entering-king declaration (入玉宣言)

package session

import (
	"go_backend/game/movement"
	pieces "go_backend/game/piece"
)

const (
	shogiImpasseSentePoints = 28
	shogiImpasseGotePoints  = 27
	shogiImpasseCampPieces  = 10
)

type shogiImpasseTally struct {
	KingInZone bool
	InCheck    bool
	CampCount  int
	Points     int
}

// shogiImpassePointsNeeded - sente 28, gote 27
func shogiImpassePointsNeeded(side pieces.PieceColor) int {
	if side == pieces.Black {
		return shogiImpasseGotePoints
	}
	return shogiImpasseSentePoints
}

// shogiImpassePiecePoints - rook/bishop (and promoted) 5; king 0; others 1
func shogiImpassePiecePoints(kind pieces.PieceKind) int {
	switch kind {
	case pieces.King:
		return 0
	case pieces.Rook, pieces.Bishop, pieces.Dragon, pieces.Horse:
		return 5
	default:
		return 1
	}
}

// tallyShogiImpasse - camp pieces and points for FESA 5.3 (camp + hand; king excluded)
func tallyShogiImpasse(side pieces.PieceColor) shogiImpasseTally {
	var t shogiImpasseTally
	t.InCheck = movement.ShogiCheckedColor() == side
	for _, p := range pieces.ChessPieces {
		if p.Color != side {
			continue
		}
		if p.Kind == pieces.King {
			t.KingInZone = movement.ShogiInPromotionZone(p.Rank, side)
			continue
		}
		if !movement.ShogiInPromotionZone(p.Rank, side) {
			continue
		}
		t.CampCount++
		t.Points += shogiImpassePiecePoints(p.Kind)
	}
	for kind, n := range shogiHandMap(side) {
		if n <= 0 {
			continue
		}
		t.Points += n * shogiImpassePiecePoints(kind)
	}
	return t
}

// shogiImpasseMeetsFESA - reports whether the side to move may declare a win (FESA 5.3)
func shogiImpasseMeetsFESA(side pieces.PieceColor) bool {
	t := tallyShogiImpasse(side)
	return t.KingInZone && !t.InCheck && t.CampCount >= shogiImpasseCampPieces && t.Points >= shogiImpassePointsNeeded(side)
}

// applyShogiImpasseDeclarationLocked - pass wins; any failed checklist loses
func applyShogiImpasseDeclarationLocked(game *RuntimeGame) {
	side := CurrentTurnColor()
	if shogiImpasseMeetsFESA(side) {
		winner := string(side)
		loser := string(opponentOf(side))
		game.Session.Outcome = GameOutcome{
			Status:     "impasse",
			Winner:     winner,
			Loser:      loser,
			LegalMoves: 0,
			Message:    "Impasse declared. " + sideLabel(side) + " wins.",
		}
	} else {
		game.Session.Outcome = xiangqiLossOutcome(
			"impasse_failed",
			string(side),
			"Impasse declaration failed. "+sideLabel(side)+" loses.",
			0,
		)
	}
	game.Session.Result = gameResultFromOutcome(game.Session.Outcome)
}
