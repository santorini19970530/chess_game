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
	GameID     string
	MoveNumber int
	Command    string
	Request    analyzerRequest
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
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "Analysis timed out. Showing previous result."
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if errors.Is(urlErr.Err, context.DeadlineExceeded) {
			return "Analysis timed out. Showing previous result."
		}
		return "Analysis service unavailable. Showing previous result."
	}
	return "Analysis temporarily unavailable. Showing previous result."
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
		return nil, fmt.Errorf("analyzer request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("analyzer response read failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("analyzer returned status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var parsed analyzerResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("analyzer response parse failed: %w", err)
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
		result, err := analyzeByRequest(job.Request)
		analysisStoreMu.Lock()
		analysisPendingByGame[job.GameID] = false
		analysisLastErrorByGame[job.GameID] = ""
		latestRequestedMove := latestRequestedByGame[job.GameID]
		analysisStoreMu.Unlock()

		if err != nil {
			analysisStoreMu.Lock()
			analysisLastErrorByGame[job.GameID] = analyzerUserSafeError(err)
			analysisStoreMu.Unlock()
			log.Printf("warning: analyzer job failed game_id=%s move=%d: %v", job.GameID, job.MoveNumber, err)
			continue
		}
		if result == nil {
			continue
		}
		if job.MoveNumber < latestRequestedMove {
			log.Printf("stale analyzer response ignored: game_id=%s move=%d latest_requested=%d", job.GameID, job.MoveNumber, latestRequestedMove)
			continue
		}
		recordMoveAnalysisForGame(job.GameID, job.MoveNumber, job.Command, *result)
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
	analysisStoreMu.Unlock()

	select {
	case analysisQueue <- job:
	default:
		analysisStoreMu.Lock()
		analysisPendingByGame[gameID] = false
		analysisLastErrorByGame[gameID] = "Analysis queue is busy. Showing previous result."
		analysisStoreMu.Unlock()
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
