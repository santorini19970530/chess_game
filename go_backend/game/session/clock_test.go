package session

import (
	"testing"
	"time"
)

func TestClock_TickDownAndFlag(t *testing.T) {
	c := NewClock(1000, 1000, 0)
	start := time.Unix(0, 0).UTC()
	c.Start("white", start)

	c.Settle(start.Add(400 * time.Millisecond))
	if got := c.Remaining("white"); got != 600 {
		t.Fatalf("white remaining after 400ms: got %d want 600", got)
	}
	if got := c.Remaining("black"); got != 1000 {
		t.Fatalf("black should be unchanged: got %d want 1000", got)
	}

	c.Settle(start.Add(1000 * time.Millisecond))
	side, ok := c.Flagged()
	if !ok || side != "white" {
		t.Fatalf("Flagged: got (%q, %v) want (white, true)", side, ok)
	}
	if got := c.Remaining("white"); got != 0 {
		t.Fatalf("flagged remaining: got %d want 0", got)
	}
}

func TestClock_OnMoveAwardsIncrement(t *testing.T) {
	// 1s base + 30s increment; fast move (~200ms) → mover ends near 800+30000.
	c := NewClock(1000, 1000, 30_000)
	start := time.Unix(0, 0).UTC()
	c.Start("white", start)

	c.OnMove("white", start.Add(200*time.Millisecond))
	got := c.Remaining("white")
	want := int64(1000 - 200 + 30_000)
	if got != want {
		t.Fatalf("white after OnMove: got %d want %d", got, want)
	}
	if c.ActiveSide() != "black" {
		t.Fatalf("active after white move: got %q want black", c.ActiveSide())
	}
	if got := c.Remaining("black"); got != 1000 {
		t.Fatalf("black remaining: got %d want 1000", got)
	}
}

func TestClock_DisabledNoFlag(t *testing.T) {
	c := NewClock(0, 0, 0) // disabled by default when both bases are 0 / not enabled
	c.Enabled = false
	start := time.Unix(0, 0).UTC()
	c.Start("white", start)
	c.Settle(start.Add(time.Hour))
	if _, ok := c.Flagged(); ok {
		t.Fatal("disabled clock must not flag")
	}
}

func TestClock_AsymmetricInitial(t *testing.T) {
	c := NewClock(5*60*1000, 60*1000, 0)
	if c.Remaining("white") != 5*60*1000 || c.Remaining("black") != 60*1000 {
		t.Fatalf("asymmetric initial: white=%d black=%d", c.Remaining("white"), c.Remaining("black"))
	}
}
