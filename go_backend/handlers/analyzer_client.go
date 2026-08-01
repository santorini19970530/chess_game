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

// http client for the analyzer
var pyAnalyzerHTTPClient = &http.Client{
	Timeout: 0,
}

// request to the analyzer
type analyzerRequest struct {
	RequestID string `json:"request_id"`
	FEN       string `json:"fen"`
	Color     string `json:"color"`
	TopK      int    `json:"top_k"`
	GameType  string `json:"game_type,omitempty"`
}

// explainRequest mirrors the payload expected by Python /explain
type explainRequest struct {
	RequestID    string   `json:"request_id"`
	FEN          string   `json:"fen"`
	Color        string   `json:"color"` // side to move AFTER the explained move
	GameType     string   `json:"game_type"`
	SkillLevel   string   `json:"skill_level,omitempty"`
	HumanColor   string   `json:"human_color,omitempty"` // human seat in HvAI (white|black)
	ConceptHints []string `json:"concept_hints,omitempty"`
	MoveUCI      string   `json:"move_uci,omitempty"`
	MoveSAN      string   `json:"move_san,omitempty"`
	MoveHistory  []string `json:"move_history,omitempty"`
	Quick        bool     `json:"quick,omitempty"` // instant ground-truth line (no Ollama)
}

// explainSkillLevelFromProfile - maps AI strength (4 levels) → explain skill_level (3)
func explainSkillLevelFromProfile(profile string) string {
	return sessionpkg.SkillLevelFromAIProfile(profile)
}

// conceptHintsFromAnalysis - builds at most 3 short cues for the explain prompt (issue0047). missing fields are skipped; never blocks explain
func conceptHintsFromAnalysis(a analyzerResponse) []string {
	hints := make([]string, 0, 3)
	if t := strings.TrimSpace(a.ThreatSummary); t != "" {
		hints = append(hints, t)
	}

	balanceOK := false
	if a.HealthSummary != nil {
		if raw, ok := a.HealthSummary["material_balance_white_minus_black"]; ok {
			switch v := raw.(type) {
			case float64:
				balanceOK = true
				if v >= 100 {
					hints = append(hints, "White is ahead on material / evaluation.")
				} else if v <= -100 {
					hints = append(hints, "Black is ahead on material / evaluation.")
				}
			case int:
				balanceOK = true
				if v >= 100 {
					hints = append(hints, "White is ahead on material / evaluation.")
				} else if v <= -100 {
					hints = append(hints, "Black is ahead on material / evaluation.")
				}
			}
		}
	}
	if !balanceOK {
		if a.EvalCPWhite >= 100 {
			hints = append(hints, "White is ahead on material / evaluation.")
		} else if a.EvalCPWhite <= -100 {
			hints = append(hints, "Black is ahead on material / evaluation.")
		}
	}

	labels := make([]string, 0, 3)
	for i, sm := range a.SuggestedMoves {
		if i >= 3 {
			break
		}
		lab := strings.TrimSpace(sm.SAN)
		if lab == "" {
			lab = strings.TrimSpace(sm.UCI)
		}
		if lab != "" {
			labels = append(labels, lab)
		}
	}
	if len(labels) == 0 {
		if best := strings.TrimSpace(a.BestMoveUCI); best != "" {
			labels = append(labels, best)
		}
	}
	if len(labels) > 0 {
		hints = append(hints, "Engine suggested replies (side to move): "+strings.Join(labels, ", ")+".")
	}
	if len(hints) > 3 {
		hints = hints[:3]
	}
	return hints
}

// conceptHintsForExplain - only uses analysis for this exact FEN. stale MultiPV from the previous ply must not cue the coach (mismatch with Suggested moves)
func conceptHintsForExplain(latest latestAnalysisState, fen string) []string {
	analysisFEN := strings.TrimSpace(latest.Analysis.FEN)
	if analysisFEN == "" || analysisFEN != strings.TrimSpace(fen) {
		return nil
	}
	return conceptHintsFromAnalysis(latest.Analysis)
}

