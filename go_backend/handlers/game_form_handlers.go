// CM3070 FP code
// game_form_handlers.go - new game, config update, and flag form handlers

package handlers

import (
	"encoding/json"
	"fmt"
	sessionpkg "go_backend/game/session"
	"log"
	"net/http"
	"strings"
)

func (h *Handler) NewGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	gameID := strings.TrimSpace(r.FormValue("gameId"))
	if gameID == "" {
		gameID = strings.TrimSpace(r.URL.Query().Get("gameId"))
	}
	if gameID == "" {
		gameID = sessionpkg.GetGameSession().ID
	}
	currentGame, err := sessionpkg.GetGameSessionByID(gameID)
	if err != nil {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}
	if err := sessionpkg.ArchiveGameIfNeededByID(gameID); err != nil {
		http.Error(w, "Failed to archive current game", http.StatusInternalServerError)
		log.Printf("archive game failed: %v", err)
		return
	}

	// parse form to allow "New Game" to respect current dropdown selections
	// without requiring the user to click "Apply Setup" first.
	if err := r.ParseForm(); err == nil {
		if m := r.FormValue("mode"); m != "" {
			currentGame.Mode = sessionpkg.GameMode(m)
		}
		if h := r.FormValue("humanColor"); h != "" {
			currentGame.Config.HumanColor = h
		}
	}

	game, err := sessionpkg.CreateGame(
		currentGame.Mode,
		currentGame.Type,
		currentGame.Config.HumanColor,
		currentGame.Config.AIGameCount,
		currentGame.Config.StartFEN,
		currentGame.Config.AIProfile,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// auto-play first AI move if human is Black (use the new game's config)
	if game.Mode == sessionpkg.GameModeHumanVsAI && strings.ToLower(game.Config.HumanColor) == "black" && game.Result == sessionpkg.GameResultInProgress {
		if aiMove, aiErr := SelectAIMove(game.ID); aiErr == nil && aiMove != "" {
			if _, applyErr := sessionpkg.ApplyMoveByCommandByID(game.ID, aiMove); applyErr != nil {
				log.Printf("warning: initial AI move failed for %s: %v", gameIDLabel(game.ID), applyErr)
			} else {
				log.Printf("human_vs_ai: initial AI move applied %s command=%s", gameIDLabel(game.ID), aiMove)
			}
			game, _ = sessionpkg.RefreshGameSessionOutcomeByID(game.ID)
		} else {
			log.Printf("warning: SelectAIMove failed: %v", aiErr)
		}
	}

	log.Printf("new game created from UI %s previous=%s", gameIDLabel(game.ID), gameIDLabel(gameID))
	snapshot, err := sessionpkg.BuildSnapshotByID(game.ID)
	if err != nil {
		http.Error(w, "Failed to load game state", http.StatusInternalServerError)
		return
	}
	response := gameStateResponse{
		CurrentTurn:     snapshot.CurrentTurn,
		CheckedSide:     snapshot.CheckedSide,
		Game:            snapshot.Game,
		Captured:        snapshot.Captured,
		History:         snapshot.History,
		HistoryDetailed: snapshot.HistoryDetailed,
		State:           snapshot.State,
	}
	exportGameAnalysisIfNeeded(game)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}

// UpdateGameConfig - applies mode/type/clock/skill setup from the form to the session
func (h *Handler) UpdateGameConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid configuration payload", http.StatusBadRequest)
		return
	}
	gameID := strings.TrimSpace(r.FormValue("gameId"))
	if gameID == "" {
		gameID = strings.TrimSpace(r.URL.Query().Get("gameId"))
	}
	if gameID == "" {
		gameID = sessionpkg.GetGameSession().ID
	}

	mode := sessionpkg.GameMode(strings.TrimSpace(r.FormValue("mode")))
	gameType := sessionpkg.GameType(strings.TrimSpace(r.FormValue("type")))
	humanColor := strings.TrimSpace(r.FormValue("humanColor"))
	fen := strings.TrimSpace(r.FormValue("fen"))
	aiGameCount := 1
	if raw := strings.TrimSpace(r.FormValue("aiGameCount")); raw != "" {
		if _, err := fmt.Sscanf(raw, "%d", &aiGameCount); err != nil {
			http.Error(w, "invalid ai game count", http.StatusBadRequest)
			return
		}
	}
	profile := strings.TrimSpace(r.FormValue("aiProfile"))
	if profile == "" {
		profile = strings.TrimSpace(r.FormValue("profile"))
	}
	skillLevel := strings.TrimSpace(r.FormValue("skillLevel"))
	if skillLevel == "" {
		skillLevel = strings.TrimSpace(r.FormValue("coachLevel"))
	}

	game, err := sessionpkg.UpdateGameConfigByID(gameID, mode, gameType, humanColor, aiGameCount, fen, profile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if skillLevel != "" {
		if updated, setErr := sessionpkg.SetSkillLevelByID(gameID, skillLevel); setErr == nil {
			game = updated
		}
	}
	log.Printf("game config updated %s mode=%s type=%s", gameIDLabel(gameID), game.Mode, game.Type)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Game sessionpkg.GameSession `json:"game"`
	}{Game: game}); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}

// FlagGame - ends the game by flag for the side that ran out of time
func (h *Handler) FlagGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	gameID := strings.TrimSpace(r.FormValue("gameId"))
	if gameID == "" {
		gameID = strings.TrimSpace(r.URL.Query().Get("gameId"))
	}
	if gameID == "" {
		gameID = sessionpkg.GetGameSession().ID
	}
	currentGame, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}
	if currentGame.Result != sessionpkg.GameResultInProgress {
		message := currentGame.Outcome.Message
		if message == "" {
			message = "Game already ended."
		}
		http.Error(w, message, http.StatusConflict)
		return
	}

	game, err := sessionpkg.FlagCurrentTurnByID(gameID)
	if err != nil {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}
	log.Printf("game flagged: game_id=%s loser=%s winner=%s", game.ID, game.Outcome.Loser, game.Outcome.Winner)
	if err := sessionpkg.ArchiveGameIfNeededByID(gameID); err != nil {
		http.Error(w, "Failed to archive flagged game", http.StatusInternalServerError)
		log.Printf("archive flagged game failed: %v", err)
		return
	}
	log.Printf("flagged game archived: game_id=%s", game.ID)

	snapshot, err := sessionpkg.BuildSnapshotByID(gameID)
	if err != nil {
		http.Error(w, "Failed to load game state", http.StatusInternalServerError)
		return
	}
	response := gameStateResponse{
		CurrentTurn:     snapshot.CurrentTurn,
		CheckedSide:     snapshot.CheckedSide,
		Game:            snapshot.Game,
		Captured:        snapshot.Captured,
		History:         snapshot.History,
		HistoryDetailed: snapshot.HistoryDetailed,
		State:           snapshot.State,
	}
	enqueueCurrentPositionAnalysis(gameID, "flag")
	log.Printf("game flagged %s loser=%s winner=%s", gameIDLabel(gameID), game.Outcome.Loser, game.Outcome.Winner)
	exportGameAnalysisIfNeeded(game)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}
