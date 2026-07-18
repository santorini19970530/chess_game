package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sessionpkg "go_backend/game/session"
)

func TestAPIGameMove_DoesNotMutateOtherGame(t *testing.T) {
	h := NewHandler()
	gameA, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("expected game A create success, got %v", err)
	}
	gameB, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("expected game B create success, got %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+gameA.ID+"/move",
		strings.NewReader("command=e2e4"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	snapshotA, err := sessionpkg.BuildSnapshotByID(gameA.ID)
	if err != nil {
		t.Fatalf("expected snapshot A success, got %v", err)
	}
	snapshotB, err := sessionpkg.BuildSnapshotByID(gameB.ID)
	if err != nil {
		t.Fatalf("expected snapshot B success, got %v", err)
	}

	if len(snapshotA.History) != 1 {
		t.Fatalf("expected game A history length 1 after move, got %d", len(snapshotA.History))
	}
	if len(snapshotB.History) != 0 {
		t.Fatalf("expected game B history length 0, got %d", len(snapshotB.History))
	}
}

func TestAPIGameConfigRoute_UpdatesOnlyTargetGame(t *testing.T) {
	h := NewHandler()
	gameA, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("expected game A create success, got %v", err)
	}
	gameB, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("expected game B create success, got %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/games/"+gameA.ID+"/config",
		strings.NewReader("type=chess&mode=human_vs_ai&humanColor=black&aiGameCount=1&fen="),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	updatedA, err := sessionpkg.GetGameSessionByID(gameA.ID)
	if err != nil {
		t.Fatalf("expected game A lookup success, got %v", err)
	}
	untouchedB, err := sessionpkg.GetGameSessionByID(gameB.ID)
	if err != nil {
		t.Fatalf("expected game B lookup success, got %v", err)
	}
	if updatedA.Mode != sessionpkg.GameModeHumanVsAI || updatedA.Config.HumanColor != "black" {
		t.Fatalf("expected game A config updated, got mode=%s color=%s", updatedA.Mode, updatedA.Config.HumanColor)
	}
	if untouchedB.Mode != sessionpkg.GameModeHumanVsHuman || untouchedB.Config.HumanColor != "white" {
		t.Fatalf("expected game B config unchanged, got mode=%s color=%s", untouchedB.Mode, untouchedB.Config.HumanColor)
	}
}

func TestAPIGameNewRoute_CreatesFreshGameSnapshot(t *testing.T) {
	h := NewHandler()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("expected game create success, got %v", err)
	}
	if _, err := sessionpkg.ApplyMoveByCommandByID(game.ID, "e2e4"); err != nil {
		t.Fatalf("expected setup move success, got %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/"+game.ID+"/new", nil)
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Game struct {
			ID string `json:"id"`
		} `json:"game"`
		History []string `json:"history"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if payload.Game.ID == "" || payload.Game.ID == game.ID {
		t.Fatalf("expected new game id different from old id; old=%s new=%s", game.ID, payload.Game.ID)
	}
	if len(payload.History) != 0 {
		t.Fatalf("expected new game history to be empty, got %d", len(payload.History))
	}
}

func TestAPIGameNewRoute_RespectsTypeDropdown(t *testing.T) {
	h := NewHandler()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("expected game create success, got %v", err)
	}

	body := strings.NewReader("type=xianqi&mode=human_vs_human&humanColor=white&aiProfile=beginner")
	req := httptest.NewRequest(http.MethodPost, "/api/games/"+game.ID+"/new", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Game struct {
			Type string `json:"type"`
		} `json:"game"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if payload.Game.Type != string(sessionpkg.GameTypeXiangqi) {
		t.Fatalf("expected type %q, got %q", sessionpkg.GameTypeXiangqi, payload.Game.Type)
	}
}

func TestAPIGameFlagRoute_SetsTerminalResult(t *testing.T) {
	h := NewHandler()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("expected game create success, got %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/"+game.ID+"/flag", nil)
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Game struct {
			Result  string `json:"result"`
			Outcome struct {
				Status string `json:"status"`
				Winner string `json:"winner"`
				Loser  string `json:"loser"`
			} `json:"outcome"`
		} `json:"game"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if payload.Game.Outcome.Status != "resigned" {
		t.Fatalf("expected resigned status, got %q", payload.Game.Outcome.Status)
	}
	if payload.Game.Outcome.Winner != "black" || payload.Game.Outcome.Loser != "white" {
		t.Fatalf("expected winner=black loser=white, got winner=%q loser=%q", payload.Game.Outcome.Winner, payload.Game.Outcome.Loser)
	}
	if payload.Game.Result != "black_win" {
		t.Fatalf("expected black_win result, got %q", payload.Game.Result)
	}
}

func TestAPIGameLegalMovesRoute_ReturnsMovesForCurrentTurnPiece(t *testing.T) {
	h := NewHandler()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("expected game create success, got %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/games/%s/legal-moves?file=5&rank=2", game.ID), nil)
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		From struct {
			File int `json:"file"`
			Rank int `json:"rank"`
		} `json:"from"`
		LegalMoves []struct {
			File int `json:"file"`
			Rank int `json:"rank"`
		} `json:"legalMoves"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if payload.From.File != 5 || payload.From.Rank != 2 {
		t.Fatalf("expected source square e2 (5,2), got (%d,%d)", payload.From.File, payload.From.Rank)
	}
	if len(payload.LegalMoves) == 0 {
		t.Fatalf("expected legal moves for e2 pawn")
	}
}

func TestAPIGameLatestAnalysisRoute_ReturnsStatusShape(t *testing.T) {
	h := NewHandler()
	game, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "", "intermediate")
	if err != nil {
		t.Fatalf("expected game create success, got %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/"+game.ID+"/analysis/latest", nil)
	rec := httptest.NewRecorder()
	h.APIGameRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		GameID string `json:"game_id"`
		Pending bool  `json:"pending"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if payload.GameID != game.ID {
		t.Fatalf("expected analysis status for game %s, got %s", game.ID, payload.GameID)
	}
}
