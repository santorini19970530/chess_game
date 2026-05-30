package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	commandpkg "go_backend/game/command"
	pieces "go_backend/game/piece"
	sessionpkg "go_backend/game/session"
)

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
	mode, gameType, humanColor, aiGameCount, fen, err := readGameConfigFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	game, err := sessionpkg.CreateGame(mode, gameType, humanColor, aiGameCount, fen)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("api create game %s mode=%s type=%s", gameIDLabel(game.ID), game.Mode, game.Type)
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
	if len(parts) == 2 && parts[1] == "legal-moves" {
		h.getAPIGameLegalMoves(w, r, gameID)
		return
	}
	if len(parts) == 3 && parts[1] == "analysis" && parts[2] == "latest" {
		h.getAPIGameLatestAnalysis(w, r, gameID)
		return
	}
	writeJSONError(w, http.StatusNotFound, "API route not found")
}

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
	game, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}
	log.Printf("api get game %s result=%s", gameIDLabel(gameID), game.Result)
	snapshot, err := sessionpkg.BuildSnapshotByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load game state")
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
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

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
	expectedColor := pieces.PieceColor(turnColor)
	parsed, err := commandpkg.ParseCommandForColor(commandText, expectedColor)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := commandpkg.ParseAndLogCommandForColor(commandText, expectedColor); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	normalizedMove, err := sessionpkg.ApplyMoveByCommandByID(gameID, commandText)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("api move accepted %s command=%s", gameIDLabel(gameID), normalizedMove)
	finalGame, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
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
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

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
	mode, gameType, humanColor, aiGameCount, fen, err := readGameConfigFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	game, err := sessionpkg.UpdateGameConfigByID(gameID, mode, gameType, humanColor, aiGameCount, fen)
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
	enqueueCurrentPositionAnalysis(gameID, "flag")
	exportGameAnalysisIfNeeded(game)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}

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
	game, err := sessionpkg.CreateGame(
		currentGame.Mode,
		currentGame.Type,
		currentGame.Config.HumanColor,
		currentGame.Config.AIGameCount,
		currentGame.Config.StartFEN,
	)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	snapshot, err := sessionpkg.BuildSnapshotByID(game.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load game state")
		return
	}
	exportGameAnalysisIfNeeded(game)
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

func (h *Handler) getAPIGameLegalMoves(w http.ResponseWriter, r *http.Request, gameID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	file, err := strconv.Atoi(r.URL.Query().Get("file"))
	if err != nil || file < 1 || file > 8 {
		writeJSONError(w, http.StatusBadRequest, "invalid file")
		return
	}
	rank, err := strconv.Atoi(r.URL.Query().Get("rank"))
	if err != nil || rank < 1 || rank > 8 {
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
