package handlers

import (
	"fmt"
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
