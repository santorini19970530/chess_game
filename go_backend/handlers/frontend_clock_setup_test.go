// CM3070 FP code
// frontend_clock_setup_test.go - tests for frontend clock setup

package handlers

import (
	"strings"
	"testing"
)

// TestFrontendClockSetup_FormAndDisplayMarkers - checks frontend clock setup form and display markers
func TestFrontendClockSetup_FormAndDisplayMarkers(t *testing.T) {
	source := loadChessCommandSource(t)
	for _, snippet := range []string{
		`getElementById("clock_enabled")`,
		`getElementById("game_info_time_white")`,
		`getElementById("game_info_time_black")`,
		"class ClockController",
		"appendClockFields(params)",
		"renderClocks(game)",
		"applyServerClock(clk, remaining)",
		"startClockTick()",
		"stopClockTick()",
		"flagOnLocalTimeout()",
		"syncClockControlsFromGame(game)",
		`params.set("clockEnabled"`,
		`"humanInitialMs"`,
		`params.set("whiteInitialMs"`,
		"return this.app.clocks.appendClockFields(params);",
		"this.clocks.renderClocks(result.game);",
		"applyGameSnapshot(result, opts = {})",
		"syncClockControlsFromGame(game)",
		"syncClockSetup",
		"void this.refreshGameSnapshotFromAPI(targetGameId);",
		"/flag`",
		"flagOnLocalTimeout",
	} {
		if !strings.Contains(source, snippet) {
			t.Fatalf("chess_command.js missing %q", snippet)
		}
	}
	// create/move/flag/new/refresh/review all paint clocks through applyGameSnapshot
	if strings.Count(source, "applyGameSnapshot(") < 4 {
		t.Fatalf("expected applyGameSnapshot on create/move/refresh/new (or flag) paths")
	}
}
