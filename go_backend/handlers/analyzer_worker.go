// CM3070 FP code
// analyzer_worker.go - analysis queue, explain enqueue, and export

package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	sessionpkg "go_backend/game/session"
)

// noteExplainRequest - records that an explain should follow analysis for this fen
func noteExplainRequest(gameID string, moveNumber int) {
	analysisStoreMu.Lock()
	defer analysisStoreMu.Unlock()
	if moveNumber >= explainLatestByGame[gameID] {
		explainLatestByGame[gameID] = moveNumber
	}
}

// isExplainStale - reports whether explain stale
func isExplainStale(gameID string, moveNumber int) bool {
	analysisStoreMu.Lock()
	defer analysisStoreMu.Unlock()
	return moveNumber < explainLatestByGame[gameID]
}

// StartAnalyzerWorker - starts analyzer worker
func StartAnalyzerWorker() {
	analysisWorkerOnce.Do(func() {
		go analysisWorkerLoop()
		log.Printf("python analyzer worker started")
	})
}

// analysisWorkerLoop - drains the analysis queue and runs analyze/explain jobs
func analysisWorkerLoop() {
	for job := range analysisQueue {
		startedAt := time.Now()
		result, err := analyzeByRequest(job.Request)
		latencyMS := time.Since(startedAt).Milliseconds()
		analysisStoreMu.Lock()
		analysisPendingByGame[job.GameID] = false
		analysisLastErrorByGame[job.GameID] = ""
		latestRequestedMove := latestRequestedByGame[job.GameID]
		analysisStoreMu.Unlock()

		if err != nil {
			errorKind, httpStatus := analyzerErrorDetails(err)
			userSafe := analyzerUserSafeErrorByKind(errorKind)
			analysisStoreMu.Lock()
			analysisLastErrorByGame[job.GameID] = userSafe
			analysisStoreMu.Unlock()
			gameSocketHub.Broadcast(job.GameID, socketEventAnalysisStatus, map[string]interface{}{
				"status":                "error",
				"pending":               false,
				"requested_move_number": latestRequestedMove,
				"latest_move_number":    job.MoveNumber,
				"last_error":            userSafe,
				"error_kind":            errorKind,
				"http_status":           httpStatus,
				"request_id":            job.Request.RequestID,
			})
			emitAnalysisLog(analysisLogEvent{
				Event:               "analysis_failed",
				GameID:              job.GameID,
				MoveNumber:          job.MoveNumber,
				RequestID:           job.Request.RequestID,
				QueueLen:            job.EnqueueQueueLen,
				Pending:             false,
				Success:             false,
				LatencyMS:           latencyMS,
				ErrorKind:           errorKind,
				ErrorMessageSafe:    userSafe,
				LatestRequestedMove: latestRequestedMove,
				HTTPStatus:          httpStatus,
			})
			log.Printf("warning: analyzer job failed game_id=%s move=%d: %v", job.GameID, job.MoveNumber, err)
			// still coach the move (without MultiPV cues) so notes are not silent on analyzer failure.
			if shouldExplainCommand(job.Command) {
				uci, san := explainArgsFromAnalysis(job.Command, nil)
				enqueueExplanation(job.GameID, uci, san)
			}
			continue
		}
		if result == nil {
			if shouldExplainCommand(job.Command) {
				uci, san := explainArgsFromAnalysis(job.Command, nil)
				enqueueExplanation(job.GameID, uci, san)
			}
			continue
		}
		if job.MoveNumber < latestRequestedMove {
			emitAnalysisLog(analysisLogEvent{
				Event:               "analysis_stale_ignored",
				GameID:              job.GameID,
				MoveNumber:          job.MoveNumber,
				RequestID:           job.Request.RequestID,
				QueueLen:            job.EnqueueQueueLen,
				Pending:             false,
				Success:             false,
				LatencyMS:           latencyMS,
				ErrorKind:           analysisErrorKindNone,
				ErrorMessageSafe:    "",
				IsStale:             true,
				LatestRequestedMove: latestRequestedMove,
				AnalyzerSource:      result.Source,
				AnalyzerLatencyMS:   result.LatencyMS,
				BestMoveUCI:         result.BestMoveUCI,
			})
			log.Printf("stale analyzer response ignored: game_id=%s move=%d latest_requested=%d", job.GameID, job.MoveNumber, latestRequestedMove)
			continue
		}
		recordMoveAnalysisForGame(job.GameID, job.MoveNumber, job.Command, *result)
		// explain after MultiPV is cached for this FEN → non-empty concept_hints aligned with Suggested moves.
		if shouldExplainCommand(job.Command) {
			uci, san := explainArgsFromAnalysis(job.Command, result)
			enqueueExplanation(job.GameID, uci, san)
		}
		gameSocketHub.Broadcast(job.GameID, socketEventAnalysisStatus, map[string]interface{}{
			"status":                "ready",
			"pending":               false,
			"requested_move_number": latestRequestedMove,
			"latest_move_number":    job.MoveNumber,
			"last_error":            "",
			"request_id":            job.Request.RequestID,
			"analysis":              result,
		})
		emitAnalysisLog(analysisLogEvent{
			Event:               "analysis_completed",
			GameID:              job.GameID,
			MoveNumber:          job.MoveNumber,
			RequestID:           job.Request.RequestID,
			QueueLen:            job.EnqueueQueueLen,
			Pending:             false,
			Success:             true,
			LatencyMS:           latencyMS,
			ErrorKind:           analysisErrorKindNone,
			ErrorMessageSafe:    "",
			LatestRequestedMove: latestRequestedMove,
			AnalyzerSource:      result.Source,
			AnalyzerLatencyMS:   result.LatencyMS,
			BestMoveUCI:         result.BestMoveUCI,
		})
	}
}

