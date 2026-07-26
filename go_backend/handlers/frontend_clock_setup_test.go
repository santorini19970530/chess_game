package handlers

import (
	"strings"
	"testing"
)

func TestFrontendClockSetup_FormAndDisplayMarkers(t *testing.T) {
	source := loadChessCommandSource(t)
	for _, snippet := range []string{
		`getElementById("clock_enabled")`,
		`getElementById("game_info_time_white")`,
		`getElementById("game_info_time_black")`,
		"const appendClockFields = ",
		"const renderClocks = ",
		"const applyServerClock = ",
		"const startClockTick = ",
		"const stopClockTick = ",
		"const flagOnLocalTimeout = ",
		"const syncClockControlsFromGame = ",
		`params.set("clockEnabled"`,
		`"humanInitialMs"`,
		`params.set("whiteInitialMs"`,
		"return appendClockFields(params);",
		"renderClocks(result.game);",
		"syncClockControlsFromGame(game)",
		"syncClockSetup",
		"void refreshGameSnapshotFromAPI(targetGameId);",
		"/flag`",
		"flagOnLocalTimeout",
	} {
		if !strings.Contains(source, snippet) {
			t.Fatalf("chess_command.js missing %q", snippet)
		}
	}
	if strings.Count(source, "renderClocks(result.game);") < 4 {
		t.Fatalf("expected renderClocks on create/move/refresh/new (or flag) paths")
	}
}
