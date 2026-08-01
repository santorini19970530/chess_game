package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	sessionpkg "go_backend/game/session"
)

// TestExtractUCIList_PlainText - checks extract uci list plain text
func TestExtractUCIList_PlainText(t *testing.T) {
	got, err := extractUCIList("e2e4 e7e5 g1f3")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"e2e4", "e7e5", "g1f3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// TestExtractUCIList_NewlinesAndCommas - checks extract uci list newlines and commas
func TestExtractUCIList_NewlinesAndCommas(t *testing.T) {
	got, err := extractUCIList("e2e4,\ne7e5\tg1f3")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"e2e4", "e7e5", "g1f3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// TestExtractUCIList_Empty - checks extract uci list empty
func TestExtractUCIList_Empty(t *testing.T) {
	if _, err := extractUCIList("  \n\t "); err == nil {
		t.Fatal("expected error for empty input")
	}
}

// TestExtractUCIFromPlayJSON_AnalysisExportHistory - checks extract uci from play json analysis export history
func TestExtractUCIFromPlayJSON_AnalysisExportHistory(t *testing.T) {
	raw := `{
		"game": {"type": "chess"},
		"history": ["White: e2e4", "Black: e7e6"]
	}`
	moves, gameType, err := extractUCIFromPlayJSON(raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gameType != "chess" {
		t.Fatalf("gameType=%q want chess", gameType)
	}
	want := []string{"e2e4", "e7e6"}
	if !reflect.DeepEqual(moves, want) {
		t.Fatalf("got %v want %v", moves, want)
	}
}

// TestExtractUCIFromPlayJSON_SimulationHistoryDetailed - checks extract uci from play json simulation history detailed
func TestExtractUCIFromPlayJSON_SimulationHistoryDetailed(t *testing.T) {
	raw := `{
		"game_type": "chess",
		"history_detailed": [
			{"command": "a2a3"},
			{"command": "a7a6"}
		]
	}`
	moves, gameType, err := extractUCIFromPlayJSON(raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if gameType != "chess" {
		t.Fatalf("gameType=%q want chess", gameType)
	}
	want := []string{"a2a3", "a7a6"}
	if !reflect.DeepEqual(moves, want) {
		t.Fatalf("got %v want %v", moves, want)
	}
}

// TestExtractUCIFromPlayJSON_Garbage - checks extract uci from play json garbage
func TestExtractUCIFromPlayJSON_Garbage(t *testing.T) {
	if _, _, err := extractUCIFromPlayJSON(`{"foo":1}`); err == nil {
		t.Fatal("expected error for JSON without history")
	}
	if _, _, err := extractUCIFromPlayJSON(`not-json`); err == nil {
		t.Fatal("expected error for non-JSON")
	}
}

// TestParseLoadMovesRaw_Dispatches - checks parse load moves raw dispatches
func TestParseLoadMovesRaw_Dispatches(t *testing.T) {
	moves, gameType, err := parseLoadMovesRaw("e2e4 e7e5")
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	if gameType != "" || !reflect.DeepEqual(moves, []string{"e2e4", "e7e5"}) {
		t.Fatalf("text got moves=%v type=%q", moves, gameType)
	}

	moves, gameType, err = parseLoadMovesRaw(`{"history":["White: e2e4"],"game":{"type":"xianqi"}}`)
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	if gameType != "xianqi" || !reflect.DeepEqual(moves, []string{"e2e4"}) {
		t.Fatalf("json got moves=%v type=%q", moves, gameType)
	}

	if _, _, err := parseLoadMovesRaw(""); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty error, got %v", err)
	}
}

// TestPostAPIGameLoadMoves_PlainUCI - checks post api game load moves plain uci
func TestPostAPIGameLoadMoves_PlainUCI(t *testing.T) {
	h := NewHandler()
	template, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsAI,
		sessionpkg.GameTypeChess,
		"white",
		1,
		"",
		"beginner",
	)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	body := `{"raw":"e2e4 e7e5"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-moves",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload gameStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Game.ID == "" || payload.Game.ID == template.ID {
		t.Fatalf("expected new game id, got %q (template %q)", payload.Game.ID, template.ID)
	}
	if payload.Game.Mode != sessionpkg.GameModeHumanVsHuman {
		t.Fatalf("mode=%q want human_vs_human", payload.Game.Mode)
	}
	if payload.Game.Clock != nil && payload.Game.Clock.Enabled {
		t.Fatal("expected clock disabled for review load")
	}
	if len(payload.History) != 2 {
		t.Fatalf("history len=%d want 2: %v", len(payload.History), payload.History)
	}
}

// TestPostAPIGameLoadMoves_IllegalPly - checks post api game load moves illegal ply
func TestPostAPIGameLoadMoves_IllegalPly(t *testing.T) {
	h := NewHandler()
	template, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsHuman,
		sessionpkg.GameTypeChess,
		"white",
		1,
		"",
		"",
	)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-moves",
		strings.NewReader(`{"raw":"e2e4 e2e5"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ply 2") {
		t.Fatalf("expected ply 2 in error, got %s", rec.Body.String())
	}
}

// TestPostAPIGameLoadMoves_AnalysisExportJSON - checks post api game load moves analysis export json
func TestPostAPIGameLoadMoves_AnalysisExportJSON(t *testing.T) {
	h := NewHandler()
	template, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsHuman,
		sessionpkg.GameTypeChess,
		"white",
		1,
		"",
		"",
	)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	rawJSON := `{"game":{"type":"chess"},"history":["White: e2e4","Black: e7e6"]}`
	body, _ := json.Marshal(map[string]string{"raw": rawJSON})
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-moves",
		strings.NewReader(string(body)),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload gameStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.History) != 2 {
		t.Fatalf("history len=%d want 2: %v", len(payload.History), payload.History)
	}
}

