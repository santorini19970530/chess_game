// CM3070 FP code
// api_clock_test.go - tests for api clock

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sessionpkg "go_backend/game/session"
)

// TestAPIGamesCreate_AcceptsClockTimeControl - checks api games create accepts clock time control
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
	clk := payload.Game.Clock
	if clk.WhiteInitialMs != 300_000 || clk.BlackInitialMs != 60_000 {
		t.Fatalf("initial white=%d black=%d", clk.WhiteInitialMs, clk.BlackInitialMs)
	}
	// Snapshot settle may debit a few ms from the active side between Start and encode.
	if clk.WhiteRemainingMs > 300_000 || clk.WhiteRemainingMs < 299_000 {
		t.Fatalf("white remaining=%d want ~300000", clk.WhiteRemainingMs)
	}
	if clk.BlackRemainingMs != 60_000 {
		t.Fatalf("black remaining=%d want 60000", clk.BlackRemainingMs)
	}
	if clk.IncrementMs != 30_000 {
		t.Fatalf("increment=%d", clk.IncrementMs)
	}
}

// TestAPIGamesCreate_OmittingClockKeepsDisabled - checks api games create omitting clock keeps disabled
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

// TestAPIGameConfig_AppliesHvAIHumanAIBases - checks api game config applies hv ai human ai bases
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

// TestAPIGameNew_AppliesClockFromForm - checks api game new applies clock from form
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
