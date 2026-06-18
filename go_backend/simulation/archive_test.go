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
		GameID:    "game-test-1",
		Profile:   "beginner",
		Result:    session.GameResultDraw,
		Winner:    "",
		MoveCount: 10,
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
	if !strings.HasSuffix(matches[0], "game-test-1.json") {
		t.Fatalf("unexpected archive path: %s", matches[0])
	}
}
