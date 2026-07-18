package session

import "testing"

func TestXiangqiCapturedSummary_StartPositionEmpty(t *testing.T) {
	resetGameSessionForTest()
	ResetGame()
	game, err := CreateGame(GameModeHumanVsHuman, GameTypeXiangqi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	snap, err := BuildSnapshotByID(game.ID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	for _, side := range []map[string]int{snap.Captured.White, snap.Captured.Black} {
		for kind, n := range side {
			if n != 0 {
				t.Fatalf("start position should have 0 captured, got %s=%d in %+v", kind, n, side)
			}
		}
	}
	// Must not invent Chess-only "queen/bishop" captures on a Xiangqi board.
	if snap.Captured.White["queen"] != 0 || snap.Captured.White["bishop"] != 0 {
		t.Fatalf("unexpected chess kinds in xiangqi captured: %+v", snap.Captured.White)
	}
}
