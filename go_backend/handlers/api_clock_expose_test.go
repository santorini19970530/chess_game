package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sessionpkg "go_backend/game/session"
)

// TestAPIGameGet_SettlesClockRemaining - checks api game get settles clock remaining
func TestAPIGameGet_SettlesClockRemaining(t *testing.T) {
	h := NewHandler()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if _, err := sessionpkg.SetClockByID(game.ID, 5000, 5000, 0); err != nil {
		t.Fatalf("SetClockByID: %v", err)
	}
	// Backdate last tick so GET must settle ~1s off white.
	if err := sessionpkg.AdjustClockLastTickByID(game.ID, time.Now().UTC().Add(-1*time.Second)); err != nil {
		t.Fatalf("AdjustClockLastTickByID: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/"+game.ID, nil)
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload gameStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Game.Clock == nil || !payload.Game.Clock.Enabled {
		t.Fatalf("expected enabled clock on GET: %+v", payload.Game.Clock)
	}
	got := payload.Game.Clock.WhiteRemainingMs
	if got < 3500 || got > 4500 {
		t.Fatalf("settled white remaining=%d want ~4000±500", got)
	}
}

// TestAPIGameMove_ResponseIncludesUpdatedClock - checks api game move response includes updated clock
func TestAPIGameMove_ResponseIncludesUpdatedClock(t *testing.T) {
	h := NewHandler()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if _, err := sessionpkg.SetClockByID(game.ID, 1000, 1000, 30_000); err != nil {
		t.Fatalf("SetClockByID: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+game.ID+"/move",
		strings.NewReader("command=e2e4"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Game sessionpkg.GameSession `json:"game"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Game.Clock == nil || !payload.Game.Clock.Enabled {
		t.Fatal("expected clock on move response")
	}
	if payload.Game.Clock.WhiteRemainingMs < 30_000 {
		t.Fatalf("expected increment on white, got %d", payload.Game.Clock.WhiteRemainingMs)
	}
	if payload.Game.Clock.Active != "black" {
		t.Fatalf("active=%q want black", payload.Game.Clock.Active)
	}
}

// TestMoveAppliedPayload_IncludesClock - checks move applied payload includes clock
func TestMoveAppliedPayload_IncludesClock(t *testing.T) {
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if _, err := sessionpkg.SetClockByID(game.ID, 60_000, 60_000, 0); err != nil {
		t.Fatalf("SetClockByID: %v", err)
	}
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "e2e4"); err != nil {
		t.Fatalf("move: %v", err)
	}
	payload := moveAppliedPayload(game.ID, "e2e4")
	clock, ok := payload["clock"].(*sessionpkg.Clock)
	if !ok || clock == nil {
		t.Fatalf("expected *Clock in payload, got %#v", payload["clock"])
	}
	if !clock.Enabled {
		t.Fatal("expected enabled clock")
	}
	if clock.Active != "black" {
		t.Fatalf("active=%q want black", clock.Active)
	}
	remaining, ok := payload["remaining"].(map[string]int64)
	if !ok {
		t.Fatalf("expected remaining map, got %#v", payload["remaining"])
	}
	if remaining["white"] != clock.WhiteRemainingMs || remaining["black"] != clock.BlackRemainingMs {
		t.Fatalf("remaining=%v clock white=%d black=%d", remaining, clock.WhiteRemainingMs, clock.BlackRemainingMs)
	}
	if remaining["white"] <= 0 || remaining["black"] != 60_000 {
		t.Fatalf("remaining white=%d black=%d", remaining["white"], remaining["black"])
	}
}

// TestAttachClockFields_TurnChangedShape - checks attach clock fields turn changed shape
func TestAttachClockFields_TurnChangedShape(t *testing.T) {
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "")
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	updated, err := sessionpkg.SetClockByID(game.ID, 90_000, 45_000, 1000)
	if err != nil {
		t.Fatalf("SetClockByID: %v", err)
	}
	turnPayload := map[string]interface{}{
		"current_turn": "White",
		"checked_side": "",
	}
	attachClockFields(turnPayload, game.ID, updated.Clock)
	remaining, ok := turnPayload["remaining"].(map[string]int64)
	if !ok {
		t.Fatalf("expected remaining on turn payload, got %#v", turnPayload["remaining"])
	}
	if remaining["white"] != 90_000 || remaining["black"] != 45_000 {
		t.Fatalf("remaining=%v", remaining)
	}
}