// enqueueCurrentPositionAnalysis - enqueues current position analysis
func enqueueCurrentPositionAnalysis(gameID, command string) {
	gameType := "chess"
	if game, err := sessionpkg.GetGameSessionByID(gameID); err == nil && string(game.Type) != "" {
		gameType = string(game.Type)
	}
	history, err := sessionpkg.MoveHistoryByID(gameID)
	if err != nil {
		log.Printf("warning: enqueue analysis failed %s: %v", gameIDLabel(gameID), err)
		return
	}
	moveNumber := len(history)
	// diagram / position-only loads have no plies; use 1 so FE polling (target > 0) can wait
	if moveNumber == 0 && command == "diagram" {
		moveNumber = 1
	}
	fen, err := sessionpkg.CurrentFENByID(gameID)
	if err != nil {
		log.Printf("warning: enqueue analysis failed %s: %v", gameIDLabel(gameID), err)
		return
	}
	color, err := sessionpkg.CurrentTurnColorByID(gameID)
	if err != nil {
		log.Printf("warning: enqueue analysis failed %s: %v", gameIDLabel(gameID), err)
		return
	}
	job := analysisJob{
		GameID:     gameID,
		MoveNumber: moveNumber,
		Command:    command,
		Request: analyzerRequest{
			RequestID: fmt.Sprintf("%s-move-%d", gameID, moveNumber),
			FEN:       fen,
			Color:     color,
			TopK:      5,
			GameType:  gameType,
		},
	}

	analysisStoreMu.Lock()
	latestRequestedByGame[gameID] = moveNumber
	analysisPendingByGame[gameID] = true
	analysisLastErrorByGame[gameID] = ""
	queueLen := len(analysisQueue)
	analysisStoreMu.Unlock()
	job.EnqueueQueueLen = queueLen

	select {
	case analysisQueue <- job:
		gameSocketHub.Broadcast(gameID, socketEventAnalysisStatus, map[string]interface{}{
			"status":                "pending",
			"pending":               true,
			"requested_move_number": moveNumber,
			"request_id":            job.Request.RequestID,
		})
		emitAnalysisLog(analysisLogEvent{
			Event:               "analysis_enqueued",
			GameID:              gameID,
			MoveNumber:          moveNumber,
			RequestID:           job.Request.RequestID,
			QueueLen:            job.EnqueueQueueLen,
			Pending:             true,
			Success:             true,
			LatencyMS:           0,
			ErrorKind:           analysisErrorKindNone,
			ErrorMessageSafe:    "",
			LatestRequestedMove: moveNumber,
		})
	default:
		analysisStoreMu.Lock()
		analysisPendingByGame[gameID] = false
		analysisLastErrorByGame[gameID] = "Analysis queue is busy. Showing previous result."
		analysisStoreMu.Unlock()
		gameSocketHub.Broadcast(gameID, socketEventAnalysisStatus, map[string]interface{}{
			"status":                "error",
			"pending":               false,
			"requested_move_number": moveNumber,
			"latest_move_number":    moveNumber,
			"last_error":            "Analysis queue is busy. Showing previous result.",
			"error_kind":            analysisErrorKindUnavailable,
			"request_id":            job.Request.RequestID,
		})
		emitAnalysisLog(analysisLogEvent{
			Event:               "analysis_dropped_queue_full",
			GameID:              gameID,
			MoveNumber:          moveNumber,
			RequestID:           job.Request.RequestID,
			QueueLen:            job.EnqueueQueueLen,
			Pending:             false,
			Success:             false,
			LatencyMS:           0,
			ErrorKind:           analysisErrorKindUnavailable,
			ErrorMessageSafe:    "Analysis queue is busy. Showing previous result.",
			LatestRequestedMove: moveNumber,
		})
		log.Printf("warning: analyzer queue full, dropped job %s move=%d", gameIDLabel(gameID), moveNumber)
	}
}

