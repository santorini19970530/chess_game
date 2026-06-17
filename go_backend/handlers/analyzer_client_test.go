package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAnalyzerRequestTimeout_UsesEnvOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","request_id":"x"}`))
	}))
	defer srv.Close()

	t.Setenv("PY_ANALYSER_URL", srv.URL)
	t.Setenv("PY_ANALYSER_TIMEOUT_MS", "100")

	_, err := analyzeByRequest(analyzerRequest{
		RequestID: "timeout-test",
		FEN:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		Color:     "white",
		TopK:      3,
	})
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	msg := analyzerUserSafeError(err)
	if !strings.Contains(strings.ToLower(msg), "timed out") {
		t.Fatalf("expected timeout-safe message, got %q", msg)
	}
}

func TestAnalyzerUserSafeError_ForUnavailableService(t *testing.T) {
	t.Setenv("PY_ANALYSER_URL", "http://127.0.0.1:1")
	t.Setenv("PY_ANALYSER_TIMEOUT_MS", "150")

	_, err := analyzeByRequest(analyzerRequest{
		RequestID: "unavailable-test",
		FEN:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		Color:     "white",
		TopK:      3,
	})
	if err == nil {
		t.Fatalf("expected unavailable-service error, got nil")
	}
	msg := analyzerUserSafeError(err)
	if !strings.Contains(strings.ToLower(msg), "unavailable") {
		t.Fatalf("expected unavailable-safe message, got %q", msg)
	}
}

func TestAnalyzerRequestTimeout_InvalidEnvFallsBackToDefault(t *testing.T) {
	t.Setenv("PY_ANALYSER_TIMEOUT_MS", "not-a-number")
	got := analyzerRequestTimeout()
	if got != 2500*time.Millisecond {
		t.Fatalf("expected default timeout 2500ms, got %s", got)
	}
}

func TestLatestAnalysisStatus_ContainsLastErrorWhenSet(t *testing.T) {
	gameID := fmt.Sprintf("test-game-%d", time.Now().UnixNano())

	analysisStoreMu.Lock()
	latestRequestedByGame[gameID] = 7
	analysisPendingByGame[gameID] = false
	analysisLastErrorByGame[gameID] = "Analysis service unavailable. Showing previous result."
	analysisStoreMu.Unlock()

	status := getLatestAnalysisStatusByGameID(gameID)
	if status.GameID != gameID {
		t.Fatalf("expected game id %s, got %s", gameID, status.GameID)
	}
	if status.LastError == "" {
		t.Fatalf("expected last_error to be populated")
	}
}

func TestEmitAnalysisLog_JSONShape(t *testing.T) {
	var buffer bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()
	log.SetOutput(&buffer)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	})

	emitAnalysisLog(analysisLogEvent{
		Event:               "analysis_completed",
		GameID:              "game-test",
		MoveNumber:          3,
		RequestID:           "game-test-move-3",
		QueueLen:            1,
		Pending:             false,
		Success:             true,
		LatencyMS:           24,
		ErrorKind:           analysisErrorKindNone,
		ErrorMessageSafe:    "",
		LatestRequestedMove: 3,
		AnalyzerSource:      "heuristic",
		AnalyzerLatencyMS:   4,
		BestMoveUCI:         "e2e4",
	})

	raw := strings.TrimSpace(buffer.String())
	if raw == "" {
		t.Fatalf("expected log output, got empty")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("expected valid JSON log line, got err=%v raw=%q", err, raw)
	}

	requiredFields := []string{
		"event",
		"game_id",
		"move_number",
		"request_id",
		"queue_len",
		"pending",
		"success",
		"latency_ms",
		"error_kind",
		"error_message_safe",
		"timestamp_utc",
	}
	for _, field := range requiredFields {
		if _, ok := payload[field]; !ok {
			t.Fatalf("expected required field %q in log payload", field)
		}
	}
	if payload["event"] != "analysis_completed" {
		t.Fatalf("expected event analysis_completed, got %v", payload["event"])
	}
}

func TestExplainByRequest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/explain" {
			t.Errorf("expected path /explain, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"request_id":"test-1",
			"status":"ok",
			"source":"ollama",
			"explanation":"e4 develops the king's pawn and opens lines.",
			"move_uci":"e2e4",
			"move_san":"e4",
			"latency_ms":123
		}`))
	}))
	defer srv.Close()

	t.Setenv("PY_ANALYSER_URL", srv.URL)

	res, err := explainByRequest(explainRequest{
		RequestID: "test-1",
		FEN:       "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1",
		Color:     "black",
		GameType:  "chess",
		MoveUCI:   "e2e4",
		MoveSAN:   "e4",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Source != "ollama" {
		t.Fatalf("expected source ollama, got %s", res.Source)
	}
	if res.Explanation == "" {
		t.Fatalf("expected non-empty explanation")
	}
}

func TestExplainByRequest_FallsBackOnError(t *testing.T) {
	t.Setenv("PY_ANALYSER_URL", "http://127.0.0.1:1") // unreachable
	t.Setenv("PY_ANALYSER_TIMEOUT_MS", "100")

	_, err := explainByRequest(explainRequest{
		RequestID: "fail-test",
		FEN:       "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		Color:     "white",
		GameType:  "chess",
		MoveUCI:   "e2e4",
	})
	if err == nil {
		t.Fatalf("expected error for unreachable service, got nil")
	}
}
