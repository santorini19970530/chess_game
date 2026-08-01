// CM3070 FP code
// api_game_analysis.go - legal-moves and latest analysis

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	sessionpkg "go_backend/game/session"
)

func (h *Handler) getAPIGameLegalMoves(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	game, err := sessionpkg.GetGameSessionByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}

	// shogi hand drop: ?dropKind=pawn (no file/rank).
	if dropKind := strings.TrimSpace(r.URL.Query().Get("dropKind")); dropKind != "" {
		if game.Type != sessionpkg.GameTypeShogi {
			writeJSONError(w, http.StatusBadRequest, "dropKind only for shogi")
			return
		}
		moves, err := sessionpkg.LegalDropsForKindByID(gameID, dropKind)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			DropKind   string                        `json:"dropKind"`
			LegalMoves []sessionpkg.LegalDestination `json:"legalMoves"`
		}{DropKind: strings.ToLower(dropKind), LegalMoves: moves}); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Response encode error")
		}
		return
	}

	maxFile, maxRank := 8, 8
	switch game.Type {
	case sessionpkg.GameTypeXiangqi:
		maxFile, maxRank = 9, 10
	case sessionpkg.GameTypeShogi:
		maxFile, maxRank = 9, 9
	}
	file, err := strconv.Atoi(r.URL.Query().Get("file"))
	if err != nil || file < 1 || file > maxFile {
		writeJSONError(w, http.StatusBadRequest, "invalid file")
		return
	}
	rank, err := strconv.Atoi(r.URL.Query().Get("rank"))
	if err != nil || rank < 1 || rank > maxRank {
		writeJSONError(w, http.StatusBadRequest, "invalid rank")
		return
	}
	moves, err := sessionpkg.LegalMovesForSquareByID(gameID, file, rank)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
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
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

// getAPIGameLatestAnalysis - returns the latest coach analysis for a game id
func (h *Handler) getAPIGameLatestAnalysis(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	status := getLatestAnalysisStatusByGameID(gameID)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}
