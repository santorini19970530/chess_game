// CM3070 FP code
// game_end_strategy_test.go - tests for ply-end strategy registry

package session

import "testing"

type stubGameEnd struct {
	name   string
	ended  bool
	status string
	calls  *int
}

func (s stubGameEnd) Name() string { return s.name }

func (s stubGameEnd) AfterPly(ctx *plyEndContext) (bool, GameOutcome) {
	if s.calls != nil {
		*s.calls++
	}
	if !s.ended {
		return false, GameOutcome{}
	}
	return true, GameOutcome{Status: s.status}
}

func TestRunGameEndStrategies_EmptyRegistryDoesNotEnd(t *testing.T) {
	clearGameEndStrategiesForTest()
	ended, out := runGameEndStrategies(&plyEndContext{GameType: GameTypeXiangqi})
	if ended {
		t.Fatalf("empty registry should not end the ply")
	}
	if out.Status != "in_progress" {
		t.Fatalf("expected in_progress, got %q", out.Status)
	}
}

func TestRunGameEndStrategies_FirstEndedWins(t *testing.T) {
	clearGameEndStrategiesForTest()
	secondCalls := 0
	registerGameEndStrategies(GameTypeXiangqi,
		stubGameEnd{name: "check", ended: true, status: "perpetual_check"},
		stubGameEnd{name: "chase", ended: true, status: "perpetual_chase", calls: &secondCalls},
	)
	ended, out := runGameEndStrategies(&plyEndContext{GameType: GameTypeXiangqi})
	if !ended {
		t.Fatalf("expected first strategy to end the ply")
	}
	if out.Status != "perpetual_check" {
		t.Fatalf("expected perpetual_check, got %q", out.Status)
	}
	if secondCalls != 0 {
		t.Fatalf("later strategy must not run after ended, calls=%d", secondCalls)
	}
}

func TestRunGameEndStrategies_SkipsUntilEnded(t *testing.T) {
	clearGameEndStrategiesForTest()
	registerGameEndStrategies(GameTypeXiangqi,
		stubGameEnd{name: "mate", ended: false},
		stubGameEnd{name: "idle", ended: true, status: "draw_no_capture"},
	)
	ended, out := runGameEndStrategies(&plyEndContext{GameType: GameTypeXiangqi})
	if !ended || out.Status != "draw_no_capture" {
		t.Fatalf("expected draw_no_capture from second strategy, ended=%v status=%q", ended, out.Status)
	}
}

func TestRunGameEndStrategies_KeyedByGameType(t *testing.T) {
	clearGameEndStrategiesForTest()
	registerGameEndStrategies(GameTypeXiangqi, stubGameEnd{name: "xq", ended: true, status: "perpetual_check"})
	registerGameEndStrategies(GameTypeChess, stubGameEnd{name: "fifty", ended: true, status: "draw_fifty_move_rule"})
	ended, out := runGameEndStrategies(&plyEndContext{GameType: GameTypeChess})
	if !ended || out.Status != "draw_fifty_move_rule" {
		t.Fatalf("chess registry should not use xiangqi list, ended=%v status=%q", ended, out.Status)
	}
}
