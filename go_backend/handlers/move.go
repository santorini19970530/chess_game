package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	sessionpkg "go_backend/game/session"
)

// GetLegalMoves returns legal destination squares for a selected source square.
func (h *Handler) GetLegalMoves(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
	if gameID == "" {
		gameID = sessionpkg.GetGameSession().ID
	}
	game, err := sessionpkg.GetGameSessionByID(gameID)
	if err != nil {
		http.Error(w, "game session not found", http.StatusNotFound)
		return
	}
	maxFile, maxRank := 8, 8
	if game.Type == sessionpkg.GameTypeXiangqi {
		maxFile, maxRank = 9, 10
	}

	file, err := strconv.Atoi(r.URL.Query().Get("file"))
	if err != nil || file < 1 || file > maxFile {
		http.Error(w, "invalid file", http.StatusBadRequest)
		return
	}
	rank, err := strconv.Atoi(r.URL.Query().Get("rank"))
	if err != nil || rank < 1 || rank > maxRank {
		http.Error(w, "invalid rank", http.StatusBadRequest)
		return
	}

	moves, err := sessionpkg.LegalMovesForSquareByID(gameID, file, rank)
	if err != nil {
		http.Error(w, "game session not found", http.StatusNotFound)
		return
	}
	response := struct {
		From struct {
			File int `json:"file"`
			Rank int `json:"rank"`
		} `json:"from"`
		LegalMoves []sessionpkg.LegalDestination `json:"legalMoves"`
	}{
		LegalMoves: moves,
	}
	response.From.File = file
	response.From.Rank = rank

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}
