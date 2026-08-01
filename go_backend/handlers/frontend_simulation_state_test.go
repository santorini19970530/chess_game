// CM3070 FP code
// frontend_simulation_state_test.go - tests for frontend simulation state

package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadChessCommandSource - returns chess command source
func loadChessCommandSource(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "..", "frontend", "scripts", "chess_command.js"),
		filepath.Join("..", "frontend", "scripts", "chess_command.js"),
		filepath.Join("frontend", "scripts", "chess_command.js"),
	}

	return loadFrontendSource(t, candidates, "chess_command.js")
}

// loadInputCSSSource - returns input css source
func loadInputCSSSource(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "..", "frontend", "styles", "input.css"),
		filepath.Join("..", "frontend", "styles", "input.css"),
		filepath.Join("frontend", "styles", "input.css"),
	}

	return loadFrontendSource(t, candidates, "input.css")
}

// loadSimulationCSSSource - returns simulation stylesheet (download button rules live here)
func loadSimulationCSSSource(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "..", "frontend", "styles", "css_parts", "simulation.css"),
		filepath.Join("..", "frontend", "styles", "css_parts", "simulation.css"),
		filepath.Join("frontend", "styles", "css_parts", "simulation.css"),
	}

	return loadFrontendSource(t, candidates, "simulation.css")
}

// loadIndexHandlerSource - returns index handler source
func loadIndexHandlerSource(t *testing.T) string {
	t.Helper()

	candidates := []string{
		"index.go",
		filepath.Join("handlers", "index.go"),
	}

	return loadFrontendSource(t, candidates, "index.go")
}

// loadFrontendSource - returns frontend source
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

// requireSnippet - returns require snippet
func requireSnippet(t *testing.T, source string, snippet string) {
	t.Helper()
	if strings.Contains(source, snippet) {
		return
	}
	t.Fatalf("expected frontend simulation logic snippet missing: %q", snippet)
}

// TestFrontendSimulationState_RunPlaybackDoneMarkers - checks frontend simulation state run playback done markers
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

// TestFrontendSimulationState_ErrorAndConflictRecoveryMarkers - checks frontend simulation state error and conflict recovery markers
func TestFrontendSimulationState_ErrorAndConflictRecoveryMarkers(t *testing.T) {
	source := loadChessCommandSource(t)

	requireSnippet(t, source, "if (resp.status === 409) {")
	requireSnippet(t, source, "Simulation already running on server.")
	requireSnippet(t, source, "Simulation failed: ")
	requireSnippet(t, source, "Simulation failed: missing results payload.")
	requireSnippet(t, source, `setCatchStatus(error, "Network error while loading simulation.");`)

	// if cleanup is removed from error paths, UI can get stuck in playback mode
	if strings.Count(source, "cleanupSimulationControls();") < 4 {
		t.Fatalf("expected multiple cleanupSimulationControls() calls across error and done paths")
	}
}

// TestFrontendSimulationState_BusyGuardMarkers - checks frontend simulation state busy guard markers
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

// TestFrontendSimulationDownload_Step1ButtonMarkers - checks frontend simulation download step1 button markers
func TestFrontendSimulationDownload_Step1ButtonMarkers(t *testing.T) {
	source := loadIndexHandlerSource(t)

	requireSnippet(t, source, `id="simulation_download_json_btn"`)
	requireSnippet(t, source, `id="simulation_download_csv_btn"`)
	requireSnippet(t, source, `class="simulation_download_actions"`)
	requireSnippet(t, source, "Download JSON")
	requireSnippet(t, source, "Download CSV")
}

// TestFrontendShogiBoard_NumericFileLabels - checks frontend shogi board numeric file labels
func TestFrontendShogiBoard_NumericFileLabels(t *testing.T) {
	jsSrc := loadChessCommandSource(t)
	requireSnippet(t, jsSrc, `const numericFiles = boardGameType === "shogi"`)
	requireSnippet(t, jsSrc, "? String(i + 1)")
}

// TestFrontendConfigPanel_DetailsCollapse - checks frontend config panel details collapse
func TestFrontendConfigPanel_DetailsCollapse(t *testing.T) {
	indexSrc := loadIndexHandlerSource(t)
	requireSnippet(t, indexSrc, `id="game_config_details"`)
	requireSnippet(t, indexSrc, `class="game_config_details"`)
	requireSnippet(t, indexSrc, `<summary class="config_panel_title">Setup new game</summary>`)

	jsSrc := loadChessCommandSource(t)
	requireSnippet(t, jsSrc, "collapseConfigPanel")
	requireSnippet(t, jsSrc, "game_config_details")
}

// TestFrontendLoadMoves_ReviewMarkers - checks frontend load moves review markers
func TestFrontendLoadMoves_ReviewMarkers(t *testing.T) {
	indexSrc := loadIndexHandlerSource(t)
	requireSnippet(t, indexSrc, `id="review_moves_input"`)
	requireSnippet(t, indexSrc, `id="review_moves_load"`)
	requireSnippet(t, indexSrc, `id="review_moves_file"`)
	requireSnippet(t, indexSrc, `id="review_moves_prev"`)
	requireSnippet(t, indexSrc, `id="review_moves_next"`)

	jsSrc := loadChessCommandSource(t)
	requireSnippet(t, jsSrc, "/load-moves")
	requireSnippet(t, jsSrc, "applyLoadedGameSnapshot")
	requireSnippet(t, jsSrc, "review_moves_load")
	requireSnippet(t, jsSrc, "seekReviewPlayback")
	requireSnippet(t, jsSrc, "reviewPlaybackMoves")
}

// TestFrontendSimulationDownload_Step2StyleMarkers - checks frontend simulation download step2 style markers
func TestFrontendSimulationDownload_Step2StyleMarkers(t *testing.T) {
	source := loadSimulationCSSSource(t)

	requireSnippet(t, source, ".simulation_download_actions")
	requireSnippet(t, source, ".simulation_download_btn")
	requireSnippet(t, source, "#simulation_download_json_btn[disabled]")
	requireSnippet(t, source, "#simulation_download_csv_btn[disabled]")
}

// TestFrontendSimulationDownload_ExportMarkers - checks frontend simulation download export markers
func TestFrontendSimulationDownload_ExportMarkers(t *testing.T) {
	source := loadChessCommandSource(t)

	requireSnippet(t, source, "buildSimulationExportPayload")
	requireSnippet(t, source, "buildSimulationJSON")
	requireSnippet(t, source, "buildSimulationCSV")
	requireSnippet(t, source, "downloadTextFile")
	requireSnippet(t, source, "setSimulationDownloadEnabled")
}
