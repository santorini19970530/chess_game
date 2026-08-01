// CM3070 FP code
// api_game_lifecycle.go - config, flag, and new-game on an id

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	sessionpkg "go_backend/game/session"
)

func (h *Handler) postAPIGameConfig(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid configuration payload")
		return
	}
	mode, gameType, humanColor, aiGameCount, fen, profile, err := readGameConfigFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	game, err := sessionpkg.UpdateGameConfigByID(gameID, mode, gameType, humanColor, aiGameCount, fen, profile)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	game = applySkillLevelFromRequest(gameID, r, game)
	game, err = applyClockFromRequest(gameID, r, humanColor, game)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Game sessionpkg.GameSession `json:"game"`
	}{Game: game}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

// postAPIGameFlag - flags the side to move and archives if the game ended
func (h *Handler) postAPIGameFlag(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	currentGame, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}
	if currentGame.Result != sessionpkg.GameResultInProgress {
		message := currentGame.Outcome.Message
		if message == "" {
			message = "Game already ended."
		}
		writeJSONError(w, http.StatusConflict, message)
		return
	}
	game, err := sessionpkg.FlagCurrentTurnByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}
	if err := sessionpkg.ArchiveGameIfNeededByID(gameID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to archive flagged game")
		return
	}
	snapshot, err := sessionpkg.BuildSnapshotByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load game state")
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
	flagTurn := map[string]interface{}{
		"current_turn": response.CurrentTurn,
		"checked_side": response.CheckedSide,
	}
	attachClockFields(flagTurn, gameID, response.Game.Clock)
	gameSocketHub.Broadcast(gameID, socketEventTurnChanged, flagTurn)
	gameSocketHub.Broadcast(gameID, socketEventGameOutcome, map[string]interface{}{
		"result": game.Result,
		"outcome": map[string]interface{}{
			"status":       game.Outcome.Status,
			"winner":       game.Outcome.Winner,
			"loser":        game.Outcome.Loser,
			"checked_side": game.Outcome.CheckedSide,
			"message":      game.Outcome.Message,
		},
	})
	enqueueCurrentPositionAnalysis(gameID, "flag")
	exportGameAnalysisIfNeeded(game)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

// postAPIGameNew - creates a new game session from the posted setup
func (h *Handler) postAPIGameNew(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	currentGame, err := sessionpkg.GetGameSessionByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}
	if err := sessionpkg.ArchiveGameIfNeededByID(gameID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to archive current game")
		return
	}
	// parse form to allow "New Game" to respect current dropdown selections
	mode := currentGame.Mode
	gameType := currentGame.Type
	humanColor := currentGame.Config.HumanColor
	aiProfile := currentGame.Config.AIProfile
	skillLevel := currentGame.Config.SkillLevel
	if err := r.ParseForm(); err == nil {
		if m := r.FormValue("mode"); m != "" {
			mode = sessionpkg.GameMode(m)
		}
		if t := strings.TrimSpace(r.FormValue("type")); t != "" {
			gameType = sessionpkg.GameType(t)
		}
		if h := r.FormValue("humanColor"); h != "" {
			humanColor = h
		}
		if p := strings.TrimSpace(r.FormValue("aiProfile")); p != "" {
			aiProfile = p
		}
		if s := readSkillLevelFromRequest(r); s != "" {
			skillLevel = s
		}
	}

	startFEN := currentGame.Config.StartFEN
	if gameType != currentGame.Type {
		startFEN = "" // prior FEN belongs to the old variant
	}
	game, err := sessionpkg.CreateGame(
		mode,
		gameType,
		humanColor,
		currentGame.Config.AIGameCount,
		startFEN,
		aiProfile,
	)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if skillLevel != "" {
		if updated, setErr := sessionpkg.SetSkillLevelByID(game.ID, skillLevel); setErr == nil {
			game = updated
		}
	}
	game, err = applyClockFromRequest(game.ID, r, humanColor, game)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// auto-play first AI move if human is Black — run in background so "New Game" returns instantly.
	if game.Mode == sessionpkg.GameModeHumanVsAI && strings.ToLower(game.Config.HumanColor) == "black" && game.Result == sessionpkg.GameResultInProgress {
		go func() {
			if aiMove, aiErr := SelectAIMove(game.ID); aiErr == nil && aiMove != "" {
				if _, applyErr := sessionpkg.ApplyMoveByCommandByID(game.ID, aiMove); applyErr != nil {
					log.Printf("warning: initial background AI move failed for %s: %v", gameIDLabel(game.ID), applyErr)
					return
				}
				log.Printf("human_vs_ai: initial background AI move applied %s command=%s", gameIDLabel(game.ID), aiMove)

				gameSocketHub.Broadcast(game.ID, socketEventMoveApplied, moveAppliedPayload(game.ID, aiMove))
				enqueueCurrentPositionAnalysis(game.ID, aiMove)

				if _, refreshErr := sessionpkg.RefreshGameSessionOutcomeByID(game.ID); refreshErr != nil {
					log.Printf("warning: refresh after initial background AI failed %s: %v", gameIDLabel(game.ID), refreshErr)
				}
			} else if aiErr != nil {
				log.Printf("warning: initial SelectAIMove failed %s: %v", gameIDLabel(game.ID), aiErr)
			}
		}()
	}
	snapshot, err := sessionpkg.BuildSnapshotByID(game.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load game state")
		return
	}
	exportGameAnalysisIfNeeded(game)
	newTurn := map[string]interface{}{
		"current_turn": snapshot.CurrentTurn,
		"checked_side": snapshot.CheckedSide,
	}
	attachClockFields(newTurn, game.ID, snapshot.Game.Clock)
	gameSocketHub.Broadcast(game.ID, socketEventTurnChanged, newTurn)
	gameSocketHub.Broadcast(game.ID, socketEventGameOutcome, map[string]interface{}{
		"result": game.Result,
		"outcome": map[string]interface{}{
			"status":       game.Outcome.Status,
			"winner":       game.Outcome.Winner,
			"loser":        game.Outcome.Loser,
			"checked_side": game.Outcome.CheckedSide,
			"message":      game.Outcome.Message,
		},
	})
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
