package session

import (
	"strings"
	"testing"
)

func TestShogiHumanMove_PawnAdvances(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "c3c4"); err != nil {
		t.Fatalf("c3c4: %v", err)
	}
	fen, err := CurrentFENByID(game.ID)
	if err != nil {
		t.Fatalf("fen: %v", err)
	}
	if fen == DefaultShogiStartFEN {
		t.Fatal("FEN should change")
	}
	if !strings.Contains(fen, " b ") {
		t.Fatalf("expected black to move, fen=%q", fen)
	}
}

func TestShogiHumanMove_IllegalRejected(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "c3c5"); err == nil {
		t.Fatal("pawn double-step should be illegal")
	}
	fen, _ := CurrentFENByID(game.ID)
	if fen != DefaultShogiStartFEN {
		t.Fatalf("FEN unchanged after illegal, got %q", fen)
	}
}

func TestShogiCapture_GoesToHand(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	// White rook e5 captures black pawn e8 → pawn enters white hand.
	fen := "4k4/4p4/9/9/4R4/9/9/9/4K4[] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "e5e8"); err != nil {
		t.Fatalf("capture: %v", err)
	}
	snap, err := BuildSnapshotByID(game.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.Captured.White["pawn"] != 1 {
		t.Fatalf("expected white hand pawn=1, got %+v", snap.Captured.White)
	}
}

func TestShogiDrop_FromHand(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	fen := "4k4/9/9/9/9/9/9/9/4K4[P] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	dests, err := LegalDropsForKindByID(game.ID, "pawn")
	if err != nil {
		t.Fatalf("legal drops: %v", err)
	}
	if len(dests) == 0 {
		t.Fatal("expected pawn drop destinations")
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "P*e5"); err != nil {
		t.Fatalf("drop P*e5: %v", err)
	}
	snap, err := BuildSnapshotByID(game.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.Captured.White["pawn"] != 0 {
		t.Fatalf("hand should consume pawn, got %+v", snap.Captured.White)
	}
	found := false
	for _, p := range snap.State {
		if p.Kind == "pawn" && p.Color == "white" && p.File == 5 && p.Rank == 5 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected dropped white pawn on e5")
	}
}

func TestShogiDrop_NifuRejected(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	// White already has pawn on e3; pawn in hand; try drop on e4.
	fen := "4k4/9/9/9/9/9/4P4/9/4K4[P] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if shogiHands.white["pawn"] != 1 {
		t.Fatalf("expected pawn in hand, hands=%+v", shogiHands.white)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "P*e4"); err == nil {
		t.Fatal("nifu drop should be rejected")
	}
	if shogiHands.white["pawn"] != 1 {
		t.Fatalf("hand should restore after failed drop, got %+v", shogiHands.white)
	}
}

func TestShogiPromotion_AutoOnLastRank(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	// White pawn on e8 can move to e9 and must promote (black king on d9).
	fen := "3k5/4P4/9/9/9/9/9/9/4K4[] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	normalized, err := ApplyMoveByCommandByID(game.ID, "e8e9")
	if err != nil {
		t.Fatalf("e8e9: %v", err)
	}
	if normalized != "e8e9+" {
		t.Fatalf("normalized=%q want e8e9+", normalized)
	}
	snap, _ := BuildSnapshotByID(game.ID)
	found := false
	for _, p := range snap.State {
		if p.File == 5 && p.Rank == 9 && p.Kind == "promoted_pawn" && p.Color == "white" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tokin on e9, state=%+v", snap.State)
	}
}

func TestShogiLegalMoves_IncludesDrops(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()

	fen := "4k4/9/9/9/9/9/9/9/4K4[P] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	moves, err := AllLegalUCIMovesByID(game.ID)
	if err != nil {
		t.Fatalf("legal: %v", err)
	}
	foundDrop := false
	for _, mv := range moves {
		if strings.HasPrefix(strings.ToLower(mv), "p*") {
			foundDrop = true
			break
		}
	}
	if !foundDrop {
		t.Fatalf("expected pawn drops in legal list, got %v", moves)
	}
}
