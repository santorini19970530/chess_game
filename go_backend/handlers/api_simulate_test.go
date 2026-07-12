package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPISimulate_ValidRequest_ReturnsSummary(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate",
		strings.NewReader(`{"games":2,"profile":"beginner"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Games     int     `json:"games"`
		WhiteWins int     `json:"white_wins"`
		BlackWins int     `json:"black_wins"`
		Draws     int     `json:"draws"`
		AvgMoves  float64 `json:"avg_moves"`
		Results   []struct {
			Result          string        `json:"result"`
			Moves           int           `json:"moves"`
			HistoryDetailed []interface{} `json:"history_detailed"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp.Games != 2 {
		t.Fatalf("expected games=2, got %d", resp.Games)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Results))
	}
	if resp.Results[0].Moves <= 0 {
		t.Fatalf("expected move count > 0, got %d", resp.Results[0].Moves)
	}
}

func TestAPISimulate_InvalidN_Returns400(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate",
		strings.NewReader(`{"games":0}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for games=0, got %d", rec.Code)
	}
}

func TestAPISimulate_FormEncoded_Works(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate",
		strings.NewReader("games=1&profile=beginner"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for form data, got %d", rec.Code)
	}
}

func TestAPISimulate_DetailsFalse_OmitsHistory(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate?details=false",
		strings.NewReader(`{"games":1,"profile":"beginner"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Results []struct {
			HistoryDetailed interface{} `json:"history_detailed"`
		} `json:"results"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	// When details=false, history_detailed should be absent or nil
	if len(resp.Results) > 0 && resp.Results[0].HistoryDetailed != nil {
		t.Fatalf("expected no history_detailed when details=false")
	}
}

func TestAPISimulate_DetailsTrue_IncludesHistory(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate?details=true",
		strings.NewReader(`{"games":1,"profile":"beginner"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Results []struct {
			Moves           int           `json:"moves"`
			HistoryDetailed []interface{} `json:"history_detailed"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Moves <= 0 {
		t.Fatalf("expected moves > 0, got %d", resp.Results[0].Moves)
	}
	if len(resp.Results[0].HistoryDetailed) == 0 {
		t.Fatal("expected history_detailed when details=true")
	}
}

func TestAPISimulate_InvalidUpperBound_Returns400(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate",
		strings.NewReader(`{"games":1001}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for games>1000, got %d", rec.Code)
	}
}

func TestAPISimulate_InvalidDetailsFlag_Returns400(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate?details=maybe",
		strings.NewReader(`{"games":1}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid details flag, got %d", rec.Code)
	}
}

func TestAPISimulate_InProgress_Returns409(t *testing.T) {
	h := NewHandler()

	simulationRunMu.Lock()
	simulationRunInFlight = true
	simulationRunMu.Unlock()
	defer finishSimulationRun()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate",
		strings.NewReader(`{"games":1}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 when simulation already running, got %d", rec.Code)
	}
}

func TestAPISimulate_SideProfiles_RecordedInResponse(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate?details=false",
		strings.NewReader(`{"games":1,"white_profile":"beginner","black_profile":"master"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		GameType      string `json:"game_type"`
		WhiteProfile  string `json:"white_profile"`
		BlackProfile  string `json:"black_profile"`
		Profile       string `json:"profile"`
		Games         int    `json:"games"`
		AvgDurationMs float64 `json:"avg_duration_ms"`
		P95DurationMs int64   `json:"p95_duration_ms"`
		Results       []struct {
			DurationMs int64 `json:"duration_ms"`
			AvgMoveMs  int64 `json:"avg_move_ms"`
			Moves      int   `json:"moves"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp.Games != 1 {
		t.Fatalf("games=%d", resp.Games)
	}
	if resp.GameType != "chess" || resp.WhiteProfile != "beginner" || resp.BlackProfile != "master" {
		t.Fatalf("unexpected profiles: %+v", resp)
	}
	if resp.Profile != "" {
		t.Fatalf("expected profile omitted when sides differ, got %q", resp.Profile)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].DurationMs < 0 || resp.AvgDurationMs < 0 || resp.P95DurationMs < 0 {
		t.Fatalf("latency fields should be non-negative: %+v", resp)
	}
	if resp.Results[0].Moves > 0 && resp.Results[0].AvgMoveMs < 0 {
		t.Fatalf("avg_move_ms invalid: %+v", resp.Results[0])
	}
}

func TestAPISimulate_InvalidSideProfile_Returns400(t *testing.T) {
	h := NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/simulate",
		strings.NewReader(`{"games":1,"white_profile":"nope"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.APISimulate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
