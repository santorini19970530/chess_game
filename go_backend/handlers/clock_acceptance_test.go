package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go_backend/game/engine"
	sessionpkg "go_backend/game/session"
)

// Clock acceptance: TC create, increment, flag-on-timeout, clock-off unchanged, time-aware go.
func TestClock_Acceptance(t *testing.T) {
	t.Run("clock_off_create_unchanged", func(t *testing.T) {
		h := NewHandler()
		req := httptest.NewRequest(
			http.MethodPost, "/api/games",
			strings.NewReader("type=chess&mode=human_vs_human&humanColor=white"),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.APIGames(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
		var payload gameStateResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload.Game.Clock == nil || payload.Game.Clock.Enabled {
			t.Fatalf("clock-off create should stay disabled: %+v", payload.Game.Clock)
		}
	})

	t.Run("create_with_tc_move_awards_increment", func(t *testing.T) {
		h := NewHandler()
		body := "type=chess&mode=human_vs_human&humanColor=white" +
			"&clockEnabled=true&whiteInitialMs=1000&blackInitialMs=1000&incrementMs=30000"
		req := httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.APIGames(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
		}
		var created gameStateResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode: %v", err)
		}
		moveReq := httptest.NewRequest(
			http.MethodPost,
			"/api/games/"+created.Game.ID+"/move",
			strings.NewReader("command=e2e4"),
		)
		moveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		moveRec := httptest.NewRecorder()
		h.APIGameRoutes(moveRec, moveReq)
		if moveRec.Code != http.StatusOK {
			t.Fatalf("move status=%d body=%s", moveRec.Code, moveRec.Body.String())
		}
		var moved struct {
			Game sessionpkg.GameSession `json:"game"`
		}
		if err := json.Unmarshal(moveRec.Body.Bytes(), &moved); err != nil {
			t.Fatalf("decode move: %v", err)
		}
		if moved.Game.Clock.WhiteRemainingMs < 30_000 {
			t.Fatalf("expected +30s increment, white=%d", moved.Game.Clock.WhiteRemainingMs)
		}
	})

	t.Run("timeout_flags_and_rejects_moves", func(t *testing.T) {
		game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "")
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		if _, err := sessionpkg.SetClockByID(game.ID, 500, 500, 0); err != nil {
			t.Fatalf("SetClockByID: %v", err)
		}
		if err := sessionpkg.AdjustClockLastTickByID(game.ID, time.Now().UTC().Add(-2*time.Second)); err != nil {
			t.Fatalf("backdate: %v", err)
		}
		if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "e2e4"); err == nil {
			t.Fatal("expected flag reject")
		}
		updated, err := sessionpkg.GetGameSessionByID(game.ID)
		if err != nil {
			t.Fatalf("lookup: %v", err)
		}
		if updated.Outcome.Status != "resigned" || updated.Outcome.Loser != "white" {
			t.Fatalf("outcome=%+v", updated.Outcome)
		}
		if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "e7e5"); err == nil {
			t.Fatal("expected reject after flag")
		}
	})

	t.Run("asymmetric_hvai_bases", func(t *testing.T) {
		game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsAI, sessionpkg.GameTypeChess, "white", 1, "", "beginner")
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		white, black := sessionpkg.ClockSidesFromHumanAI("white", 300_000, 60_000)
		updated, err := sessionpkg.SetClockByID(game.ID, white, black, 0)
		if err != nil {
			t.Fatalf("SetClockByID: %v", err)
		}
		if updated.Clock.WhiteRemainingMs != 300_000 || updated.Clock.BlackRemainingMs != 60_000 {
			t.Fatalf("asymmetric: %+v", updated.Clock)
		}
	})

	t.Run("clock_on_ai_limit_is_time_aware", func(t *testing.T) {
		clk := sessionpkg.NewClock(120_000, 120_000, 1000)
		clk.Start("white", time.Unix(0, 0).UTC())
		limit := fsLimitForGame("intermediate", clk, "white")
		cmd := engine.GoCommand(limit)
		for _, part := range []string{"wtime 120000", "btime 120000", "winc 1000", "binc 1000"} {
			if !strings.Contains(cmd, part) {
				t.Fatalf("time-aware go missing %q in %q", part, cmd)
			}
		}
		offCmd := engine.GoCommand(fsLimitForGame("intermediate", nil, "white"))
		if offCmd != "go movetime 600" {
			t.Fatalf("clock-off go=%q", offCmd)
		}
	})
}
