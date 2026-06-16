// CM3070 FP code
// index.go - game playing page
// index page is having chess board (handle in issue 1), chess pieces (handle in issue 2), and other elements (to be handled later)

package handlers

import (
	"encoding/json"
	"fmt"
	chessboard "go_backend/game/board"
	commandpkg "go_backend/game/command"
	pieces "go_backend/game/piece"
	sessionpkg "go_backend/game/session"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type gameStateResponse struct {
	CurrentTurn     string                        `json:"currentTurn"`
	CheckedSide     string                        `json:"checkedSide"`
	Game            sessionpkg.GameSession        `json:"game"`
	Captured        sessionpkg.CapturedSummary    `json:"captured"`
	Analysis        *analyzerResponse             `json:"analysis,omitempty"`
	History         []string                      `json:"history"`
	HistoryDetailed []sessionpkg.MoveHistoryEntry `json:"historyDetailed"`
	State           []sessionpkg.PieceState       `json:"state"`
}

// generateChessBoard builds the chessboard html for the index page
// game state integration (gameSession) will be added later.
func generateChessBoard() template.HTML {
	gameBoard := chessboard.NewChessBoard()

	return gameBoard.Draw()
}

// Index renders the main game page template
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	// reject non-root paths
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// parse the base page and reusable partial templates
	t, err := template.ParseFiles(
		"../frontend/index.html",
		"../frontend/html_puzzles/head.html",
		"../frontend/html_puzzles/header.html",
		"../frontend/html_puzzles/footer.html",
	)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("index template parse error: %v", err)
		return
	}

	// build dynamic main content html in sequence
	var mainHTMLCode strings.Builder
	currentTurnLabel := sessionpkg.CurrentTurnLabel()
	whiteTurnClass := "game_info_col_white"
	blackTurnClass := "game_info_col_black"
	if currentTurnLabel == "White" {
		whiteTurnClass += " game_info_col_active"
	} else {
		blackTurnClass += " game_info_col_active"
	}

	// left panel
	mainHTMLCode.WriteString(`<div class="game_panel">`)

	mainHTMLCode.WriteString(`<div class="game_panel_config">`)
	mainHTMLCode.WriteString(`<h3 class="config_panel_title">Setup New Games</h3>`)
	mainHTMLCode.WriteString(`<label for="game_type">Game</label>`)
	mainHTMLCode.WriteString(`<select id="game_type"><option value="chess">Chess</option><option value="xianqi">Xiangqi</option><option value="shogi">Shogi</option></select>`)
	mainHTMLCode.WriteString(`<label for="game_mode">Mode</label>`)
	mainHTMLCode.WriteString(`<select id="game_mode"><option value="human_vs_human">Human vs Human</option><option value="human_vs_ai">Human vs AI</option><option value="ai_vs_ai">AI vs AI</option></select>`)
	mainHTMLCode.WriteString(`<label for="human_side">Human's side</label>`)
	mainHTMLCode.WriteString(`<select id="human_side"><option value="white">White</option><option value="black">Black</option></select>`)
	mainHTMLCode.WriteString(`<label for="ai_game_count">AI game count</label>`)
	mainHTMLCode.WriteString(`<input id="ai_game_count" type="number" min="1" value="1" />`)
	mainHTMLCode.WriteString(`<label for="fen_input">Starting FEN (optional)</label>`)
	mainHTMLCode.WriteString(`<textarea id="fen_input" rows="3" placeholder="rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"></textarea>`)
	mainHTMLCode.WriteString(`<button id="game_config_apply" type="button">Apply Setup</button>`)
	mainHTMLCode.WriteString(`</div>`)

	mainHTMLCode.WriteString(`<div class="game_panel_left">`)
	mainHTMLCode.WriteString(string(generateChessBoard()))
	mainHTMLCode.WriteString(`</div>`)

	// right panel
	mainHTMLCode.WriteString(`<div class="game_panel_right_top" style="display:flex;flex-direction:column;min-height:0;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<div class="game_info_table" role="table" aria-label="Game information table">`)
	mainHTMLCode.WriteString(`<div class="game_info_winprob_wrapper" role="presentation">`)
	mainHTMLCode.WriteString(`<div class="game_info_winprob_track">`)
	mainHTMLCode.WriteString(`<div id="game_info_winprob_white_bar" class="game_info_winprob_segment game_info_winprob_segment_white" style="width: 50%;">`)
	mainHTMLCode.WriteString(`<span id="game_info_winprob_white" class="game_info_winprob_label">50%</span>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div id="game_info_winprob_black_bar" class="game_info_winprob_segment game_info_winprob_segment_black" style="width: 50%;">`)
	mainHTMLCode.WriteString(`<span id="game_info_winprob_black" class="game_info_winprob_label">50%</span>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="game_info_row game_info_header" role="row">`)
	mainHTMLCode.WriteString(`<div id="game_info_side_white" class="game_info_cell game_info_side ` + whiteTurnClass + `" role="columnheader">White</div>`)
	mainHTMLCode.WriteString(`<div id="game_info_side_black" class="game_info_cell game_info_side ` + blackTurnClass + `" role="columnheader">Black</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="game_info_row" role="row">`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + whiteTurnClass + `" role="cell"><span id="game_info_captured_white" class="game_info_item_value game_info_capture_value"></span></div>`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + blackTurnClass + `" role="cell"><span id="game_info_captured_black" class="game_info_item_value game_info_capture_value"></span></div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="game_info_row" role="row">`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + whiteTurnClass + `" role="cell"><span id="game_info_time_white" class="game_info_item_value">⏱ --:--</span></div>`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + blackTurnClass + `" role="cell"><span id="game_info_time_black" class="game_info_item_value">⏱ --:--</span></div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="game_info_row" role="row">`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + whiteTurnClass + `" role="cell"><span id="game_info_result_white" class="game_info_item_value">Result: PLAYING</span></div>`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + blackTurnClass + `" role="cell"><span id="game_info_result_black" class="game_info_item_value">Result: PLAYING</span></div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_section" style="margin-top:16px;display:flex;flex-direction:column;flex:1;min-height:0;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<h3 class="chess_move_history_title">Move history</h3>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panels" style="display:grid;grid-template-columns:1fr 1fr;gap:10px;min-height:0;flex:1;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panel" style="display:flex;flex-direction:column;min-height:0;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<h4 class="chess_move_history_side_title">White</h4>`)
	mainHTMLCode.WriteString(`<ol id="chess_move_history_white" class="chess_move_history_list" style="flex:1;min-height:0;overflow-y:auto;overflow-x:hidden;">`)
	mainHTMLCode.WriteString(`<li class="chess_move_history_placeholder">No moves yet.</li>`)
	mainHTMLCode.WriteString(`</ol>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panel" style="display:flex;flex-direction:column;min-height:0;overflow:hidden;">`)
	mainHTMLCode.WriteString(`<h4 class="chess_move_history_side_title">Black</h4>`)
	mainHTMLCode.WriteString(`<ol id="chess_move_history_black" class="chess_move_history_list" style="flex:1;min-height:0;overflow-y:auto;overflow-x:hidden;">`)
	mainHTMLCode.WriteString(`<li class="chess_move_history_placeholder">No moves yet.</li>`)
	mainHTMLCode.WriteString(`</ol>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)

	mainHTMLCode.WriteString(`<div class="game_panel_right_bottom">`)
	mainHTMLCode.WriteString(`<label for="chess_command">Chess command</label>`)
	mainHTMLCode.WriteString(`<div class="command_row">`)
	mainHTMLCode.WriteString(`<input id="chess_command" type="text" placeholder="e2e4" />`)
	mainHTMLCode.WriteString(`<button id="chess_command_submit" type="button">Submit</button>`)
	mainHTMLCode.WriteString(`<button id="chess_flag" type="button">Flag</button>`)
	mainHTMLCode.WriteString(`<button id="chess_new_game" type="button">New Game</button>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<p id="chess_command_status" class="command_status" role="status" aria-live="polite"></p>`)
	mainHTMLCode.WriteString(`<input id="active_game_id" type="hidden" value="" />`)
	mainHTMLCode.WriteString(`<div id="promotion_picker" class="promotion_picker_hidden" role="dialog" aria-modal="true" aria-labelledby="promotion_picker_title">`)
	mainHTMLCode.WriteString(`<div class="promotion_picker_panel">`)
	mainHTMLCode.WriteString(`<h4 id="promotion_picker_title">Choose promotion piece</h4>`)
	mainHTMLCode.WriteString(`<div class="promotion_picker_choices">`)
	mainHTMLCode.WriteString(`<button type="button" class="promotion_choice_btn" data-promotion="q">Queen</button>`)
	mainHTMLCode.WriteString(`<button type="button" class="promotion_choice_btn" data-promotion="r">Rook</button>`)
	mainHTMLCode.WriteString(`<button type="button" class="promotion_choice_btn" data-promotion="b">Bishop</button>`)
	mainHTMLCode.WriteString(`<button type="button" class="promotion_choice_btn" data-promotion="n">Knight</button>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<textarea id="game_info_notes" class="game_info_notes_box" placeholder="Reserved for future use" rows="3" readonly></textarea>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<script src="/scripts/chess_command.js"></script>`)

	mainHTMLCode.WriteString(`</div>`)

	// prepare template data for rendering
	data := struct {
		PageTitle   string
		MainContent template.HTML
	}{
		PageTitle:   "Chess Game",
		MainContent: template.HTML(mainHTMLCode.String()),
	}

	// execute the index template with the prepared data
	if err := t.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, "Template render error", http.StatusInternalServerError)
		log.Printf("index template execute error: %v", err)
		return
	}
}

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
	game, err := sessionpkg.CreateGame(
		currentGame.Mode,
		currentGame.Type,
		currentGame.Config.HumanColor,
		currentGame.Config.AIGameCount,
		currentGame.Config.StartFEN,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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

	game, err := sessionpkg.UpdateGameConfigByID(gameID, mode, gameType, humanColor, aiGameCount, fen)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("game config updated %s mode=%s type=%s", gameIDLabel(gameID), game.Mode, game.Type)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Game sessionpkg.GameSession `json:"game"`
	}{Game: game}); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}

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

// SubmitChessCommand receives input from command textbox and send to server for processing
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
	expectedColor := pieces.PieceColor(turnColor)
	parsed, err := commandpkg.ParseCommandForColor(commandText, expectedColor)
	if err != nil {
		log.Printf("warning: invalid chess command: %q (%v)", commandText, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := commandpkg.ParseAndLogCommandForColor(commandText, expectedColor); err != nil {
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

	finalGame, err := sessionpkg.RefreshGameSessionOutcomeByID(gameID)
	if err != nil {
		http.Error(w, "Game session not found", http.StatusNotFound)
		return
	}

	aiMoveApplied := ""

	// Human vs AI orchestration (legacy path): after human move, call decision layer if mode is human_vs_ai
	if finalGame.Mode == sessionpkg.GameModeHumanVsAI && finalGame.Result == sessionpkg.GameResultInProgress {
		if aiMove, aiErr := SelectAIMove(gameID); aiErr == nil && aiMove != "" {
			if _, applyErr := sessionpkg.ApplyMoveByCommandByID(gameID, aiMove); applyErr != nil {
				log.Printf("warning: AI move failed to apply in human_vs_ai mode %s: %v", gameIDLabel(gameID), applyErr)
			} else {
				aiMoveApplied = aiMove
				log.Printf("human_vs_ai: AI move applied %s command=%s", gameIDLabel(gameID), aiMove)
			}
			finalGame, err = sessionpkg.RefreshGameSessionOutcomeByID(gameID)
			if err != nil {
				http.Error(w, "Game session not found after AI move", http.StatusNotFound)
				return
			}
		} else if aiErr != nil {
			log.Printf("warning: SelectAIMove failed for %s: %v", gameIDLabel(gameID), aiErr)
		}
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
	response.From.File = string(parsed.FromFile)
	response.From.Rank = parsed.FromRank
	response.To.File = string(parsed.ToFile)
	response.To.Rank = parsed.ToRank

	// Testing phase: call Python analyzer after each successful move
	// and print full response in Go server terminal.
	enqueueCurrentPositionAnalysis(gameID, normalizedMove)
	if finalGame.Result != sessionpkg.GameResultInProgress {
		exportGameAnalysisIfNeeded(finalGame)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}

func readGameConfigFromRequest(r *http.Request) (sessionpkg.GameMode, sessionpkg.GameType, string, int, string, error) {
	mode := sessionpkg.GameMode(strings.TrimSpace(r.FormValue("mode")))
	if mode == "" {
		mode = sessionpkg.GameModeHumanVsHuman
	}
	gameType := sessionpkg.GameType(strings.TrimSpace(r.FormValue("type")))
	if gameType == "" {
		gameType = sessionpkg.GameTypeChess
	}
	humanColor := strings.TrimSpace(r.FormValue("humanColor"))
	if humanColor == "" {
		humanColor = "white"
	}
	fen := strings.TrimSpace(r.FormValue("fen"))
	aiGameCount := 1
	if raw := strings.TrimSpace(r.FormValue("aiGameCount")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return "", "", "", 0, "", fmt.Errorf("invalid ai game count")
		}
		aiGameCount = parsed
	}
	return mode, gameType, humanColor, aiGameCount, fen, nil
}
