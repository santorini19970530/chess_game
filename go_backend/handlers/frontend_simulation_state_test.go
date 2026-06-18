package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadChessCommandSource(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "..", "frontend", "scripts", "chess_command.js"),
		filepath.Join("..", "frontend", "scripts", "chess_command.js"),
		filepath.Join("frontend", "scripts", "chess_command.js"),
	}

	return loadFrontendSource(t, candidates, "chess_command.js")
}

func loadIndexHandlerSource(t *testing.T) string {
	t.Helper()

	candidates := []string{
		"index.go",
		filepath.Join("handlers", "index.go"),
	}

	return loadFrontendSource(t, candidates, "index.go")
}

func loadFrontendSource(t *testing.T, candidates []string, label string) string {
	t.Helper()

	var lastErr error
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return string(data)
		}
		lastErr = err
	}

	t.Fatalf("failed to load %s for regression test: %v", label, lastErr)
	return ""
}

func requireSnippet(t *testing.T, source string, snippet string) {
	t.Helper()
	if strings.Contains(source, snippet) {
		return
	}
	t.Fatalf("expected frontend simulation logic snippet missing: %q", snippet)
}

func TestFrontendSimulationState_RunPlaybackDoneMarkers(t *testing.T) {
	source := loadChessCommandSource(t)

	requireSnippet(t, source, `simRunBtn.style.display = "none";`)
	requireSnippet(t, source, "ensureSimulationControls();")
	requireSnippet(t, source, "startNextSimulationGame();")
	requireSnippet(t, source, "if (currentSimMoveIdx >= moves.length) {")
	requireSnippet(t, source, "finishCurrentSimulationGame();")
	requireSnippet(t, source, "if (isLastGame) {")
	requireSnippet(t, source, "cleanupSimulationControls();")
	requireSnippet(t, source, `simRunBtn.style.display = "inline-block";`)
	requireSnippet(t, source, `simRunBtn.disabled = false;`)
	requireSnippet(t, source, "if (!isAIVsAIModeSelected()) {")
	requireSnippet(t, source, `simRunBtn.style.display = "none";`)
}

func TestFrontendSimulationState_ErrorAndConflictRecoveryMarkers(t *testing.T) {
	source := loadChessCommandSource(t)

	requireSnippet(t, source, "if (resp.status === 409) {")
	requireSnippet(t, source, "Simulation already running on server.")
	requireSnippet(t, source, "Simulation failed: ")
	requireSnippet(t, source, "Simulation failed: missing results payload.")
	requireSnippet(t, source, "setStatus(\"Network error while loading simulation.\", \"error\");")

	// ponytail: if cleanup call is removed from any error path, UI can get stuck in playback mode.
	if strings.Count(source, "cleanupSimulationControls();") < 4 {
		t.Fatalf("expected multiple cleanupSimulationControls() calls across error and done paths")
	}
}

func TestFrontendSimulationState_BusyGuardMarkers(t *testing.T) {
	source := loadChessCommandSource(t)

	requireSnippet(t, source, "const simulationBusy = simulationRequestInFlight || isSimulationPlayback;")
	requireSnippet(t, source, "if (newGameButton) newGameButton.disabled = simulationBusy;")
	requireSnippet(t, source, "if (configApplyButton) configApplyButton.disabled = simulationBusy;")
	requireSnippet(t, source, "if (button) button.disabled = simulationBusy || gameOver;")
	requireSnippet(t, source, "if (flagButton) flagButton.disabled = simulationBusy || gameOver;")
	requireSnippet(t, source, "Please enter an integer game count between 1 and 1000.")

	if strings.Count(source, "Simulation is in progress. Please wait for it to finish.") < 3 {
		t.Fatalf("expected simulation-in-progress guard message in multiple action handlers")
	}
}

func TestFrontendSimulationDownload_Step1ButtonMarkers(t *testing.T) {
	source := loadIndexHandlerSource(t)

	requireSnippet(t, source, `id="simulation_download_json_btn"`)
	requireSnippet(t, source, `id="simulation_download_csv_btn"`)
	requireSnippet(t, source, `class="simulation_download_actions"`)
	requireSnippet(t, source, "Download JSON")
	requireSnippet(t, source, "Download CSV")
}
