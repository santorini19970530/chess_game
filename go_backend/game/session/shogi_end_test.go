// CM3070 FP code
// shogi_end_test.go - tests for shogi ply-end context (board + hands + side)

package session

import "testing"

func TestShogiPlyContext_StartCountedOnce(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	if _, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", ""); err != nil {
		t.Fatalf("create: %v", err)
	}
	ctx := buildShogiPlyEndContext()
	if ctx.PositionKey == "" {
		t.Fatal("start position key must include board, hands, and side")
	}
	if ctx.PositionCount != 1 {
		t.Fatalf("start count=%d want 1", ctx.PositionCount)
	}
}

func TestShogiPlyContext_HandsChangeTheKey(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	empty := "4k4/9/9/9/9/9/9/9/4K4[] w - - 0 1"
	if _, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, empty, ""); err != nil {
		t.Fatalf("empty: %v", err)
	}
	keyEmpty := buildShogiPlyEndContext().PositionKey

	resetGameSessionForTest()
	ResetGame()
	hand := "4k4/9/9/9/9/9/9/9/4K4[P] w - - 0 1"
	if _, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, hand, ""); err != nil {
		t.Fatalf("hand: %v", err)
	}
	keyHand := buildShogiPlyEndContext().PositionKey
	if keyEmpty == "" || keyHand == "" {
		t.Fatal("keys must not be empty")
	}
	if keyEmpty == keyHand {
		t.Fatalf("same board and side with different hands must not share a key: %q", keyEmpty)
	}
}

func TestShogiPlyContext_SideChangesTheKey(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	white := "4k4/9/9/9/9/9/9/9/4K4[] w - - 0 1"
	if _, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, white, ""); err != nil {
		t.Fatalf("white: %v", err)
	}
	keyW := buildShogiPlyEndContext().PositionKey

	resetGameSessionForTest()
	ResetGame()
	black := "4k4/9/9/9/9/9/9/9/4K4[] b - - 0 1"
	if _, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, black, ""); err != nil {
		t.Fatalf("black: %v", err)
	}
	keyB := buildShogiPlyEndContext().PositionKey
	if keyW == keyB {
		t.Fatalf("side to move must be in the key: %q", keyW)
	}
}

func TestShogiPlyContext_QuietPlyIncrementsNewPosition(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	startKey := buildShogiPlyEndContext().PositionKey
	if _, err := ApplyMoveByCommandByID(game.ID, "c3c4"); err != nil {
		t.Fatalf("c3c4: %v", err)
	}
	ctx := buildShogiPlyEndContext()
	if ctx.PositionKey == "" || ctx.PositionKey == startKey {
		t.Fatalf("quiet ply must be a new key, start=%q now=%q", startKey, ctx.PositionKey)
	}
	if ctx.PositionCount != 1 {
		t.Fatalf("new position count=%d want 1", ctx.PositionCount)
	}
	if ctx.SideJustMoved != "white" {
		t.Fatalf("side just moved=%q", ctx.SideJustMoved)
	}
}

func TestShogiPlyContext_ReturnToStartIsSecondOccurrence(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	startKey := buildShogiPlyEndContext().PositionKey
	for _, mv := range []string{"d1d2", "d9d8", "d2d1", "d8d9"} {
		if _, err := ApplyMoveByCommandByID(game.ID, mv); err != nil {
			t.Fatalf("%s: %v", mv, err)
		}
	}
	ctx := buildShogiPlyEndContext()
	if ctx.PositionKey != startKey {
		t.Fatalf("after reversing the pawn pair, key=%q want start %q", ctx.PositionKey, startKey)
	}
	if ctx.PositionCount != 2 {
		t.Fatalf("second visit count=%d want 2 (sennichite still later)", ctx.PositionCount)
	}
	if EvaluateShogiGameOutcome().Status == "draw_sennichite" {
		t.Fatal("step 3 only records; fourfold draw is not this step")
	}
}

func TestShogiPlyContext_DropChangesHandsInKey(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	fen := "4k4/9/9/9/9/9/9/9/4K4[P] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := buildShogiPlyEndContext().PositionKey
	if _, err := ApplyMoveByCommandByID(game.ID, "P*e5"); err != nil {
		t.Fatalf("drop: %v", err)
	}
	ctx := buildShogiPlyEndContext()
	if ctx.PositionKey == before {
		t.Fatal("drop must change the key because hands changed")
	}
	if ctx.PositionCount != 1 {
		t.Fatalf("drop position count=%d want 1", ctx.PositionCount)
	}
}

func TestShogiPlyContext_GaveCheckOnCheckingPly(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	fen := "4k4/9/4G4/9/9/9/9/9/4K4[] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "e7e8"); err != nil {
		t.Fatalf("e7e8: %v", err)
	}
	ctx := buildShogiPlyEndContext()
	if !ctx.GaveCheck {
		t.Fatal("checking ply must set GaveCheck for later FESA 3.12")
	}
	if ctx.SideJustMoved != "white" {
		t.Fatalf("side just moved=%q", ctx.SideJustMoved)
	}
}

func TestShogiPlyContext_PreviewDoesNotRecord(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := buildShogiPlyEndContext()
	if _, _, err := PreviewChildFENByCommandByID(game.ID, "c3c4"); err != nil {
		t.Fatalf("preview: %v", err)
	}
	after := buildShogiPlyEndContext()
	if after.PositionKey != before.PositionKey {
		t.Fatalf("preview must not change live key, before=%q after=%q", before.PositionKey, after.PositionKey)
	}
	if after.PositionCount != before.PositionCount {
		t.Fatalf("preview must not count, before=%d after=%d", before.PositionCount, after.PositionCount)
	}
	if len(shogiPositionKeys) != 1 {
		t.Fatalf("preview leaked ply keys=%d want 1", len(shogiPositionKeys))
	}
}