// recordMoveAnalysis - records move analysis
func recordMoveAnalysis(command string, result analyzerResponse) {
	game := sessionpkg.GetGameSession()
	history := sessionpkg.GetMoveHistory()
	moveNumber := len(history)
	recordMoveAnalysisForGame(game.ID, moveNumber, command, result)
}

// shouldExplainCommand - reports whether this move should trigger coach explain
func shouldExplainCommand(command string) bool {
	c := strings.TrimSpace(strings.ToLower(command))
	return c != "" && c != "flag"
}

// explainArgsFromAnalysis - picks /explain move labels; diagram tips use best move (not the sentinel)
func explainArgsFromAnalysis(command string, result *analyzerResponse) (uci string, san string) {
	cmd := strings.TrimSpace(command)
	if !strings.EqualFold(cmd, "diagram") {
		return cmd, cmd
	}
	if result != nil {
		best := strings.TrimSpace(result.BestMoveUCI)
		if best != "" {
			uci, san = best, best
		}
		if len(result.SuggestedMoves) > 0 {
			if u := strings.TrimSpace(result.SuggestedMoves[0].UCI); u != "" {
				uci = u
			}
			if s := strings.TrimSpace(result.SuggestedMoves[0].SAN); s != "" {
				san = s
			} else if uci != "" {
				san = uci
			}
		}
		if uci != "" {
			if san == "" {
				san = uci
			}
			return uci, san
		}
	}
	return "position", "position"
}

