// CM3070 FP code
// api_game_query.go - get game snapshot and top-moves

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"go_backend/game/engine"
	sessionpkg "go_backend/game/session"
	"go_backend/usecase/aimove"
)

// getAPIGameByID - returns a game snapshot for the given id
func (h *Handler) getAPIGameByID(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if err := sessionpkg.ActivateGame(gameID); err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}
	if _, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID); err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}
	snapshot, err := sessionpkg.BuildSnapshotByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load game state")
		return
	}
	log.Printf("api get game %s result=%s", gameIDLabel(gameID), snapshot.Game.Result)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(gameStateResponse{
		CurrentTurn:     snapshot.CurrentTurn,
		CheckedSide:     snapshot.CheckedSide,
		Game:            snapshot.Game, // includes settled clock
		Captured:        snapshot.Captured,
		History:         snapshot.History,
		HistoryDetailed: snapshot.HistoryDetailed,
		State:           snapshot.State,
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

// getAPIGameTopMoves - returns the top-k moves with scores using Fairy-Stockfish
func (h *Handler) getAPIGameTopMoves(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	kStr := r.URL.Query().Get("k")
	profile := r.URL.Query().Get("profile")

	k := 3
	if kStr != "" {
		if parsed, err := strconv.Atoi(kStr); err == nil && parsed > 0 {
			k = parsed
		}
	}
	if k > 10 {
		k = 10
	}

	game, err := sessionpkg.GetGameSessionByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}

	fen, err := sessionpkg.CurrentFENByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to get FEN")
		return
	}

	if profile == "" {
		profile = game.Config.AIProfile
	}
	if profile == "" {
		profile = "intermediate"
	}

	// only available when Go UCI path is enabled
	if !aimove.UseFairyStockfish() {
		writeJSONError(w, http.StatusServiceUnavailable, "Fairy-Stockfish path is disabled (set USE_FAIRY_STOCKFISH=true)")
		return
	}

	fs, err := aimove.GetFairyStockfish("white")
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Fairy-Stockfish engine unavailable")
		return
	}
	if err := fs.SetVariant(string(game.Type)); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to set engine variant")
		return
	}

	limit := engine.Limit{Depth: 12}
	results, err := fs.TopKWithProfile(fen, k, profile, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to get top moves")
		return
	}

	// build set of legal UCI moves for the current position
	legalSet := make(map[string]struct{})
	if legalMoves, err := sessionpkg.AllLegalUCIMovesByID(gameID); err == nil {
		for _, mv := range legalMoves {
			legalSet[strings.ToLower(mv)] = struct{}{}
		}
	}

	// filter to only legal moves
	legalResults := make([]engine.UCIResult, 0, len(results))
	for _, r := range results {
		if _, ok := legalSet[strings.ToLower(r.Move)]; ok {
			legalResults = append(legalResults, r)
		}
	}
	if len(legalResults) == 0 {
		legalResults = results // fallback if filtering removed everything
	}

	type moveSuggestion struct {
		Move    string `json:"move"`
		Score   int    `json:"score_cp"`
		Depth   int    `json:"depth"`
		MultiPV int    `json:"multipv"`
	}

	suggestions := make([]moveSuggestion, 0, len(legalResults))
	for _, r := range legalResults {
		suggestions = append(suggestions, moveSuggestion{
			Move:    r.Move,
			Score:   r.Score,
			Depth:   r.Depth,
			MultiPV: r.MultiPV,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"game_id":     gameID,
		"profile":     profile,
		"k":           k,
		"suggestions": suggestions,
	})
}
