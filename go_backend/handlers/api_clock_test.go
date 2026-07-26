package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sessionpkg "go_backend/game/session"
)

func TestAPIGamesCreate_AcceptsClockTimeControl(t *testing.T) {
	h := NewHandler()
	body := strings.NewReader(
		"type=chess&mode=human_vs_human&humanColor=white&aiGameCount=1" +
			"&clockEnabled=true&whiteInitialMs=300000&blackInitialMs=60000&incrementMs=30000",
	)
	req := httptest.NewRequest(http.MethodPost, "/api/games", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.APIGames(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload gameStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Game.Clock == nil || !payload.Game.Clock.Enabled {
		t.Fatalf("expected enabled clock, got %+v", payload.Game.Clock)
	}
	if payload.Game.Clock.WhiteRemainingMs != 300_000 || payload.Game.Clock.BlackRemainingMs != 60_000 {
		t.Fatalf("bases white=%d black=%d", payload.Game.Clock.WhiteRemainingMs, payload.Game.Clock.BlackRemainingMs)
	}
	if payload.Game.Clock.IncrementMs != 30_000 {
		t.Fatalf("increment=%d", payload.Game.Clock.IncrementMs)
	}
}

func TestAPIGamesCreate_OmittingClockKeepsDisabled(t *testing.T) {
	h := NewHandler()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games",
		strings.NewReader("type=chess&mode=human_vs_human&humanColor=white"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.APIGames(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload gameStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Game.Clock == nil || payload.Game.Clock.Enabled {
		t.Fatalf("expected disabled clock, got %+v", payload.Game.Clock)
	}
}

func TestAPIGameConfig_AppliesHvAIHumanAIBases(t *testing.T) {
	h := NewHandler()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsAI, sessionpkg.GameTypeChess, "black", 1, "", "beginner")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/config",
		strings.NewReader(
			"type=chess&mode=human_vs_ai&humanColor=black&aiGameCount=1"+
				"&humanInitialMs=300000&aiInitialMs=60000&incrementMs=0",
		),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := sessionpkg.GetGameSessionByID(game.ID)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	// Human black → white=AI 60s, black=human 300s
	if updated.Clock.WhiteRemainingMs != 60_000 || updated.Clock.BlackRemainingMs != 300_000 {
		t.Fatalf("mapped bases white=%d black=%d", updated.Clock.WhiteRemainingMs, updated.Clock.BlackRemainingMs)
	}
}

func TestAPIGameNew_AppliesClockFromForm(t *testing.T) {
	h := NewHandler()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/new",
		strings.NewReader(
			"type=chess&mode=human_vs_human&humanColor=white"+
				"&clockEnabled=true&whiteInitialMs=60000&blackInitialMs=60000&incrementMs=1000",
		),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload gameStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.Game.Clock.Enabled || payload.Game.Clock.WhiteRemainingMs != 60_000 {
		t.Fatalf("new game clock: %+v", payload.Game.Clock)
	}
}