// explainHintWait - returns how long explain may wait for same-fen analysis cues (default 0)
func explainHintWait() time.Duration {
	raw := strings.TrimSpace(os.Getenv("EXPLAIN_HINT_WAIT_MS"))
	if raw == "" {
		return 0
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return 0
	}
	if ms > 15000 {
		ms = 15000
	}
	return time.Duration(ms) * time.Millisecond
}

// waitConceptHintsForFEN - peeks (and optionally polls) for same-FEN analysis cues. never blocks the game HTTP thread — only the explain goroutine
func waitConceptHintsForFEN(gameID, fen string, timeout time.Duration) []string {
	fen = strings.TrimSpace(fen)
	peek := func() []string {
		latest, ok := getLatestAnalysisByGameID(gameID)
		if !ok {
			return nil
		}
		return conceptHintsForExplain(latest, fen)
	}
	if timeout <= 0 {
		return peek()
	}
	deadline := time.Now().Add(timeout)
	for {
		if hints := peek(); hints != nil {
			return hints
		}
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// explainResponse is the JSON shape returned by Python /explain.
type explainResponse struct {
	RequestID    string   `json:"request_id"`
	Status       string   `json:"status"`
	Source       string   `json:"source"`
	Explanation  string   `json:"explanation"`
	MoveUCI      string   `json:"move_uci"`
	MoveSAN      string   `json:"move_san"`
	LatencyMS    int      `json:"latency_ms"`
	ConceptHints []string `json:"concept_hints,omitempty"`
}

// job to analyze the current position
type analysisJob struct {
	GameID          string
	MoveNumber      int
	Command         string
	Request         analyzerRequest
	EnqueueQueueLen int
}

// suggested move from the analyzer
type analyzerSuggestedMove struct {
	Rank  int    `json:"rank"`
	UCI   string `json:"uci"`
	SAN   string `json:"san"`
	Score int    `json:"score"`
}

// response from the analyzer
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

// record of the move analysis
type moveAnalysisRecord struct {
	MoveNumber int              `json:"move_number"`
	Command    string           `json:"command"`
	Analysis   analyzerResponse `json:"analysis"`
}

// moveExplanationRecord is coach evidence for report / QA
type moveExplanationRecord struct {
	MoveNumber   int      `json:"move_number"`
	MoveUCI      string   `json:"move_uci"`
	MoveSAN      string   `json:"move_san,omitempty"`
	FEN          string   `json:"fen"`
	SkillLevel   string   `json:"skill_level"`
	Source       string   `json:"source"`
	Explanation  string   `json:"explanation"`
	ConceptHints []string `json:"concept_hints,omitempty"`
	LatencyMS    int      `json:"latency_ms"`
	RecordedAt   string   `json:"recorded_at"`
}

// latest analysis state
type latestAnalysisState struct {
	GameID     string           `json:"game_id"`
	MoveNumber int              `json:"move_number"`
	Command    string           `json:"command"`
	Analysis   analyzerResponse `json:"analysis"`
	UpdatedAt  string           `json:"updated_at"`
}

// latest analysis status
type analysisLatestStatus struct {
	GameID              string               `json:"game_id"`
	RequestedMoveNumber int                  `json:"requested_move_number"`
	LatestMoveNumber    int                  `json:"latest_move_number"`
	Pending             bool                 `json:"pending"`
	LastError           string               `json:"last_error,omitempty"`
	Latest              *latestAnalysisState `json:"latest,omitempty"`
}

// error from the analyzer call
type analyzerCallError struct {
	Kind       string
	HTTPStatus int
	Err        error
}

// Error - returns the error message
func (e *analyzerCallError) Error() string {
	if e == nil || e.Err == nil {
		return "analyzer call error"
	}
	return e.Err.Error()
}

// Unwrap - returns the wrapped error
func (e *analyzerCallError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// log event for the analyzer
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

// global variables for the analyzer
var (
	moveAnalysisByGame      = map[string][]moveAnalysisRecord{}
	moveExplanationByGame   = map[string][]moveExplanationRecord{}
	latestAnalysisByGame    = map[string]latestAnalysisState{}
	latestRequestedByGame   = map[string]int{}
	analysisPendingByGame   = map[string]bool{}
	analysisLastErrorByGame = map[string]string{}
	exportedGames           = map[string]bool{}
	explainLatestByGame     = map[string]int{} // coalesce Ollama: only newest ply runs
	analysisStoreMu         sync.Mutex
	analysisQueue           = make(chan analysisJob, 128)
	analysisWorkerOnce      sync.Once
	// one Ollama /explain at a time — unbounded goroutines melt the machine late-game.
	explainOllamaSem = make(chan struct{}, 1)
)

// error kinds for the analyzer
const (
	analysisErrorKindNone        = "none"
	analysisErrorKindTimeout     = "timeout"
	analysisErrorKindUnavailable = "unavailable"
	analysisErrorKindBadStatus   = "bad_status"
	analysisErrorKindBadJSON     = "bad_json"
	analysisErrorKindOther       = "other"
)

// analyzerBaseURL - returns the base URL for the analyzer
func analyzerBaseURL() string {
	v := os.Getenv("PY_ANALYSER_URL")
	if v == "" {
		return "http://127.0.0.1:8001"
	}
	return v
}

// analyzerRequestTimeout - returns the http timeout for analyzer calls
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

// analyzerUserSafeError - maps an analyzer error to a user-safe message
func analyzerUserSafeError(err error) string {
	kind, _ := analyzerErrorDetails(err)
	return analyzerUserSafeErrorByKind(kind)
}

// analyzerUserSafeErrorByKind - maps an analyzer error kind to a user-safe message
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

// analyzerErrorDetails - builds structured analyzer error details for logs
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

// emitAnalysisLog - emits analysis log
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

// analyzeByRequest - posts one /analyze request to the python service
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

	// printed for testing as requested.
	log.Printf("analyzer response: %s", string(respBody))
	return &parsed, nil
}

// explainByRequest - performs a POST to the Python /explain endpoint
func explainByRequest(reqPayload explainRequest) (*explainResponse, error) {
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("explain request marshal failed: %w", err)
	}

	// use a slightly longer timeout than analysis because LLM generation can be slower.
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, analyzerBaseURL()+"/explain", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("explain request build failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pyAnalyzerHTTPClient.Do(req)
	if err != nil {
		kind, _ := analyzerErrorDetails(err)
		return nil, &analyzerCallError{
			Kind: kind,
			Err:  fmt.Errorf("explain request failed: %w", err),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("explain response read failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &analyzerCallError{
			Kind:       analysisErrorKindBadStatus,
			HTTPStatus: resp.StatusCode,
			Err:        fmt.Errorf("explain returned status=%d body=%s", resp.StatusCode, string(respBody)),
		}
	}

	var parsed explainResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, &analyzerCallError{
			Kind: analysisErrorKindBadJSON,
			Err:  fmt.Errorf("explain response parse failed: %w", err),
		}
	}

	if len(respBody) > 240 {
		log.Printf("explain response: %s…", string(respBody[:240]))
	} else {
		log.Printf("explain response: %s", string(respBody))
	}
	return &parsed, nil
}

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
				enqueueExplanation(job.GameID, job.Command, job.Command)
			}
			continue
		}
		if result == nil {
			if shouldExplainCommand(job.Command) {
				enqueueExplanation(job.GameID, job.Command, job.Command)
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
			enqueueExplanation(job.GameID, job.Command, job.Command)
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
		GameID        string                   `json:"game_id"`
		Result        sessionpkg.GameResult    `json:"result"`
		Game          sessionpkg.GameSession   `json:"game"`
		History       []string                 `json:"history"`
		MoveAnalysis  []moveAnalysisRecord     `json:"move_analysis"`
		Explanations  []moveExplanationRecord  `json:"explanations"`
		ExportedAt    string                   `json:"exported_at"`
	}{
		GameID:       game.ID,
		Result:       game.Result,
		Game:         game,
		History:      historyByGameID(game.ID),
		MoveAnalysis: records,
		Explanations: explains,
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

// historyByGameID - history by game id
func historyByGameID(gameID string) []string {
	history, err := sessionpkg.MoveHistoryByID(gameID)
	if err != nil {
		return []string{}
	}
	return history
}
