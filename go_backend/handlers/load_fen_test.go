// CM3070 FP code
// load_fen_test.go - api checks for confirmed-fen session load

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
