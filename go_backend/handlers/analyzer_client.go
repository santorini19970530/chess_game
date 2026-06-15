package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	sessionpkg "go_backend/game/session"
)

var pyAnalyzerHTTPClient = &http.Client{
	Timeout: 0,
}

type analyzerRequest struct {
	RequestID string `json:"request_id"`
	FEN       string `json:"fen"`
	Color     string `json:"color"`
	TopK      int    `json:"top_k"`
}

type analysisJob struct {
	GameID          string
	MoveNumber      int
	Command         string
	Request         analyzerRequest
	EnqueueQueueLen int
}

type analyzerSuggestedMove struct {
	Rank  int    `json:"rank"`
	UCI   string `json:"uci"`
	SAN   string `json:"san"`
	Score int    `json:"score"`
}

type analyzerResponse struct {
	RequestID      string                  `json:"request_id"`
	Status         string                  `json:"status"`
	Source         string                  `json:"source"`
	FEN            string                  `json:"fen"`
	EvaluatedColor string                  `json:"evaluated_for_color"`
	HealthSummary  map[string]interface{}  `json:"health_summary"`
	IsCheck        bool                    `json:"is_check"`
	IsCheckmate    bool                    `json:"is_checkmate"`
	IsStalemate    bool                    `json:"is_stalemate"`
	EvalCPWhite    int                     `json:"eval_cp_white"`
	WinChanceWhite float64                 `json:"win_chance_white"`
	WinChanceBlack float64                 `json:"win_chance_black"`
	ThreatSummary  string                  `json:"threat_summary"`
	BestMoveUCI    string                  `json:"best_move_uci"`
	SuggestedMoves []analyzerSuggestedMove `json:"suggested_moves"`
	LatencyMS      int                     `json:"latency_ms"`
}

type moveAnalysisRecord struct {
	MoveNumber int              `json:"move_number"`
	Command    string           `json:"command"`
	Analysis   analyzerResponse `json:"analysis"`
}

type latestAnalysisState struct {
	GameID     string           `json:"game_id"`
	MoveNumber int              `json:"move_number"`
	Command    string           `json:"command"`
	Analysis   analyzerResponse `json:"analysis"`
	UpdatedAt  string           `json:"updated_at"`
}

type analysisLatestStatus struct {
	GameID              string               `json:"game_id"`
	RequestedMoveNumber int                  `json:"requested_move_number"`
	LatestMoveNumber    int                  `json:"latest_move_number"`
	Pending             bool                 `json:"pending"`
	LastError           string               `json:"last_error,omitempty"`
	Latest              *latestAnalysisState `json:"latest,omitempty"`
}

type analyzerCallError struct {
	Kind       string
	HTTPStatus int
	Err        error
}

func (e *analyzerCallError) Error() string {
	if e == nil || e.Err == nil {
		return "analyzer call error"
	}
	return e.Err.Error()
}

func (e *analyzerCallError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type analysisLogEvent struct {
	Event               string `json:"event"`
	GameID              string `json:"game_id"`
	MoveNumber          int    `json:"move_number"`
	RequestID           string `json:"request_id"`
	QueueLen            int    `json:"queue_len"`
	Pending             bool   `json:"pending"`
	Success             bool   `json:"success"`
	LatencyMS           int64  `json:"latency_ms"`
	ErrorKind           string `json:"error_kind"`
	ErrorMessageSafe    string `json:"error_message_safe"`
	TimestampUTC        string `json:"timestamp_utc"`
	IsStale             bool   `json:"is_stale"`
	LatestRequestedMove int    `json:"latest_requested_move"`
	HTTPStatus          int    `json:"http_status,omitempty"`
	AnalyzerSource      string `json:"analyzer_source,omitempty"`
	AnalyzerLatencyMS   int    `json:"analyzer_latency_ms,omitempty"`
	BestMoveUCI         string `json:"best_move_uci,omitempty"`
}

var (
	moveAnalysisByGame      = map[string][]moveAnalysisRecord{}
	latestAnalysisByGame    = map[string]latestAnalysisState{}
	latestRequestedByGame   = map[string]int{}
	analysisPendingByGame   = map[string]bool{}
	analysisLastErrorByGame = map[string]string{}
	exportedGames           = map[string]bool{}
	analysisStoreMu         sync.Mutex
	analysisQueue           = make(chan analysisJob, 128)
	analysisWorkerOnce      sync.Once
)

const (
	analysisErrorKindNone        = "none"
	analysisErrorKindTimeout     = "timeout"
	analysisErrorKindUnavailable = "unavailable"
	analysisErrorKindBadStatus   = "bad_status"
	analysisErrorKindBadJSON     = "bad_json"
	analysisErrorKindOther       = "other"
)

func analyzerBaseURL() string {
	v := os.Getenv("PY_ANALYSER_URL")
	if v == "" {
		return "http://127.0.0.1:8001"
	}
	return v
}

func analyzerRequestTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("PY_ANALYSER_TIMEOUT_MS"))
	if raw == "" {
		return 2500 * time.Millisecond
	}
	timeoutMS, err := strconv.Atoi(raw)
	if err != nil || timeoutMS < 100 {
		return 2500 * time.Millisecond
	}
	return time.Duration(timeoutMS) * time.Millisecond
}

