package session

import (
	"strings"
	"time"
)

// ClockSidesFromHumanAI - maps HvAI human/AI bases onto white/black
func ClockSidesFromHumanAI(humanColor string, humanMs, aiMs int64) (whiteMs, blackMs int64) {
	if strings.ToLower(strings.TrimSpace(humanColor)) == "black" {
		return aiMs, humanMs
	}
	return humanMs, aiMs
}

// clock is a Fischer time-control clock (server-authoritative).
// mode "fischer" now; byoyomi can reuse this type later.
type Clock struct {
	Enabled          bool   `json:"enabled"`
	Mode             string `json:"mode"` // "fischer"
	WhiteInitialMs   int64  `json:"whiteInitialMs"`
	BlackInitialMs   int64  `json:"blackInitialMs"`
	WhiteRemainingMs int64  `json:"whiteRemainingMs"`
	BlackRemainingMs int64  `json:"blackRemainingMs"`
	IncrementMs      int64  `json:"incrementMs"`
	Active           string `json:"active"` // "white" | "black"
	LastTickUnixMs   int64  `json:"lastTickUnixMs,omitempty"`
	running          bool
	flaggedSide      string
}

// NewClock - builds an enabled Fischer clock when either base is > 0. both bases 0 → disabled (unlimited)
func NewClock(whiteInitialMs, blackInitialMs, incrementMs int64) *Clock {
	enabled := whiteInitialMs > 0 || blackInitialMs > 0
	return &Clock{
		Enabled:          enabled,
		Mode:             "fischer",
		WhiteInitialMs:   whiteInitialMs,
		BlackInitialMs:   blackInitialMs,
		WhiteRemainingMs: whiteInitialMs,
		BlackRemainingMs: blackInitialMs,
		IncrementMs:      incrementMs,
	}
}

// ActiveSide - returns active side
func (c *Clock) ActiveSide() string {
	if c == nil {
		return ""
	}
	return c.Active
}

// Start - starts the operation
func (c *Clock) Start(active string, now time.Time) {
	if c == nil || !c.Enabled {
		return
	}
	c.Active = normalizeClockSide(active)
	c.LastTickUnixMs = now.UnixMilli()
	c.running = true
	c.flaggedSide = ""
}

// Remaining - returns remaining
func (c *Clock) Remaining(side string) int64 {
	if c == nil {
		return 0
	}
	switch normalizeClockSide(side) {
	case "white":
		return c.WhiteRemainingMs
	case "black":
		return c.BlackRemainingMs
	default:
		return 0
	}
}

// Flagged - performs flagged
func (c *Clock) Flagged() (side string, ok bool) {
	if c == nil || !c.Enabled || c.flaggedSide == "" {
		return "", false
	}
	return c.flaggedSide, true
}

// Settle - deducts elapsed time from the active side since LastTick
func (c *Clock) Settle(now time.Time) {
	if c == nil || !c.Enabled || !c.running || c.flaggedSide != "" {
		return
	}
	elapsed := now.UnixMilli() - c.LastTickUnixMs
	if elapsed < 0 {
		elapsed = 0
	}
	c.debit(c.Active, elapsed)
	c.LastTickUnixMs = now.UnixMilli()
}

// OnMove - settles, awards Fischer increment to the mover, then starts the opponent
func (c *Clock) OnMove(mover string, now time.Time) {
	if c == nil || !c.Enabled || c.flaggedSide != "" {
		return
	}
	c.Settle(now)
	if c.flaggedSide != "" {
		return
	}
	side := normalizeClockSide(mover)
	c.credit(side, c.IncrementMs)
	c.Active = opponentClockSide(side)
	c.LastTickUnixMs = now.UnixMilli()
	c.running = true
}

// debit - debits the operation
func (c *Clock) debit(side string, ms int64) {
	if ms <= 0 {
		return
	}
	switch normalizeClockSide(side) {
	case "white":
		c.WhiteRemainingMs -= ms
		if c.WhiteRemainingMs <= 0 {
			c.WhiteRemainingMs = 0
			c.flaggedSide = "white"
			c.running = false
		}
	case "black":
		c.BlackRemainingMs -= ms
		if c.BlackRemainingMs <= 0 {
			c.BlackRemainingMs = 0
			c.flaggedSide = "black"
			c.running = false
		}
	}
}

// credit - credits the operation
func (c *Clock) credit(side string, ms int64) {
	if ms <= 0 {
		return
	}
	switch normalizeClockSide(side) {
	case "white":
		c.WhiteRemainingMs += ms
	case "black":
		c.BlackRemainingMs += ms
	}
}

// normalizeClockSide - normalizes clock side
func normalizeClockSide(side string) string {
	switch side {
	case "white", "black":
		return side
	default:
		return ""
	}
}

// opponentClockSide - returns opponent clock side
func opponentClockSide(side string) string {
	if side == "white" {
		return "black"
	}
	if side == "black" {
		return "white"
	}
	return ""
}
