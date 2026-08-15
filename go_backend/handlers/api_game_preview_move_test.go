// CM3070 FP code
// api_game_preview_move_test.go - tests for preview-move api

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sessionpkg "go_backend/game/session"
)

// TestPostAPIGamePreviewMove_ReturnsEvalWithoutMutating - checks legal preview returns eval and leaves session intact
func TestPostAPIGamePreviewMove_ReturnsEvalWithoutMutating(t *testing.T) {
	var analyzeHits atomic.Int32
	var explainHits atomic.Int32
	var gotFEN atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/analyze":
			analyzeHits.Add(1)
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if fen, ok := body["fen"].(string); ok {
				gotFEN.Store(fen)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"request_id":"preview-a","status":"ok","source":"mock",
				"evaluation_source":"fairy-stockfish",
				"fen":"rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
				"evaluated_for_color":"black","health_summary":{},
				"eval_cp_white":25,"win_chance_white":0.53,"win_chance_black":0.47,
				"threat_summary":"","best_move_uci":"e7e5",
				"suggested_moves":[{"rank":1,"uci":"e7e5","san":"e5","score":20}],
				"latency_ms":1
			}`))
		case "/explain":
			explainHits.Add(1)
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["preview"] != true {
				t.Errorf("expected preview=true on explain")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"request_id":"preview-e","status":"ok","source":"mock",
				"explanation":"Preview mode: if white played e2e4, black would move next.",
				"move_uci":"e2e4","move_san":"e2e4","latency_ms":1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("PY_ANALYSER_URL", srv.URL)

	sessionpkg.ResetGame()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	beforeFEN, _ := sessionpkg.CurrentFENByID(game.ID)
	beforeHist, _ := sessionpkg.MoveHistoryByID(game.ID)
	beforeTurn, _ := sessionpkg.CurrentTurnColorByID(game.ID)

	h := NewHandler()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/preview-move",
		strings.NewReader(`{"command":"e2e4"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if resp["command"] != "e2e4" {
		t.Fatalf("command=%v", resp["command"])
	}
	fen, _ := resp["fen"].(string)
	if fen == "" || fen == beforeFEN {
		t.Fatalf("expected child fen, got %q", fen)
	}
	if resp["evaluation_source"] != "fairy-stockfish" {
		t.Fatalf("evaluation_source=%v", resp["evaluation_source"])
	}
	if resp["eval_cp_white"].(float64) != 25 {
		t.Fatalf("eval_cp_white=%v", resp["eval_cp_white"])
	}
	if analyzeHits.Load() != 1 {
		t.Fatalf("analyze hits=%d want 1", analyzeHits.Load())
	}
	if explainHits.Load() < 1 {
		t.Fatalf("explain must run for preview coach, hits=%d", explainHits.Load())
	}
	if exp, _ := resp["explanation"].(string); !strings.Contains(exp, "Preview mode") {
		t.Fatalf("explanation=%v", resp["explanation"])
	}
	if _, ok := resp["state"]; !ok {
		t.Fatal("expected child board state in preview response")
	}
	if analyzed, _ := gotFEN.Load().(string); analyzed != fen {
		t.Fatalf("analyzer fen=%q response fen=%q", analyzed, fen)
	}

	afterFEN, _ := sessionpkg.CurrentFENByID(game.ID)
	afterHist, _ := sessionpkg.MoveHistoryByID(game.ID)
	afterTurn, _ := sessionpkg.CurrentTurnColorByID(game.ID)
	if afterFEN != beforeFEN || afterTurn != beforeTurn {
		t.Fatalf("session mutated fen/turn")
	}
	if len(afterHist) != len(beforeHist) {
		t.Fatalf("history mutated: before=%v after=%v", beforeHist, afterHist)
	}
	if _, ok := getLatestAnalysisByGameID(game.ID); ok {
		t.Fatal("preview must not write latest analysis store")
	}
}

// TestPostAPIGamePreviewMove_IllegalCommand - checks illegal preview returns 400 without analyze
func TestPostAPIGamePreviewMove_IllegalCommand(t *testing.T) {
	var analyzeHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/analyze" {
			analyzeHits.Add(1)
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("PY_ANALYSER_URL", srv.URL)

	sessionpkg.ResetGame()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	h := NewHandler()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/preview-move",
		strings.NewReader(`{"command":"e2e5"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
	if analyzeHits.Load() != 0 {
		t.Fatalf("analyze should not run for illegal, hits=%d", analyzeHits.Load())
	}
}

// TestPostAPIGamePreviewMove_TerminalRejected - checks terminal game preview returns conflict
func TestPostAPIGamePreviewMove_TerminalRejected(t *testing.T) {
	sessionpkg.ResetGame()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := sessionpkg.FlagCurrentTurnByID(game.ID); err != nil {
		t.Fatalf("flag: %v", err)
	}

	h := NewHandler()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/preview-move",
		strings.NewReader(`{"command":"e2e4"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d want 409 body=%s", rec.Code, rec.Body.String())
	}
}

// TestPostAPIGamePreviewMove_XiangqiSmoke - checks xiangqi preview path hits analyze with child fen
func TestPostAPIGamePreviewMove_XiangqiSmoke(t *testing.T) {
	var gotGameType atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analyze" {
			http.NotFound(w, r)
			return
		}
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotGameType.Store(body["game_type"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"request_id":"xq","status":"ok","source":"mock","evaluation_source":"fairy-stockfish",
			"fen":"child","evaluated_for_color":"black","health_summary":{},
			"eval_cp_white":10,"win_chance_white":0.51,"win_chance_black":0.49,
			"suggested_moves":[],"latency_ms":1
		}`))
	}))
	defer srv.Close()
	t.Setenv("PY_ANALYSER_URL", srv.URL)

	sessionpkg.ResetGame()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeXiangqi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	beforeFEN, _ := sessionpkg.CurrentFENByID(game.ID)

	h := NewHandler()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/preview-move",
		strings.NewReader(`{"command":"a4a5"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if gt, _ := gotGameType.Load().(string); gt != string(sessionpkg.GameTypeXiangqi) {
		t.Fatalf("game_type=%v", gotGameType.Load())
	}
	afterFEN, _ := sessionpkg.CurrentFENByID(game.ID)
	if afterFEN != beforeFEN {
		t.Fatalf("xiangqi session fen mutated")
	}
}

// TestPostAPIGamePreviewMove_ShogiSmoke - checks shogi preview path leaves live fen unchanged
func TestPostAPIGamePreviewMove_ShogiSmoke(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"request_id":"sg","status":"ok","source":"mock","evaluation_source":"fairy-stockfish",
			"fen":"child","evaluated_for_color":"black","health_summary":{},
			"eval_cp_white":-5,"win_chance_white":0.48,"win_chance_black":0.52,
			"suggested_moves":[],"latency_ms":1
		}`))
	}))
	defer srv.Close()
	t.Setenv("PY_ANALYSER_URL", srv.URL)

	sessionpkg.ResetGame()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeShogi, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	beforeFEN, _ := sessionpkg.CurrentFENByID(game.ID)

	h := NewHandler()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/preview-move",
		strings.NewReader(`{"command":"c3c4"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	afterFEN, _ := sessionpkg.CurrentFENByID(game.ID)
	if afterFEN != beforeFEN {
		t.Fatalf("shogi session fen mutated")
	}
}

// TestPostAPIGamePreviewMove_PromotionSmoke - checks chess promotion preview returns queen child fen without mutation
func TestPostAPIGamePreviewMove_PromotionSmoke(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"request_id":"promo","status":"ok","source":"mock","evaluation_source":"fairy-stockfish",
			"fen":"child","evaluated_for_color":"black","health_summary":{},
			"eval_cp_white":900,"win_chance_white":0.95,"win_chance_black":0.05,
			"suggested_moves":[],"latency_ms":1
		}`))
	}))
	defer srv.Close()
	t.Setenv("PY_ANALYSER_URL", srv.URL)

	sessionpkg.ResetGame()
	fen := "8/4P3/8/8/8/8/8/4K2k w - - 0 1"
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, fen, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	beforeFEN, _ := sessionpkg.CurrentFENByID(game.ID)

	h := NewHandler()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/preview-move",
		strings.NewReader(`{"command":"e7e8q"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["command"] != "e7e8q" {
		t.Fatalf("command=%v", resp["command"])
	}
	child, _ := resp["fen"].(string)
	if !strings.Contains(child, "Q") {
		t.Fatalf("expected queen in child fen, got %q", child)
	}
	afterFEN, _ := sessionpkg.CurrentFENByID(game.ID)
	if afterFEN != beforeFEN {
		t.Fatalf("promotion preview mutated live fen")
	}
}

// TestPostAPIGamePreviewMove_AnalyzerUnavailable - checks analyzer failure returns 503 without mutation
func TestPostAPIGamePreviewMove_AnalyzerUnavailable(t *testing.T) {
	t.Setenv("PY_ANALYSER_URL", "http://127.0.0.1:1")
	t.Setenv("PY_ANALYSER_TIMEOUT_MS", "100")

	sessionpkg.ResetGame()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	beforeFEN, _ := sessionpkg.CurrentFENByID(game.ID)

	h := NewHandler()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/preview-move",
		strings.NewReader(`{"command":"e2e4"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rec.Code, rec.Body.String())
	}
	afterFEN, _ := sessionpkg.CurrentFENByID(game.ID)
	if afterFEN != beforeFEN {
		t.Fatalf("session mutated after analyzer failure")
	}
	time.Sleep(20 * time.Millisecond)
	analysisStoreMu.Lock()
	pending := analysisPendingByGame[game.ID]
	analysisStoreMu.Unlock()
	if pending {
		t.Fatal("preview must not mark analysis pending")
	}
}