// enqueueExplanation - calls python /explain asynchronously after analysis cues are ready for the same fen
func enqueueExplanation(gameID, moveUCI, moveSAN string) {
	go func() {
		gameType := "chess"
		skillLevel := "intermediate"
		humanColor := ""
		if game, err := sessionpkg.GetGameSessionByID(gameID); err == nil {
			if string(game.Type) != "" {
				gameType = string(game.Type)
			}
			skillLevel = sessionpkg.ResolveSkillLevel(game.Config.SkillLevel, game.Config.AIProfile)
			humanColor = strings.ToLower(strings.TrimSpace(game.Config.HumanColor))
		}
		history, err := sessionpkg.MoveHistoryByID(gameID)
		if err != nil {
			return
		}
		fen, err := sessionpkg.CurrentFENByID(gameID)
		if err != nil {
			return
		}
		color, err := sessionpkg.CurrentTurnColorByID(gameID)
		if err != nil {
			return
		}
		moveNumber := len(history)
		// tip / diagram loads have no plies; keep explain keys aligned with analysis move #1
		if moveNumber == 0 {
			moveNumber = 1
		}
		noteExplainRequest(gameID, moveNumber)

		hints := waitConceptHintsForFEN(gameID, fen, explainHintWait())
		if isExplainStale(gameID, moveNumber) {
			return
		}
		base := explainRequest{
			FEN:          fen,
			Color:        color,
			GameType:     gameType,
			SkillLevel:   skillLevel,
			HumanColor:   humanColor,
			ConceptHints: hints,
			MoveUCI:      moveUCI,
			MoveSAN:      moveSAN,
			MoveHistory:  history,
		}

		// 1) Instant ground-truth line (no Ollama) so notes are not stuck on "Thinking…" for ~8s.
		quickReq := base
		quickReq.RequestID = fmt.Sprintf("%s-explain-quick-%d", gameID, moveNumber)
		quickReq.Quick = true
		if quick, qerr := explainByRequest(quickReq); qerr == nil && quick != nil && strings.TrimSpace(quick.Explanation) != "" {
			if !isExplainStale(gameID, moveNumber) {
				gameSocketHub.Broadcast(gameID, socketEventExplanationReady, map[string]interface{}{
					"move_number":   moveNumber,
					"move_uci":      quick.MoveUCI,
					"move_san":      quick.MoveSAN,
					"explanation":   quick.Explanation,
					"source":        quick.Source,
					"latency_ms":    quick.LatencyMS,
					"skill_level":   skillLevel,
					"concept_hints": hints,
					"quick":         true,
				})
			}
		}

		// 2) Full Ollama coach — one at a time; skip if a newer ply already superseded this.
		explainOllamaSem <- struct{}{}
		defer func() { <-explainOllamaSem }()
		if isExplainStale(gameID, moveNumber) {
			return
		}
		req := base
		req.RequestID = fmt.Sprintf("%s-explain-%d", gameID, moveNumber)
		result, err := explainByRequest(req)
		if err != nil || result == nil || strings.TrimSpace(result.Explanation) == "" {
			return // graceful: keep quick line if any
		}
		if isExplainStale(gameID, moveNumber) {
			return
		}

		if len(result.ConceptHints) > 0 {
			hints = result.ConceptHints
		}
		recordMoveExplanationForGame(gameID, moveExplanationRecord{
			MoveNumber:   moveNumber,
			MoveUCI:      result.MoveUCI,
			MoveSAN:      result.MoveSAN,
			FEN:          fen,
			SkillLevel:   skillLevel,
			Source:       result.Source,
			Explanation:  result.Explanation,
			ConceptHints: append([]string(nil), hints...),
			LatencyMS:    result.LatencyMS,
			RecordedAt:   time.Now().UTC().Format(time.RFC3339),
		})

		gameSocketHub.Broadcast(gameID, socketEventExplanationReady, map[string]interface{}{
			"move_number":   moveNumber,
			"move_uci":      result.MoveUCI,
			"move_san":      result.MoveSAN,
			"explanation":   result.Explanation,
			"source":        result.Source,
			"latency_ms":    result.LatencyMS,
			"skill_level":   skillLevel,
			"concept_hints": hints,
		})
	}()
}

// recordMoveExplanationForGame - records move explanation for game
func recordMoveExplanationForGame(gameID string, entry moveExplanationRecord) {
	analysisStoreMu.Lock()
	defer analysisStoreMu.Unlock()
	moveExplanationByGame[gameID] = append(moveExplanationByGame[gameID], entry)
}

