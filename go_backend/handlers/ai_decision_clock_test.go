package handlers

import (
	"testing"
	"time"

	sessionpkg "go_backend/game/session"
)

func TestFsLimitForGame_ClockOffKeepsProfileMovetime(t *testing.T) {
	limit := fsLimitForGame("beginner", nil, "white")
	if limit.MoveTime != 250*time.Millisecond {
		t.Fatalf("movetime=%v", limit.MoveTime)
	}
	if limit.WhiteTime != 0 || limit.BlackTime != 0 {
		t.Fatalf("unexpected clock fields: %+v", limit)
	}

	disabled := sessionpkg.NewClock(0, 0, 0)
	limit = fsLimitForGame("beginner", disabled, "white")
	if limit.WhiteTime != 0 || limit.MoveTime != 250*time.Millisecond {
		t.Fatalf("disabled clock should use profile only: %+v", limit)
	}
}

func TestFsLimitForGame_ClockOnSetsWtimeAndCapsMovetime(t *testing.T) {
	clk := sessionpkg.NewClock(300_000, 60_000, 30_000)
	clk.Start("black", time.Unix(0, 0).UTC())
	limit := fsLimitForGame("master", clk, "black")
	if limit.WhiteTime != 300_000*time.Millisecond || limit.BlackTime != 60_000*time.Millisecond {
		t.Fatalf("w/b time: white=%v black=%v", limit.WhiteTime, limit.BlackTime)
	}
	if limit.WhiteInc != 30_000*time.Millisecond || limit.BlackInc != 30_000*time.Millisecond {
		t.Fatalf("inc: white=%v black=%v", limit.WhiteInc, limit.BlackInc)
	}
	// master profile movetime is 1200ms; black remaining/20 = 3000ms → keep profile 1200.
	if limit.MoveTime != 1200*time.Millisecond {
		t.Fatalf("movetime cap=%v want 1200ms", limit.MoveTime)
	}

	// Short remaining: cap below profile budget.
	short := sessionpkg.NewClock(1000, 1000, 0)
	short.Start("white", time.Unix(0, 0).UTC())
	limit = fsLimitForGame("master", short, "white")
	// remaining/20 = 50ms → floor 50ms
	if limit.MoveTime != 50*time.Millisecond {
		t.Fatalf("short remaining movetime=%v want 50ms", limit.MoveTime)
	}
}
