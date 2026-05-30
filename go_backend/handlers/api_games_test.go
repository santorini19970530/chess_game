package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sessionpkg "go_backend/game/session"
)

func TestAPIGameMove_DoesNotMutateOtherGame(t *testing.T) {
	h := NewHandler()
	gameA, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "")
	if err != nil {
		t.Fatalf("expected game A create success, got %v", err)
	}
	gameB, err := sessionpkg.CreateGame(sessionpkg.GameModeHumanVsHuman, sessionpkg.GameTypeChess, "white", 1, "")
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
