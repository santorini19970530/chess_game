package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"go_backend/game/engine"
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
	log.Printf("api create game %s mode=%s type=%s", gameIDLabel(game.ID), game.Mode, game.Type)

	// If human is Black in Human vs AI mode, let the AI (White) play the first move immediately
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

// getAPIGameTopMoves returns the top-k moves with scores using Fairy-Stockfish.
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

	// Only available when Go UCI path is enabled
	if !useFairyStockfish() {
		writeJSONError(w, http.StatusServiceUnavailable, "Fairy-Stockfish path is disabled (set USE_FAIRY_STOCKFISH=true)")
		return
	}

	fs, err := getFairyStockfish("white")
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

	// Build set of legal UCI moves for the current position
	legalSet := make(map[string]struct{})
	if legalMoves, err := sessionpkg.AllLegalUCIMovesByID(gameID); err == nil {
		for _, mv := range legalMoves {
			legalSet[strings.ToLower(mv)] = struct{}{}
		}
	}

	// Filter to only legal moves
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
	// Prefer squares from the accepted/normalized UCI (handles shogi "+").
	if ff, fr, tf, tr, parseErr := parseVariantUCISquares(normalizedMove); parseErr == nil {
		fromFile, fromRank, toFile, toRank = ff, fr, tf, tr
	}

	// Enqueue LLM explanation for the move just played (human move).
	enqueueExplanation(gameID, normalizedMove, normalizedMove)

	finalGame, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}

	aiMoveApplied := ""

	// Human vs AI: start AI thinking in background so the human move returns immediately.
	// The AI move (governed by the selected strength profile) will be applied later.
	if finalGame.Mode == sessionpkg.GameModeHumanVsAI && finalGame.Result == sessionpkg.GameResultInProgress {
		go func() {
			aiMove, aiErr := SelectAIMove(gameID)
			if aiErr != nil || aiMove == "" {
				if aiErr != nil {
					log.Printf("warning: background SelectAIMove failed for %s: %v", gameIDLabel(gameID), aiErr)
				}
				// AI may have flagged on thinking timeout — push outcome to the client.
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

			// Broadcast the AI move via WebSocket so the frontend updates immediately
			gameSocketHub.Broadcast(gameID, socketEventMoveApplied, map[string]interface{}{
				"command": aiMove,
			})

			// Enqueue analysis (for the analysis panel / win prob update)
			enqueueCurrentPositionAnalysis(gameID, aiMove)

			// Enqueue LLM explanation for the AI move.
			enqueueExplanation(gameID, aiMove, aiMove)

			// Refresh outcome (may end the game)
			if _, refreshErr := sessionpkg.RefreshGameSessionOutcomeByID(gameID); refreshErr != nil {
				log.Printf("warning: refresh after background AI move failed %s: %v", gameIDLabel(gameID), refreshErr)
			}
			// Archive if the game just ended
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

	gameSocketHub.Broadcast(gameID, socketEventMoveApplied, map[string]interface{}{
		"command":     normalizedMove,
		"from_file":   response.From.File,
		"from_rank":   response.From.Rank,
		"to_file":     response.To.File,
		"to_rank":     response.To.Rank,
		"history_len": len(snapshot.History),
	})
	gameSocketHub.Broadcast(gameID, socketEventTurnChanged, map[string]interface{}{
		"current_turn": response.CurrentTurn,
		"checked_side": response.CheckedSide,
	})
	if finalGame.Result != sessionpkg.GameResultInProgress {
		gameSocketHub.Broadcast(gameID, socketEventGameOutcome, map[string]interface{}{
			"result": finalGame.Result,
			"outcome": map[string]interface{}{
				"status":       finalGame.Outcome.Status,
				"winner":       finalGame.Outcome.Winner,
				"loser":        finalGame.Outcome.Loser,
				"checked_side": finalGame.Outcome.CheckedSide,
				"message":      finalGame.Outcome.Message,
			},
		})
	}

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
	gameSocketHub.Broadcast(gameID, socketEventTurnChanged, map[string]interface{}{
		"current_turn": response.CurrentTurn,
		"checked_side": response.CheckedSide,
	})
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
	// Parse form to allow "New Game" to respect current dropdown selections
	mode := currentGame.Mode
	gameType := currentGame.Type
	humanColor := currentGame.Config.HumanColor
	aiProfile := currentGame.Config.AIProfile
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

	// Auto-play first AI move if human is Black — run in background so "New Game" returns instantly.
	if game.Mode == sessionpkg.GameModeHumanVsAI && strings.ToLower(game.Config.HumanColor) == "black" && game.Result == sessionpkg.GameResultInProgress {
		go func() {
			if aiMove, aiErr := SelectAIMove(game.ID); aiErr == nil && aiMove != "" {
				if _, applyErr := sessionpkg.ApplyMoveByCommandByID(game.ID, aiMove); applyErr != nil {
					log.Printf("warning: initial background AI move failed for %s: %v", gameIDLabel(game.ID), applyErr)
					return
				}
				log.Printf("human_vs_ai: initial background AI move applied %s command=%s", gameIDLabel(game.ID), aiMove)

				gameSocketHub.Broadcast(game.ID, socketEventMoveApplied, map[string]interface{}{
					"command": aiMove,
				})
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
	gameSocketHub.Broadcast(game.ID, socketEventTurnChanged, map[string]interface{}{
		"current_turn": snapshot.CurrentTurn,
		"checked_side": snapshot.CheckedSide,
	})
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