// TestPostAPIGameLoadMoves_SimHistoryDetailed - checks post api game load moves sim history detailed
func TestPostAPIGameLoadMoves_SimHistoryDetailed(t *testing.T) {
	h := NewHandler()
	template, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsHuman,
		sessionpkg.GameTypeChess,
		"white",
		1,
		"",
		"",
	)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	rawJSON := `{
		"game_type":"chess",
		"history_detailed":[
			{"command":"e2e4"},
			{"command":"e7e5"},
			{"command":"g1f3"}
		]
	}`
	body, _ := json.Marshal(map[string]string{"raw": rawJSON})
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-moves",
		strings.NewReader(string(body)),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload gameStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.History) != 3 {
		t.Fatalf("history len=%d want 3: %v", len(payload.History), payload.History)
	}
}

// TestPostAPIGameLoadMoves_EmptyRawStartPosition - checks post api game load moves empty raw start position
func TestPostAPIGameLoadMoves_EmptyRawStartPosition(t *testing.T) {
	h := NewHandler()
	template, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsHuman,
		sessionpkg.GameTypeChess,
		"white",
		1,
		"",
		"",
	)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	if _, err := sessionpkg.ApplyMoveByCommandByID(template.ID, "e2e4"); err != nil {
		t.Fatalf("seed move: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-moves",
		strings.NewReader(`{"raw":""}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload gameStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.History) != 0 {
		t.Fatalf("history len=%d want 0 (start)", len(payload.History))
	}
	if payload.Game.ID == template.ID {
		t.Fatal("expected new game id for empty load")
	}
}

// TestPostAPIGameLoadMoves_RealExportPrefix - checks post api game load moves real export prefix
func TestPostAPIGameLoadMoves_RealExportPrefix(t *testing.T) {
	// Smoke against a shortened slice of a real analysis_exports history shape.
	rawJSON := `{"game":{"type":"chess"},"history":["White: e2e4","Black: e7e6","White: c2c3","Black: d8h4","White: d2d3","Black: f8e7"]}`
	h := NewHandler()
	template, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsHuman,
		sessionpkg.GameTypeChess,
		"white",
		1,
		"",
		"",
	)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	body, _ := json.Marshal(map[string]string{"raw": rawJSON})
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-moves",
		strings.NewReader(string(body)),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload gameStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.History) != 6 {
		t.Fatalf("history len=%d want 6", len(payload.History))
	}
	if payload.Game.Mode != sessionpkg.GameModeHumanVsHuman {
		t.Fatalf("mode=%q", payload.Game.Mode)
	}
}