// recordMoveAnalysisForGame - records move analysis for game
func recordMoveAnalysisForGame(gameID string, moveNumber int, command string, result analyzerResponse) {
	analysisStoreMu.Lock()
	defer analysisStoreMu.Unlock()
	entry := moveAnalysisRecord{
		MoveNumber: moveNumber,
		Command:    command,
		Analysis:   result,
	}
	moveAnalysisByGame[gameID] = append(moveAnalysisByGame[gameID], entry)
	latestAnalysisByGame[gameID] = latestAnalysisState{
		GameID:     gameID,
		MoveNumber: moveNumber,
		Command:    command,
		Analysis:   result,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
}

// getLatestAnalysisByGameID - returns the latest analysis payload for a game
func getLatestAnalysisByGameID(gameID string) (latestAnalysisState, bool) {
	analysisStoreMu.Lock()
	defer analysisStoreMu.Unlock()
	entry, ok := latestAnalysisByGame[gameID]
	return entry, ok
}

// getLatestAnalysisStatusByGameID - returns latest analysis status metadata for a game
func getLatestAnalysisStatusByGameID(gameID string) analysisLatestStatus {
	analysisStoreMu.Lock()
	defer analysisStoreMu.Unlock()
	status := analysisLatestStatus{
		GameID:              gameID,
		RequestedMoveNumber: latestRequestedByGame[gameID],
		Pending:             analysisPendingByGame[gameID],
		LastError:           analysisLastErrorByGame[gameID],
	}
	if latest, ok := latestAnalysisByGame[gameID]; ok {
		latestCopy := latest
		status.LatestMoveNumber = latest.MoveNumber
		status.Latest = &latestCopy
	}
	return status
}

// analysisExportDir - resolves data/analysis_exports from the go.mod directory (stable under go test cwd)
func analysisExportDir() string {
	if v := strings.TrimSpace(os.Getenv("ANALYSIS_EXPORT_DIR")); v != "" {
		return v
	}
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Join("data", "analysis_exports")
	}
	dir := cwd
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			if resolved, resolveErr := filepath.EvalSymlinks(dir); resolveErr == nil {
				dir = resolved
			}
			return filepath.Join(dir, "data", "analysis_exports")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("data", "analysis_exports")
}

// exportGameAnalysisIfNeeded - exports game analysis if needed
func exportGameAnalysisIfNeeded(game sessionpkg.GameSession) {
	if game.Result == sessionpkg.GameResultInProgress {
		return
	}

	analysisStoreMu.Lock()
	if exportedGames[game.ID] {
		analysisStoreMu.Unlock()
		return
	}
	records := append([]moveAnalysisRecord(nil), moveAnalysisByGame[game.ID]...)
	explains := append([]moveExplanationRecord(nil), moveExplanationByGame[game.ID]...)
	exportedGames[game.ID] = true
	analysisStoreMu.Unlock()

	payload := struct {
		GameID       string                  `json:"game_id"`
		Result       sessionpkg.GameResult   `json:"result"`
		Game         sessionpkg.GameSession  `json:"game"`
		History      []string                `json:"history"`
		MoveAnalysis []moveAnalysisRecord    `json:"move_analysis"`
		Explanations []moveExplanationRecord `json:"explanations"`
		ExportedAt   string                  `json:"exported_at"`
	}{
		GameID:       game.ID,
		Result:       game.Result,
		Game:         game,
		History:      historyByGameID(game.ID),
		MoveAnalysis: records,
		Explanations: explains,
		ExportedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	outputDir := analysisExportDir()
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Printf("warning: failed to create analysis export directory: %v", err)
		return
	}

	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.json", game.ID))
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Printf("warning: failed to marshal analysis export: %v", err)
		return
	}
	if err := os.WriteFile(outputPath, raw, 0o644); err != nil {
		log.Printf("warning: failed to write analysis export: %v", err)
		return
	}
	log.Printf("analysis export saved: %s", outputPath)
}

// historyByGameID - history by game id
func historyByGameID(gameID string) []string {
	history, err := sessionpkg.MoveHistoryByID(gameID)
	if err != nil {
		return []string{}
	}
	return history
}
