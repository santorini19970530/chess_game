// CM3070 FP code
// submit_command.go - latest analysis and chess command submit

package handlers

import (
	"encoding/json"
	pieces "go_backend/game/piece"
	sessionpkg "go_backend/game/session"
	"log"
	"net/http"
	"strings"
)

func (h *Handler) GetLatestAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
	if gameID == "" {
		gameID = sessionpkg.GetGameSession().ID
	}
	status := getLatestAnalysisStatusByGameID(gameID)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}

// SubmitChessCommand - receives input from command textbox and send to server for processing
func (h *Handler) SubmitChessCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid command payload", http.StatusBadRequest)
		return
	}

	commandText := strings.ToLower(strings.TrimSpace(r.FormValue("command")))
	gameID := strings.TrimSpace(r.FormValue("gameId"))
	if gameID == "" {
		gameID = strings.TrimSpace(r.URL.Query().Get("gameId"))
	}
	if gameID == "" {
		gameID = sessionpkg.GetGameSession().ID
	}

	if commandText == "" {
		http.Error(w, "Empty command", http.StatusBadRequest)
		return
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

	turnColor, err := sessionpkg.CurrentTurnColorByID(gameID)
	if err != nil {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}
	fromFile, fromRank, toFile, toRank, err := resolveMoveSquares(
		currentGame.Type, commandText, pieces.PieceColor(turnColor),
	)
	if err != nil {
		log.Printf("warning: invalid command: %q (%v)", commandText, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	normalizedMove, err := sessionpkg.ApplyMoveByCommandByID(gameID, commandText)
	if err != nil {
		log.Printf("warning: failed to apply command %q: %v", commandText, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("command accepted %s command=%s", gameIDLabel(gameID), normalizedMove)
	if ff, fr, tf, tr, parseErr := parseVariantUCISquares(normalizedMove); parseErr == nil {
		fromFile, fromRank, toFile, toRank = ff, fr, tf, tr
	}

	finalGame, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}

	aiMoveApplied := ""

	// human vs AI: start AI thinking in background (legacy command path)
	if finalGame.Mode == sessionpkg.GameModeHumanVsAI && finalGame.Result == sessionpkg.GameResultInProgress {
		go func() {
			if aiMove, aiErr := SelectAIMove(gameID); aiErr == nil && aiMove != "" {
				if _, applyErr := sessionpkg.ApplyMoveByCommandByID(gameID, aiMove); applyErr != nil {
					log.Printf("warning: background AI move failed %s: %v", gameIDLabel(gameID), applyErr)
					return
				}
				log.Printf("human_vs_ai: background AI move applied %s command=%s", gameIDLabel(gameID), aiMove)

				// broadcast via WebSocket so frontend updates
				gameSocketHub.Broadcast(gameID, socketEventMoveApplied, moveAppliedPayload(gameID, aiMove))

				// enqueue analysis
				enqueueCurrentPositionAnalysis(gameID, aiMove)

				if _, refreshErr := sessionpkg.RefreshGameSessionOutcomeByID(gameID); refreshErr != nil {
					log.Printf("warning: refresh after background AI failed %s: %v", gameIDLabel(gameID), refreshErr)
				}
				if g, _ := sessionpkg.GetGameSessionByID(gameID); g.Result != sessionpkg.GameResultInProgress {
					_ = sessionpkg.ArchiveGameIfNeededByID(gameID)
				}
			} else if aiErr != nil {
				log.Printf("warning: background SelectAIMove failed %s: %v", gameIDLabel(gameID), aiErr)
			}
		}()
	}

	if finalGame.Result != sessionpkg.GameResultInProgress {
		if err := sessionpkg.ArchiveGameIfNeededByID(gameID); err != nil {
			http.Error(w, "Failed to archive completed game", http.StatusInternalServerError)
			log.Printf("archive completed game failed: %v", err)
			return
		}
	}

	snapshot, err := sessionpkg.BuildSnapshotByID(gameID)
	if err != nil {
		http.Error(w, "Failed to load game state", http.StatusInternalServerError)
		return
	}
	response := struct {
		Command     string                     `json:"command"`
		CurrentTurn string                     `json:"currentTurn"`
		CheckedSide string                     `json:"checkedSide"`
		Game        sessionpkg.GameSession     `json:"game"`
		Captured    sessionpkg.CapturedSummary `json:"captured"`
		Analysis    *analyzerResponse          `json:"analysis,omitempty"`
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
		Game:            finalGame,
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

	// enqueue coach analysis for the position after a successful move.
	enqueueCurrentPositionAnalysis(gameID, normalizedMove)
	if finalGame.Result != sessionpkg.GameResultInProgress {
		exportGameAnalysisIfNeeded(finalGame)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}
