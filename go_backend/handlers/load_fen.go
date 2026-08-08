// CM3070 FP code
// load_fen.go - load a confirmed diagram fen into a new review session

package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	sessionpkg "go_backend/game/session"
)

// request body for the load-fen endpoint
type loadFenRequest struct {
	FEN  string `json:"fen"`
	Game string `json:"game"`
}

// normalizeDiagramGameType - maps api game aliases to a session GameType
func normalizeDiagramGameType(raw string, fallback sessionpkg.GameType) (sessionpkg.GameType, error) {
	key := strings.TrimSpace(strings.ToLower(raw))
	if key == "" {
		if fallback == "" {
			return sessionpkg.GameTypeChess, nil
		}
		return fallback, nil
	}
	switch key {
	case "chess":
		return sessionpkg.GameTypeChess, nil
	case "xianqi", "xiangqi":
		return sessionpkg.GameTypeXiangqi, nil
	case "shogi":
		return sessionpkg.GameTypeShogi, nil
	default:
		return "", fmt.Errorf("unsupported game type %q", raw)
	}
}

// postAPIGameLoadFen - creates a clock-off hvh session from a confirmed fen for coach review
func (h *Handler) postAPIGameLoadFen(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	template, err := sessionpkg.GetGameSessionByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid load-fen payload")
		return
	}
	var req loadFenRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, `Expected JSON body {"fen":"...","game":"chess"}`)
		return
	}

	fen := strings.TrimSpace(req.FEN)
	if fen == "" {
		writeJSONError(w, http.StatusBadRequest, `Missing required field: "fen"`)
		return
	}

	gameType, err := normalizeDiagramGameType(req.Game, template.Type)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := sessionpkg.ArchiveGameIfNeededByID(gameID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to archive current game")
		return
	}

	// fresh session: HvH + default clock-off so diagram import cannot flag.
	game, err := sessionpkg.CreateGame(
		sessionpkg.GameModeHumanVsHuman,
		gameType,
		"white",
		1,
		fen,
		template.Config.AIProfile,
	)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if skill := strings.TrimSpace(template.Config.SkillLevel); skill != "" {
		if updated, setErr := sessionpkg.SetSkillLevelByID(game.ID, skill); setErr == nil {
			game = updated
		}
	}

	// tip position has no move history — still enqueue coach for the loaded fen
	enqueueCurrentPositionAnalysis(game.ID, "diagram")

	snapshot, err := sessionpkg.BuildSnapshotByID(game.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load game state")
		return
	}
	log.Printf(
		"api load-fen %s -> %s type=%s",
		gameIDLabel(gameID),
		gameIDLabel(game.ID),
		game.Type,
	)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(gameStateResponse{
		CurrentTurn:     snapshot.CurrentTurn,
		CheckedSide:     snapshot.CheckedSide,
		Game:            snapshot.Game,
		Captured:        snapshot.Captured,
		History:         snapshot.History,
		HistoryDetailed: snapshot.HistoryDetailed,
		State:           snapshot.State,
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}
