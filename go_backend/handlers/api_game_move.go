// CM3070 FP code
// api_game_move.go - post move on a game

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	pieces "go_backend/game/piece"
	sessionpkg "go_backend/game/session"
)

func (h *Handler) postAPIGameMove(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid command payload")
		return
	}
	commandText := strings.ToLower(strings.TrimSpace(r.FormValue("command")))
	if commandText == "" {
		writeJSONError(w, http.StatusBadRequest, "Empty command")
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
	turnColor, err := sessionpkg.CurrentTurnColorByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}
	fromFile, fromRank, toFile, toRank, err := resolveMoveSquares(
		currentGame.Type, commandText, pieces.PieceColor(turnColor),
	)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	normalizedMove, err := sessionpkg.ApplyMoveByCommandByID(gameID, commandText)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("api move accepted %s command=%s", gameIDLabel(gameID), normalizedMove)
	// prefer squares from the accepted/normalized UCI (handles shogi "+").
	if ff, fr, tf, tr, parseErr := parseVariantUCISquares(normalizedMove); parseErr == nil {
		fromFile, fromRank, toFile, toRank = ff, fr, tf, tr
	}

	// coach explain runs after /analyze completes (see analysisWorkerLoop) so concept_hints
	// match Suggested moves. do not enqueue here — that raced MultiPV and left hints empty.

	finalGame, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}

	aiMoveApplied := ""

	// human vs AI: start AI thinking in background so the human move returns immediately.
	// the AI move (governed by the selected strength profile) will be applied later.
	if finalGame.Mode == sessionpkg.GameModeHumanVsAI && finalGame.Result == sessionpkg.GameResultInProgress {
		go func() {
			aiMove, aiErr := SelectAIMove(gameID)
			if aiErr != nil || aiMove == "" {
				if aiErr != nil {
					log.Printf("warning: background SelectAIMove failed for %s: %v", gameIDLabel(gameID), aiErr)
				}
				// aI may have flagged on thinking timeout — push outcome to the client.
				if g, gerr := sessionpkg.GetGameSessionByID(gameID); gerr == nil && g.Result != sessionpkg.GameResultInProgress {
					gameSocketHub.Broadcast(gameID, socketEventGameOutcome, map[string]interface{}{
						"result":  g.Result,
						"outcome": g.Outcome,
					})
				}
				return
			}
			if _, applyErr := sessionpkg.ApplyMoveByCommandByID(gameID, aiMove); applyErr != nil {
				log.Printf("warning: background AI move failed for %s: %v", gameIDLabel(gameID), applyErr)
				return
			}
			log.Printf("human_vs_ai: background AI move applied %s command=%s", gameIDLabel(gameID), aiMove)

			// broadcast the AI move via WebSocket so the frontend updates immediately
			gameSocketHub.Broadcast(gameID, socketEventMoveApplied, moveAppliedPayload(gameID, aiMove))

			// enqueue analysis; explain follows when analysis is recorded (same-FEN cues).
			enqueueCurrentPositionAnalysis(gameID, aiMove)

			// refresh outcome (may end the game)
			if _, refreshErr := sessionpkg.RefreshGameSessionOutcomeByID(gameID); refreshErr != nil {
				log.Printf("warning: refresh after background AI move failed %s: %v", gameIDLabel(gameID), refreshErr)
			}
			// archive if the game just ended
			if g, _ := sessionpkg.GetGameSessionByID(gameID); g.Result != sessionpkg.GameResultInProgress {
				_ = sessionpkg.ArchiveGameIfNeededByID(gameID)
			}
		}()
	}

	if finalGame.Result != sessionpkg.GameResultInProgress {
		if err := sessionpkg.ArchiveGameIfNeededByID(gameID); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to archive completed game")
			return
		}
	}
	snapshot, err := sessionpkg.BuildSnapshotByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load game state")
		return
	}

	response := struct {
		Command     string                     `json:"command"`
		CurrentTurn string                     `json:"currentTurn"`
		CheckedSide string                     `json:"checkedSide"`
		Game        sessionpkg.GameSession     `json:"game"`
		Captured    sessionpkg.CapturedSummary `json:"captured"`
		From        struct {
			File string `json:"file"`
			Rank int    `json:"rank"`
		} `json:"from"`
		To struct {
			File string `json:"file"`
			Rank int    `json:"rank"`
		} `json:"to"`
		History         []string                      `json:"history"`
		HistoryDetailed []sessionpkg.MoveHistoryEntry `json:"historyDetailed"`
		State           []sessionpkg.PieceState       `json:"state"`
		AIMove          string                        `json:"aiMove,omitempty"`
	}{
		Command:         normalizedMove,
		CurrentTurn:     snapshot.CurrentTurn,
		CheckedSide:     snapshot.CheckedSide,
		Game:            snapshot.Game, // post-move clock (increment / active side)
		Captured:        snapshot.Captured,
		History:         snapshot.History,
		HistoryDetailed: snapshot.HistoryDetailed,
		State:           snapshot.State,
		AIMove:          aiMoveApplied,
	}
	response.From.File = fromFile
	response.From.Rank = fromRank
	response.To.File = toFile
	response.To.Rank = toRank

	payload := moveAppliedPayload(gameID, normalizedMove)
	payload["from_file"] = response.From.File
	payload["from_rank"] = response.From.Rank
	payload["to_file"] = response.To.File
	payload["to_rank"] = response.To.Rank
	payload["history_len"] = len(snapshot.History)
	gameSocketHub.Broadcast(gameID, socketEventMoveApplied, payload)
	turnPayload := map[string]interface{}{
		"current_turn": response.CurrentTurn,
		"checked_side": response.CheckedSide,
	}
	attachClockFields(turnPayload, gameID, response.Game.Clock)
	gameSocketHub.Broadcast(gameID, socketEventTurnChanged, turnPayload)
	if snapshot.Game.Result != sessionpkg.GameResultInProgress {
		gameSocketHub.Broadcast(gameID, socketEventGameOutcome, map[string]interface{}{
			"result": snapshot.Game.Result,
			"outcome": map[string]interface{}{
				"status":       snapshot.Game.Outcome.Status,
				"winner":       snapshot.Game.Outcome.Winner,
				"loser":        snapshot.Game.Outcome.Loser,
				"checked_side": snapshot.Game.Outcome.CheckedSide,
				"message":      snapshot.Game.Outcome.Message,
			},
		})
	}

	enqueueCurrentPositionAnalysis(gameID, normalizedMove)
	if snapshot.Game.Result != sessionpkg.GameResultInProgress {
		exportGameAnalysisIfNeeded(snapshot.Game)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}
