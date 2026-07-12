package simulation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	session "go_backend/game/session"
)

func TestSimulationArchiveRoot_UsesGoModDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	modDir := filepath.Join(t.TempDir(), "module")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatalf("mkdir module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.Chdir(modDir); err != nil {
		t.Fatalf("chdir module: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(modDir); err == nil {
		modDir = resolved
	}

	got := simulationArchiveRoot()
	want := filepath.Join(modDir, "data", "simulations")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSimulationArchiveRoot_RespectsEnvOverride(t *testing.T) {
	t.Setenv("SIMULATION_ARCHIVE_DIR", "/tmp/custom-sim-archive")
	if got := simulationArchiveRoot(); got != "/tmp/custom-sim-archive" {
		t.Fatalf("expected env override, got %q", got)
	}
}

func TestArchiveSimulationRun_WritesUnderModuleRoot(t *testing.T) {
	archiveDir := t.TempDir()
	t.Setenv("SIMULATION_ARCHIVE_DIR", archiveDir)

	err := ArchiveSimulationRun([]ResultWithGameID{{
		GameID:       "game-test-1",
		GameType:     "chess",
		WhiteProfile: "beginner",
		BlackProfile: "master",
		Result:       session.GameResultDraw,
		Winner:       "",
		MoveCount:    10,
		DurationMs:   1500,
		AvgMoveMs:    150,
	}})
	if err != nil {
		t.Fatalf("archive run failed: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(archiveDir, "*", "game-test-1.json"))
	if err != nil {
		t.Fatalf("glob archive file: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one archived game file, got %d (%v)", len(matches), matches)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	body := string(raw)
	for _, want := range []string{
		`"white_profile": "beginner"`,
		`"black_profile": "master"`,
		`"game_type": "chess"`,
		`"duration_ms": 1500`,
		`"avg_move_ms": 150`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("archive missing %s\n%s", want, body)
		}
	}
	if strings.Contains(body, `"profile"`) {
		t.Fatalf("expected profile omitted when white!=black\n%s", body)
	}
}
