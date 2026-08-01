// CM3070 FP code
// api_games.go - /api/games collection create and id route dispatch

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	sessionpkg "go_backend/game/session"
)

// APIGames - dispatches /api/games collection requests
func (h *Handler) APIGames(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid game payload")
		return
	}
	mode, gameType, humanColor, aiGameCount, fen, profile, err := readGameConfigFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	game, err := sessionpkg.CreateGame(mode, gameType, humanColor, aiGameCount, fen, profile)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	game = applySkillLevelFromRequest(game.ID, r, game)
	game, err = applyClockFromRequest(game.ID, r, humanColor, game)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("api create game %s mode=%s type=%s", gameIDLabel(game.ID), game.Mode, game.Type)

	// if human is Black in Human vs AI mode, let the AI (White) play the first move immediately
	if mode == sessionpkg.GameModeHumanVsAI && strings.ToLower(humanColor) == "black" && game.Result == sessionpkg.GameResultInProgress {
		if aiMove, aiErr := SelectAIMove(game.ID); aiErr == nil && aiMove != "" {
			if _, applyErr := sessionpkg.ApplyMoveByCommandByID(game.ID, aiMove); applyErr != nil {
				log.Printf("warning: initial AI move failed for %s: %v", gameIDLabel(game.ID), applyErr)
			} else {
				log.Printf("human_vs_ai: initial AI move applied %s command=%s", gameIDLabel(game.ID), aiMove)
			}
			game, _ = sessionpkg.RefreshGameSessionOutcomeByID(game.ID)
		}
	}

	snapshot, err := sessionpkg.BuildSnapshotByID(game.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load game state")
		return
	}
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

// APIGameRoutes - routes /api/games/{id}/... to the matching handler
func (h *Handler) APIGameRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/games/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSONError(w, http.StatusNotFound, "API route not found")
		return
	}
	parts := strings.Split(path, "/")
	gameID := parts[0]
	if len(parts) == 1 {
		h.getAPIGameByID(w, r, gameID)
		return
	}
	if len(parts) == 2 && parts[1] == "move" {
		h.postAPIGameMove(w, r, gameID)
		return
	}
	if len(parts) == 2 && parts[1] == "config" {
		h.postAPIGameConfig(w, r, gameID)
		return
	}
	if len(parts) == 2 && parts[1] == "flag" {
		h.postAPIGameFlag(w, r, gameID)
		return
	}
	if len(parts) == 2 && parts[1] == "new" {
		h.postAPIGameNew(w, r, gameID)
		return
	}
	if len(parts) == 2 && parts[1] == "load-moves" {
		h.postAPIGameLoadMoves(w, r, gameID)
		return
	}
	if len(parts) == 2 && parts[1] == "legal-moves" {
		h.getAPIGameLegalMoves(w, r, gameID)
		return
	}
	if len(parts) == 2 && parts[1] == "top-moves" {
		h.getAPIGameTopMoves(w, r, gameID)
		return
	}
	if len(parts) == 3 && parts[1] == "analysis" && parts[2] == "latest" {
		h.getAPIGameLatestAnalysis(w, r, gameID)
		return
	}
	writeJSONError(w, http.StatusNotFound, "API route not found")
}
