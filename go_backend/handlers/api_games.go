package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	commandpkg "go_backend/game/command"
	pieces "go_backend/game/piece"
	sessionpkg "go_backend/game/session"
)

func (h *Handler) APIGames(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid game payload", http.StatusBadRequest)
		return
	}
	mode, gameType, humanColor, aiGameCount, fen, err := readGameConfigFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	game, err := sessionpkg.CreateGame(mode, gameType, humanColor, aiGameCount, fen)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("api create game %s mode=%s type=%s", gameIDLabel(game.ID), game.Mode, game.Type)
	snapshot, err := sessionpkg.BuildSnapshotByID(game.ID)
	if err != nil {
		http.Error(w, "Failed to load game state", http.StatusInternalServerError)
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
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}

func (h *Handler) APIGameRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/games/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.NotFound(w, r)
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
	http.NotFound(w, r)
}

func (h *Handler) getAPIGameByID(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := sessionpkg.ActivateGame(gameID); err != nil {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}
	game, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}
	log.Printf("api get game %s result=%s", gameIDLabel(gameID), game.Result)
	snapshot, err := sessionpkg.BuildSnapshotByID(gameID)
	if err != nil {
		http.Error(w, "Failed to load game state", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(gameStateResponse{
		CurrentTurn:     snapshot.CurrentTurn,
		CheckedSide:     snapshot.CheckedSide,
		Game:            game,
		Captured:        snapshot.Captured,
		History:         snapshot.History,
		HistoryDetailed: snapshot.HistoryDetailed,
		State:           snapshot.State,
	}); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}

func (h *Handler) postAPIGameMove(w http.ResponseWriter, r *http.Request, gameID string) {
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
	expectedColor := pieces.PieceColor(turnColor)
	parsed, err := commandpkg.ParseCommandForColor(commandText, expectedColor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := commandpkg.ParseAndLogCommandForColor(commandText, expectedColor); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	normalizedMove, err := sessionpkg.ApplyMoveByCommandByID(gameID, commandText)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("api move accepted %s command=%s", gameIDLabel(gameID), normalizedMove)
	finalGame, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}
	if finalGame.Result != sessionpkg.GameResultInProgress {
		if err := sessionpkg.ArchiveGameIfNeededByID(gameID); err != nil {
			http.Error(w, "Failed to archive completed game", http.StatusInternalServerError)
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
	}{
		Command:         normalizedMove,
		CurrentTurn:     snapshot.CurrentTurn,
		CheckedSide:     snapshot.CheckedSide,
		Game:            finalGame,
		Captured:        snapshot.Captured,
		History:         snapshot.History,
		HistoryDetailed: snapshot.HistoryDetailed,
		State:           snapshot.State,
	}
	response.From.File = string(parsed.FromFile)
	response.From.Rank = parsed.FromRank
	response.To.File = string(parsed.ToFile)
	response.To.Rank = parsed.ToRank

	enqueueCurrentPositionAnalysis(gameID, normalizedMove)
	if finalGame.Result != sessionpkg.GameResultInProgress {
		exportGameAnalysisIfNeeded(finalGame)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}
