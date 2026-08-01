package engine

import (
	"strings"
	"testing"
	"time"
)

// TestBuildGoCmd_ClockOffUsesMovetime - checks build go cmd clock off uses movetime
func TestBuildGoCmd_ClockOffUsesMovetime(t *testing.T) {
	fs := &FairyStockfish{}
	cmd := fs.buildGoCmd(Limit{Depth: 8, MoveTime: 600 * time.Millisecond})
	if cmd != "go movetime 600" {
		t.Fatalf("got %q", cmd)
	}
}

// TestBuildGoCmd_ClockOnUsesWtimeBtimeInc - checks build go cmd clock on uses wtime btime inc
func TestBuildGoCmd_ClockOnUsesWtimeBtimeInc(t *testing.T) {
	fs := &FairyStockfish{}
	cmd := fs.buildGoCmd(Limit{
		MoveTime:  200 * time.Millisecond, // cap
		WhiteTime: 300_000 * time.Millisecond,
		BlackTime: 60_000 * time.Millisecond,
		WhiteInc:  30_000 * time.Millisecond,
		BlackInc:  30_000 * time.Millisecond,
	})
	if !strings.Contains(cmd, "wtime 300000") || !strings.Contains(cmd, "btime 60000") {
		t.Fatalf("missing wtime/btime: %q", cmd)
	}
	if !strings.Contains(cmd, "winc 30000") || !strings.Contains(cmd, "binc 30000") {
		t.Fatalf("missing winc/binc: %q", cmd)
	}
	if !strings.Contains(cmd, "movetime 200") {
		t.Fatalf("expected movetime cap: %q", cmd)
	}
	if !strings.HasPrefix(cmd, "go ") {
		t.Fatalf("prefix: %q", cmd)
	}
}