func analyzerUserSafeError(err error) string {
	kind, _ := analyzerErrorDetails(err)
	return analyzerUserSafeErrorByKind(kind)
}

func analyzerUserSafeErrorByKind(kind string) string {
	switch kind {
	case analysisErrorKindTimeout:
		return "Analysis timed out. Showing previous result."
	case analysisErrorKindUnavailable:
		return "Analysis service unavailable. Showing previous result."
	case analysisErrorKindBadStatus:
		return "Analysis service returned an invalid response."
	case analysisErrorKindBadJSON:
		return "Analysis response could not be processed."
	case analysisErrorKindNone:
		return ""
	default:
		return "Analysis temporarily unavailable. Showing previous result."
	}
}

func analyzerErrorDetails(err error) (string, int) {
	if err == nil {
		return analysisErrorKindNone, 0
	}
	var callErr *analyzerCallError
	if errors.As(err, &callErr) {
		kind := strings.TrimSpace(callErr.Kind)
		if kind == "" {
			kind = analysisErrorKindOther
		}
		return kind, callErr.HTTPStatus
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return analysisErrorKindTimeout, 0
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if errors.Is(urlErr.Err, context.DeadlineExceeded) {
			return analysisErrorKindTimeout, 0
		}
		return analysisErrorKindUnavailable, 0
	}
	return analysisErrorKindOther, 0
}

func emitAnalysisLog(entry analysisLogEvent) {
	if strings.TrimSpace(entry.ErrorKind) == "" {
		entry.ErrorKind = analysisErrorKindNone
	}
	if strings.TrimSpace(entry.TimestampUTC) == "" {
		entry.TimestampUTC = time.Now().UTC().Format(time.RFC3339Nano)
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		log.Printf("warning: analysis log marshal failed: %v", err)
		return
	}
	log.Print(string(raw))
}

func analyzeByRequest(reqPayload analyzerRequest) (*analyzerResponse, error) {
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("analyzer request marshal failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), analyzerRequestTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, analyzerBaseURL()+"/analyze", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("analyzer request build failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pyAnalyzerHTTPClient.Do(req)
	if err != nil {
		kind, _ := analyzerErrorDetails(err)
		return nil, &analyzerCallError{
			Kind: kind,
			Err:  fmt.Errorf("analyzer request failed: %w", err),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("analyzer response read failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &analyzerCallError{
			Kind:       analysisErrorKindBadStatus,
			HTTPStatus: resp.StatusCode,
			Err:        fmt.Errorf("analyzer returned status=%d body=%s", resp.StatusCode, string(respBody)),
		}
	}

	var parsed analyzerResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, &analyzerCallError{
			Kind: analysisErrorKindBadJSON,
			Err:  fmt.Errorf("analyzer response parse failed: %w", err),
		}
	}

	// Printed for testing as requested.
	log.Printf("analyzer response: %s", string(respBody))
	return &parsed, nil
}

func StartAnalyzerWorker() {
	analysisWorkerOnce.Do(func() {
		go analysisWorkerLoop()
		log.Printf("python analyzer worker started")
	})
}

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
			continue
		}
		if result == nil {
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

func enqueueCurrentPositionAnalysis(gameID, command string) {
	history, err := sessionpkg.MoveHistoryByID(gameID)
	if err != nil {
		log.Printf("warning: enqueue analysis failed %s: %v", gameIDLabel(gameID), err)
		return
	}
	moveNumber := len(history)
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

func recordMoveAnalysis(command string, result analyzerResponse) {
	game := sessionpkg.GetGameSession()
	history := sessionpkg.GetMoveHistory()
	moveNumber := len(history)
	recordMoveAnalysisForGame(game.ID, moveNumber, command, result)
}

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

func getLatestAnalysisByGameID(gameID string) (latestAnalysisState, bool) {
	analysisStoreMu.Lock()
	defer analysisStoreMu.Unlock()
	entry, ok := latestAnalysisByGame[gameID]
	return entry, ok
}

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
	exportedGames[game.ID] = true
	analysisStoreMu.Unlock()

	payload := struct {
		GameID       string                 `json:"game_id"`
		Result       sessionpkg.GameResult  `json:"result"`
		Game         sessionpkg.GameSession `json:"game"`
		History      []string               `json:"history"`
		MoveAnalysis []moveAnalysisRecord   `json:"move_analysis"`
		ExportedAt   string                 `json:"exported_at"`
	}{
		GameID:       game.ID,
		Result:       game.Result,
		Game:         game,
		History:      historyByGameID(game.ID),
		MoveAnalysis: records,
		ExportedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	outputDir := filepath.Join("data", "analysis_exports")
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

func historyByGameID(gameID string) []string {
	history, err := sessionpkg.MoveHistoryByID(gameID)
	if err != nil {
		return []string{}
	}
	return history
}
