// CM3070 FP code
// shogi_end_test.go - tests for shogi ply-end context, sennichite, and continuous check

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
		t.Fatalf("second visit count=%d want 2", ctx.PositionCount)
	}
	if EvaluateShogiGameOutcome().Status == "draw_sennichite" {
		t.Fatal("twofold must not be sennichite")
	}
}

func TestShogiContinuousCheckStrategy_CheckerLoses(t *testing.T) {
	ended, out := (ShogiContinuousCheckStrategy{}).AfterPly(&plyEndContext{
		GameType:            GameTypeShogi,
		LegalMoves:          4,
		PerpetualCheckLoser: "white",
	})
	if !ended {
		t.Fatal("expected continuous check to end the ply")
	}
	if out.Status != "continuous_check" {
		t.Fatalf("status=%q", out.Status)
	}
	if out.Loser != "white" || out.Winner != "black" {
		t.Fatalf("winner=%q loser=%q", out.Winner, out.Loser)
	}
}

func TestShogiSennichiteStrategy_FourfoldDraw(t *testing.T) {
	s := ShogiSennichiteStrategy{}
	ended, _ := s.AfterPly(&plyEndContext{GameType: GameTypeShogi, CycleRepeat: false, PositionCount: 3})
	if ended {
		t.Fatal("threefold must not be sennichite")
	}
	ended, out := s.AfterPly(&plyEndContext{GameType: GameTypeShogi, CycleRepeat: true, LegalMoves: 8})
	if !ended || out.Status != "draw_sennichite" {
		t.Fatalf("ended=%v status=%q", ended, out.Status)
	}
}

func TestClassifyShogiRepeatCycle_IncludesFirstPlyOfCycle(t *testing.T) {
	// keys[0] is start (no fact). facts[i] produced keys[i+1]. fourfold START at 0,4,8,12.
	shogiPositionKeys = []string{
		"START", "p1", "p2", "p3",
		"START", "p1", "p2", "p3",
		"START", "p1", "p2", "p3",
		"START",
	}
	shogiPositionCounts = map[string]int{"START": 4}
	shogiPlyFacts = make([]shogiPlyFact, 12)
	for i := range shogiPlyFacts {
		side := "white"
		if i%2 == 1 {
			side = "black"
		}
		shogiPlyFacts[i] = shogiPlyFact{Side: side, GaveCheck: false}
	}
	shogiPlyFacts[10].GaveCheck = true
	ctx := &plyEndContext{PositionCount: 4}
	classifyShogiRepeatCycle(ctx)
	if !ctx.CycleRepeat {
		t.Fatal("fourfold must set CycleRepeat")
	}
	if ctx.PerpetualCheckLoser != "" {
		t.Fatalf("a cycle with a quiet white ply must be sennichite, loser=%q", ctx.PerpetualCheckLoser)
	}
}

func TestClassifyShogiRepeatCycle_BothSidesCheckingIsDraw(t *testing.T) {
	shogiPositionKeys = []string{
		"START", "p1", "p2", "p3",
		"START", "p1", "p2", "p3",
		"START", "p1", "p2", "p3",
		"START",
	}
	shogiPositionCounts = map[string]int{"START": 4}
	shogiPlyFacts = make([]shogiPlyFact, 12)
	for i := range shogiPlyFacts {
		side := "white"
		if i%2 == 1 {
			side = "black"
		}
		shogiPlyFacts[i] = shogiPlyFact{Side: side, GaveCheck: true}
	}
	ctx := &plyEndContext{PositionCount: 4}
	classifyShogiRepeatCycle(ctx)
	if !ctx.CycleRepeat {
		t.Fatal("fourfold must set CycleRepeat")
	}
	if ctx.PerpetualCheckLoser != "" {
		t.Fatalf("both sides checking must not name a loser, got %q", ctx.PerpetualCheckLoser)
	}
}

func TestShogiGameEndChain_MateBeforeCheckBeforeSennichite(t *testing.T) {
	clearGameEndStrategiesForTest()
	registerGameEndStrategies(GameTypeShogi, shogiGameEndStrategies()...)

	ended, out := runGameEndStrategies(&plyEndContext{
		GameType:            GameTypeShogi,
		LegalMoves:          0,
		SideToMove:          "black",
		InCheck:             true,
		PerpetualCheckLoser: "white",
		CycleRepeat:         true,
	})
	if !ended || out.Status != "checkmate" {
		t.Fatalf("mate must win the chain, ended=%v status=%q", ended, out.Status)
	}

	ended, out = runGameEndStrategies(&plyEndContext{
		GameType:            GameTypeShogi,
		LegalMoves:          5,
		PerpetualCheckLoser: "white",
		CycleRepeat:         true,
	})
	if !ended || out.Status != "continuous_check" {
		t.Fatalf("continuous check must beat sennichite, ended=%v status=%q", ended, out.Status)
	}
}

func TestShogiSennichite_FourfoldStartIsDraw(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cycle := []string{"d1d2", "d9d8", "d2d1", "d8d9"}
	for i := 0; i < 2; i++ {
		for _, mv := range cycle {
			if _, err := ApplyMoveByCommandByID(game.ID, mv); err != nil {
				t.Fatalf("cycle %d %s: %v", i, mv, err)
			}
		}
	}
	mid, err := GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if mid.Outcome.Status == "draw_sennichite" {
		t.Fatal("third visit of start must not draw")
	}
	for _, mv := range cycle {
		if _, err := ApplyMoveByCommandByID(game.ID, mv); err != nil {
			t.Fatalf("fourth %s: %v", mv, err)
		}
	}
	ended, err := GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ended.Result != GameResultDraw || ended.Outcome.Status != "draw_sennichite" {
		t.Fatalf("result=%q status=%q count=%d", ended.Result, ended.Outcome.Status, buildShogiPlyEndContext().PositionCount)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "c3c4"); err == nil {
		t.Fatal("next /move must be rejected after sennichite")
	}
}

func TestShogiContinuousCheck_RookShuttleLoses(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	fen := "4k4/9/9/9/4R4/9/9/9/4K4[] w - - 0 1"
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeShogi, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "e5e8"); err != nil {
		t.Fatalf("e5e8: %v", err)
	}
	cycle := []string{"e9d9", "e8d8", "d9e9", "d8e8"}
	for i := 0; i < 3; i++ {
		for _, mv := range cycle {
			if _, err := ApplyMoveByCommandByID(game.ID, mv); err != nil {
				t.Fatalf("cycle %d %s: %v", i, mv, err)
			}
		}
	}
	ended, err := GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ended.Outcome.Status != "continuous_check" {
		t.Fatalf("status=%q want continuous_check (count=%d)", ended.Outcome.Status, buildShogiPlyEndContext().PositionCount)
	}
	if ended.Result != GameResultBlackWin || ended.Outcome.Loser != "white" {
		t.Fatalf("result=%q loser=%q", ended.Result, ended.Outcome.Loser)
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
