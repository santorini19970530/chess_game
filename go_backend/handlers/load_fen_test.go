// CM3070 FP code
// load_fen_test.go - api checks for confirmed-fen session load

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

const diagramChessFEN = "r4r1k/p1qb2pp/1ppb4/5pBQ/2BP4/8/P1P2PPP/1R2R1K1 w - - 0 1"

// TestPostAPIGameLoadFen_ChessOK - checks load-fen creates a new hvh session at the fen
func TestPostAPIGameLoadFen_ChessOK(t *testing.T) {
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

	body := `{"fen":"` + diagramChessFEN + `","game":"chess"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-fen",
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
		t.Fatal("expected clock disabled for diagram load")
	}
	if payload.Game.Config.StartFEN != diagramChessFEN {
		t.Fatalf("startFen=%q want %q", payload.Game.Config.StartFEN, diagramChessFEN)
	}
	gotFEN, err := sessionpkg.CurrentFENByID(payload.Game.ID)
	if err != nil {
		t.Fatalf("current fen: %v", err)
	}
	if !strings.HasPrefix(gotFEN, "r4r1k/p1qb2pp/1ppb4/5pBQ/2BP4/8/P1P2PPP/1R2R1K1") {
		t.Fatalf("current fen=%q does not match loaded position", gotFEN)
	}
}

// TestPostAPIGameLoadFen_XiangqiOK - checks load-fen accepts a xiangqi fen into a new session
func TestPostAPIGameLoadFen_XiangqiOK(t *testing.T) {
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

	fen := sessionpkg.DefaultXiangqiStartFEN
	body := `{"fen":"` + fen + `","game":"xianqi"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-fen",
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
	if payload.Game.Type != sessionpkg.GameTypeXiangqi {
		t.Fatalf("type=%q want xianqi", payload.Game.Type)
	}
	gotFEN, err := sessionpkg.CurrentFENByID(payload.Game.ID)
	if err != nil {
		t.Fatalf("current fen: %v", err)
	}
	if !strings.HasPrefix(gotFEN, "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR") {
		t.Fatalf("current fen=%q does not match xiangqi start", gotFEN)
	}
}

// TestPostAPIGameLoadFen_ShogiOK - checks load-fen accepts a shogi fen (empty hand) into a new session
func TestPostAPIGameLoadFen_ShogiOK(t *testing.T) {
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

	fen := sessionpkg.DefaultShogiStartFEN
	body := `{"fen":"` + fen + `","game":"shogi"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-fen",
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
	if payload.Game.Type != sessionpkg.GameTypeShogi {
		t.Fatalf("type=%q want shogi", payload.Game.Type)
	}
	gotFEN, err := sessionpkg.CurrentFENByID(payload.Game.ID)
	if err != nil {
		t.Fatalf("current fen: %v", err)
	}
	if !strings.Contains(gotFEN, "lnsgkgsnl/1r5b1/ppppppppp") {
		t.Fatalf("current fen=%q does not match shogi start", gotFEN)
	}
	if !strings.Contains(gotFEN, "[]") {
		t.Fatalf("current fen=%q missing empty-hand marker (image model cannot recover hand)", gotFEN)
	}
}

// TestPostAPIGameLoadFen_BadFEN - checks load-fen rejects unparseable fen loudly
func TestPostAPIGameLoadFen_BadFEN(t *testing.T) {
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
		"/api/games/"+template.ID+"/load-fen",
		strings.NewReader(`{"fen":"not-a-fen","game":"chess"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

// TestPostAPIGameLoadFen_MissingFEN - checks load-fen requires fen
func TestPostAPIGameLoadFen_MissingFEN(t *testing.T) {
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
		"/api/games/"+template.ID+"/load-fen",
		strings.NewReader(`{"game":"chess"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

// TestPostAPIGameLoadFen_EnqueuesAnalyzeAndExplain - checks load-fen hits existing /analyze + /explain only
func TestPostAPIGameLoadFen_EnqueuesAnalyzeAndExplain(t *testing.T) {
	var analyzeHits atomic.Int32
	var explainHits atomic.Int32
	var lastExplainUCI atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/analyze":
			analyzeHits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"request_id":"diagram-a","status":"ok","source":"mock",
				"fen":"` + diagramChessFEN + `","evaluated_for_color":"white","health_summary":{},
				"eval_cp_white":120,"win_chance_white":0.62,"win_chance_black":0.38,
				"threat_summary":"White presses the kingside.","best_move_uci":"c4f7",
				"suggested_moves":[{"rank":1,"uci":"c4f7","san":"Bxf7+","score":120}],
				"latency_ms":1
			}`))
		case "/explain":
			explainHits.Add(1)
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if uci, ok := body["move_uci"].(string); ok {
				lastExplainUCI.Store(uci)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"request_id":"diagram-e","status":"ok","source":"heuristic",
				"explanation":"Tip coach line.","move_uci":"c4f7","move_san":"Bxf7+",
				"latency_ms":1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("PY_ANALYSER_URL", srv.URL)

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

	StartAnalyzerWorker()
	body := `{"fen":"` + diagramChessFEN + `","game":"chess"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+template.ID+"/load-fen",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if analyzeHits.Load() >= 1 && explainHits.Load() >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if analyzeHits.Load() < 1 {
		t.Fatal("expected /analyze after load-fen")
	}
	if explainHits.Load() < 1 {
		t.Fatal("expected /explain after load-fen (existing coach pipe)")
	}
	uci, _ := lastExplainUCI.Load().(string)
	if uci == "diagram" {
		t.Fatalf("explain move_uci must not be the diagram sentinel, got %q", uci)
	}
	if uci != "c4f7" && uci != "position" {
		t.Fatalf("explain move_uci=%q want best-move c4f7 or position", uci)
	}
}

// TestExplainArgsFromAnalysis_DiagramUsesBestMove - checks diagram explain labels prefer best move
func TestExplainArgsFromAnalysis_DiagramUsesBestMove(t *testing.T) {
	uci, san := explainArgsFromAnalysis("diagram", &analyzerResponse{
		BestMoveUCI: "c4f7",
		SuggestedMoves: []analyzerSuggestedMove{
			{Rank: 1, UCI: "c4f7", SAN: "Bxf7+"},
		},
	})
	if uci != "c4f7" || san != "Bxf7+" {
		t.Fatalf("got uci=%q san=%q", uci, san)
	}
	uci, san = explainArgsFromAnalysis("diagram", nil)
	if uci != "position" || san != "position" {
		t.Fatalf("nil analysis: got uci=%q san=%q", uci, san)
	}
	uci, san = explainArgsFromAnalysis("e2e4", nil)
	if uci != "e2e4" || san != "e2e4" {
		t.Fatalf("normal move: got uci=%q san=%q", uci, san)
	}
}
