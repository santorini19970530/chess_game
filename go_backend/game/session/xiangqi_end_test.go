// CM3070 FP code
// xiangqi_end_test.go - tests for xiangqi ply-end strategies

package session

import "testing"

func TestXiangqiPerpetualCheckStrategy_CheckerLoses(t *testing.T) {
	ended, out := (XiangqiPerpetualCheckStrategy{}).AfterPly(&plyEndContext{
		GameType:            GameTypeXiangqi,
		LegalMoves:          4,
		PerpetualCheckLoser: "white",
	})
	if !ended {
		t.Fatal("expected perpetual check to end the ply")
	}
	if out.Status != "perpetual_check" {
		t.Fatalf("status=%q", out.Status)
	}
	if out.Loser != "white" || out.Winner != "black" {
		t.Fatalf("winner=%q loser=%q", out.Winner, out.Loser)
	}
}

func TestXiangqiPerpetualChaseStrategy_ChaserLoses(t *testing.T) {
	ended, out := (XiangqiPerpetualChaseStrategy{}).AfterPly(&plyEndContext{
		GameType:            GameTypeXiangqi,
		LegalMoves:          4,
		PerpetualChaseLoser: "black",
	})
	if !ended {
		t.Fatal("expected perpetual chase to end the ply")
	}
	if out.Status != "perpetual_chase" {
		t.Fatalf("status=%q", out.Status)
	}
	if out.Loser != "black" || out.Winner != "white" {
		t.Fatalf("winner=%q loser=%q", out.Winner, out.Loser)
	}
}

func TestXiangqiMutualRepetitionStrategy_Draw(t *testing.T) {
	ended, out := (XiangqiMutualRepetitionStrategy{}).AfterPly(&plyEndContext{
		GameType:    GameTypeXiangqi,
		LegalMoves:  4,
		CycleRepeat: true,
	})
	if !ended || out.Status != "draw_mutual_repetition" {
		t.Fatalf("ended=%v status=%q", ended, out.Status)
	}
}

func TestXiangqiNoCaptureStrategy_IdleSixty(t *testing.T) {
	s := XiangqiNoCaptureStrategy{}
	ended, _ := s.AfterPly(&plyEndContext{GameType: GameTypeXiangqi, IdlePly: 59})
	if ended {
		t.Fatal("59 idle ply must not draw")
	}
	ended, out := s.AfterPly(&plyEndContext{GameType: GameTypeXiangqi, IdlePly: 60, LegalMoves: 8})
	if !ended || out.Status != "draw_no_capture" {
		t.Fatalf("ended=%v status=%q", ended, out.Status)
	}
}

func TestXiangqiPerpetualChaseStrategy_KingOrSoldierNotChase(t *testing.T) {
	ended, _ := (XiangqiPerpetualChaseStrategy{}).AfterPly(&plyEndContext{
		GameType:                 GameTypeXiangqi,
		CycleRepeat:              true,
		ChaseIsKingOrSoldierOnly: true,
	})
	if ended {
		t.Fatal("king/soldier-only chase must not be a chase loss")
	}
}

func TestXiangqiGameEndChain_MateBeforeCheckBeforeChase(t *testing.T) {
	clearGameEndStrategiesForTest()
	registerGameEndStrategies(GameTypeXiangqi, xiangqiGameEndStrategies()...)

	ended, out := runGameEndStrategies(&plyEndContext{
		GameType:            GameTypeXiangqi,
		LegalMoves:          0,
		SideToMove:          "black",
		InCheck:             true,
		PerpetualCheckLoser: "white",
		PerpetualChaseLoser: "white",
	})
	if !ended || out.Status != "checkmate" {
		t.Fatalf("mate must win the chain, ended=%v status=%q", ended, out.Status)
	}

	ended, out = runGameEndStrategies(&plyEndContext{
		GameType:            GameTypeXiangqi,
		LegalMoves:          5,
		SideToMove:          "black",
		PerpetualCheckLoser: "white",
		PerpetualChaseLoser: "white",
	})
	if !ended || out.Status != "perpetual_check" {
		t.Fatalf("check must beat chase, ended=%v status=%q", ended, out.Status)
	}
}

func TestApplyXiangqiUCIMove_IdlePlyIncrementsAndResets(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if xiangqiIdlePly != 0 {
		t.Fatalf("idle after create=%d", xiangqiIdlePly)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "a4a5"); err != nil {
		t.Fatalf("a4a5: %v", err)
	}
	if xiangqiIdlePly != 1 {
		t.Fatalf("idle after quiet ply=%d", xiangqiIdlePly)
	}
}

// TestXiangqiWXFEnd_StopsFurtherMoves - 60-ply idle draw must stop /move, HvAI, and CanAcceptMoves
func TestXiangqiWXFEnd_StopsFurtherMoves(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsAI, GameTypeXiangqi, "white", 1, "", "beginner")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := SetXiangqiIdlePlyByID(game.ID, 59); err != nil {
		t.Fatalf("idle: %v", err)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "a4a5"); err != nil {
		t.Fatalf("a4a5: %v", err)
	}
	ended, err := GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ended.Result != GameResultDraw || ended.Outcome.Status != "draw_no_capture" {
		t.Fatalf("result=%q status=%q", ended.Result, ended.Outcome.Status)
	}
	if ended.Mode != GameModeHumanVsAI {
		t.Fatalf("mode=%q", ended.Mode)
	}
	if _, err := ApplyMoveByCommandByID(game.ID, "a7a6"); err == nil {
		t.Fatal("next /move must be rejected after WXF end")
	}
	if CanAcceptMoves() {
		t.Fatal("CanAcceptMoves must be false after WXF end")
	}
}
