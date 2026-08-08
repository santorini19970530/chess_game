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

	roots := []string{
		filepath.Join("..", "..", "frontend", "scripts"),
		filepath.Join("..", "frontend", "scripts"),
		filepath.Join("frontend", "scripts"),
	}
	// same order as frontend/html_puzzles/game_panel.html script tags
	partNames := []string{
		"game_app.js",
		"dom_state.js",
		"util.js",
		"socket.js",
		"clocks.js",
		"board.js",
		"interaction.js",
		"promotion.js",
		"hints_coach.js",
		"game_info.js",
		"move_history.js",
		"setup_command.js",
		"session_actions.js",
		"review.js",
		"diagram_import.js",
		"simulation.js",
	}
	for _, root := range roots {
		partsDir := filepath.Join(root, "js_parts")
		if _, err := os.Stat(filepath.Join(partsDir, partNames[0])); err != nil {
			continue
		}
		var source strings.Builder
		for _, name := range partNames {
			partFile := filepath.Join(partsDir, name)
			data, err := os.ReadFile(partFile)
			if err != nil {
				t.Fatalf("failed to load frontend script part %s: %v", partFile, err)
			}
			source.Write(data)
			source.WriteByte('\n')
		}
		data, err := os.ReadFile(filepath.Join(root, "chess_command.js"))
		if err != nil {
			t.Fatalf("failed to load frontend boot script: %v", err)
		}
		source.Write(data)
		return source.String()
	}

	t.Fatal("failed to load frontend script parts for regression test")
	return ""
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

// loadIndexHandlerSource - returns concatenated game-panel html puzzles (markup lives in templates now)
func loadIndexHandlerSource(t *testing.T) string {
	t.Helper()

	puzzleNames := []string{
		"game_panel.html",
		"game_config.html",
		"game_info.html",
		"game_play.html",
	}
	var b strings.Builder
	for _, name := range puzzleNames {
		candidates := []string{
			filepath.Join("..", "..", "frontend", "html_puzzles", name),
			filepath.Join("..", "frontend", "html_puzzles", name),
			filepath.Join("frontend", "html_puzzles", name),
		}
		b.WriteString(loadFrontendSource(t, candidates, name))
		b.WriteByte('\n')
	}
	return b.String()
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
	canonicalSource := strings.NewReplacer(
		"G.state.", "", "G.el.", "", "G.",
		"", "this.app.state.", "", "this.app.el.", "", "this.app.",
		"",
	).Replace(source)
	if strings.Contains(source, snippet) || strings.Contains(canonicalSource, snippet) {
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
	requireSnippet(t, source, "loadCurrentSimGameIntoReview")
	requireSnippet(t, source, "finishCurrentSimulationGame();")
	requireSnippet(t, source, "if (isLastGame) {")
	requireSnippet(t, source, "cleanupSimulationControls();")
	requireSnippet(t, source, `simRunBtn.style.display = "inline-block";`)
	requireSnippet(t, source, `simRunBtn.disabled = false;`)
	requireSnippet(t, source, "if (!this.app.util.isAIVsAIModeSelected()) {")
	requireSnippet(t, source, `simRunBtn.style.display = "none";`)
	// Sim playback reuses review Back/Forward (no separate Next Move stepper).
	requireSnippet(t, source, "seekReviewPlayback")
	if strings.Contains(source, "sim_next_move_btn") || strings.Contains(source, "playNextSimulationMove") {
		t.Fatal("ai-vs-ai playback must use review Back/Forward, not a separate Next Move button")
	}
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

// TestFrontendConfigPanel_AlwaysVisibleLeft - checks setup panel is a static left column, not a details toggle
func TestFrontendConfigPanel_AlwaysVisibleLeft(t *testing.T) {
	indexSrc := loadIndexHandlerSource(t)
	requireSnippet(t, indexSrc, `id="game_config_panel"`)
	requireSnippet(t, indexSrc, `class="game_config_panel"`)
	requireSnippet(t, indexSrc, `<div class="config_panel_title">Setup new game</div>`)
	if strings.Contains(indexSrc, `id="game_config_details"`) || strings.Contains(indexSrc, `<summary class="config_panel_title">Setup new game</summary>`) {
		t.Fatal("setup panel must not use details/summary collapse")
	}

	jsSrc := loadChessCommandSource(t)
	if strings.Contains(jsSrc, "collapseConfigPanel") || strings.Contains(jsSrc, "game_config_details") {
		t.Fatal("chess_command.js must not collapse the setup panel")
	}
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

// TestFrontendDiagramImport_Markers - checks diagram import confirm-load ui markers
func TestFrontendDiagramImport_Markers(t *testing.T) {
	indexSrc := loadIndexHandlerSource(t)
	requireSnippet(t, indexSrc, `id="diagram_import_file"`)
	requireSnippet(t, indexSrc, `id="diagram_import_recognize"`)
	requireSnippet(t, indexSrc, `id="diagram_import_confirm"`)
	requireSnippet(t, indexSrc, `id="diagram_import_confirm_btn"`)
	requireSnippet(t, indexSrc, `id="diagram_import_cancel_btn"`)

	jsSrc := loadChessCommandSource(t)
	requireSnippet(t, jsSrc, "/api/diagram/fen")
	requireSnippet(t, jsSrc, "/load-fen")
	requireSnippet(t, jsSrc, "DiagramImport")
	requireSnippet(t, jsSrc, "confirmLoad")
	requireSnippet(t, jsSrc, "analysisMoveNumber")
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

// TestFrontendGameAppClasses - checks the frontend uses class-based controllers
func TestFrontendGameAppClasses(t *testing.T) {
	source := loadChessCommandSource(t)

	for _, snippet := range []string{
		"class GameApp",
		"class DomState",
		"class Util",
		"class SocketClient",
		"class ClockController",
		"class BoardView",
		"class BoardInteraction",
		"class PromotionPicker",
		"class HintsCoach",
		"class GameInfoView",
		"class SetupCommand",
		"class SessionActions",
		"class ReviewPlayback",
		"class SimulationPanel",
		"new GameApp()",
	} {
		requireSnippet(t, source, snippet)
	}
	if strings.Contains(source, "G.register") || strings.Contains(source, "G.parts") {
		t.Fatal("frontend must not use the GameUI callback registry")
	}
}
