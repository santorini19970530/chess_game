package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	sessionpkg "go_backend/game/session"
)

var pyAnalyzerHTTPClient = &http.Client{
	Timeout: 2500 * time.Millisecond,
}

type analyzerRequest struct {
	RequestID string `json:"request_id"`
	FEN       string `json:"fen"`
	Color     string `json:"color"`
	TopK      int    `json:"top_k"`
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

var (
	moveAnalysisByGame = map[string][]moveAnalysisRecord{}
	exportedGames      = map[string]bool{}
	analysisStoreMu    sync.Mutex
)

func analyzerBaseURL() string {
	v := os.Getenv("PY_ANALYSER_URL")
	if v == "" {
		return "http://127.0.0.1:8001"
	}
	return v
}

func analyzeCurrentPosition() (*analyzerResponse, error) {
	game := sessionpkg.GetGameSession()
	history := sessionpkg.GetMoveHistory()
	reqPayload := analyzerRequest{
		RequestID: fmt.Sprintf("%s-move-%d", game.ID, len(history)),
		FEN:       sessionpkg.CurrentFEN(),
		Color:     string(sessionpkg.CurrentTurnColor()),
		TopK:      5,
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("analyzer request marshal failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
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

func recordMoveAnalysis(command string, result analyzerResponse) {
	game := sessionpkg.GetGameSession()
	history := sessionpkg.GetMoveHistory()
	analysisStoreMu.Lock()
	defer analysisStoreMu.Unlock()
	moveAnalysisByGame[game.ID] = append(moveAnalysisByGame[game.ID], moveAnalysisRecord{
		MoveNumber: len(history),
		Command:    command,
		Analysis:   result,
	})
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
		History:      sessionpkg.GetMoveHistory(),
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
