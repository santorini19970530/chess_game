// CM3070 FP code
// index.go - game playing page
// index page is having chess board (handle in issue 1), chess pieces (handle in issue 2), and other elements (to be handled later)

package handlers

import (
	"encoding/json"
	chessboard "go_backend/game/board"
	commandpkg "go_backend/game/command"
	sessionpkg "go_backend/game/session"
	"html/template"
	"log"
	"net/http"
	"strings"
)

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

	mainHTMLCode.WriteString(`<div class="game_panel_left">`)
	mainHTMLCode.WriteString(string(generateChessBoard()))
	mainHTMLCode.WriteString(`</div>`)

	// right panel
	mainHTMLCode.WriteString(`<div class="game_panel_right_top">`)
	mainHTMLCode.WriteString(`<div class="game_info_table" role="table" aria-label="Game information table">`)
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
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + whiteTurnClass + `" role="cell"><span id="game_info_winprob_white" class="game_info_item_value">◎ TBD</span></div>`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + blackTurnClass + `" role="cell"><span id="game_info_winprob_black" class="game_info_item_value">◎ TBD</span></div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="game_info_row" role="row">`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + whiteTurnClass + `" role="cell"><span id="game_info_result_white" class="game_info_item_value">Result: PLAYING</span></div>`)
	mainHTMLCode.WriteString(`<div class="game_info_cell ` + blackTurnClass + `" role="cell"><span id="game_info_result_black" class="game_info_item_value">Result: PLAYING</span></div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panels">`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panel">`)
	mainHTMLCode.WriteString(`<ol id="chess_move_history_white" class="chess_move_history_list">`)
	mainHTMLCode.WriteString(`<li class="chess_move_history_placeholder">No moves yet.</li>`)
	mainHTMLCode.WriteString(`</ol>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panel">`)
	mainHTMLCode.WriteString(`<ol id="chess_move_history_black" class="chess_move_history_list">`)
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
	mainHTMLCode.WriteString(`<button id="chess_new_game" type="button">New Game</button>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<p id="chess_command_status" class="command_status" role="status" aria-live="polite"></p>`)
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

	sessionpkg.ResetGame()
	game := sessionpkg.RefreshGameSessionOutcome()
	response := struct {
		CurrentTurn string                     `json:"currentTurn"`
		CheckedSide string                     `json:"checkedSide"`
		Game        sessionpkg.GameSession     `json:"game"`
		Captured    sessionpkg.CapturedSummary `json:"captured"`
		History     []string                   `json:"history"`
		State       []sessionpkg.PieceState    `json:"state"`
	}{
		CurrentTurn: sessionpkg.CurrentTurnLabel(),
		CheckedSide: sessionpkg.CheckedSideLabel(),
		Game:        game,
		Captured:    sessionpkg.GetCapturedSummary(),
		History:     sessionpkg.GetMoveHistory(),
		State:       sessionpkg.GetBoardState(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
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

	if commandText == "" {
		http.Error(w, "Empty command", http.StatusBadRequest)
		return
	}
	currentGame := sessionpkg.RefreshGameSessionOutcome()
	if currentGame.Outcome.Status == "checkmate" || currentGame.Outcome.Status == "stalemate" {
		message := currentGame.Outcome.Message
		if message == "" {
			message = "Game already ended."
		}
		http.Error(w, message, http.StatusConflict)
		return
	}

	expectedColor := sessionpkg.CurrentTurnColor()
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

	normalizedMove, err := sessionpkg.ApplyMoveByCommand(commandText)
	if err != nil {
		log.Printf("warning: failed to apply command %q: %v", commandText, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	finalGame := sessionpkg.RefreshGameSessionOutcome()

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
		History []string                `json:"history"`
		State   []sessionpkg.PieceState `json:"state"`
	}{
		Command:     normalizedMove,
		CurrentTurn: sessionpkg.CurrentTurnLabel(),
		CheckedSide: sessionpkg.CheckedSideLabel(),
		Game:        finalGame,
		Captured:    sessionpkg.GetCapturedSummary(),
		History:     sessionpkg.GetMoveHistory(),
		State:       sessionpkg.GetBoardState(),
	}
	response.From.File = string(parsed.FromFile)
	response.From.Rank = parsed.FromRank
	response.To.File = string(parsed.ToFile)
	response.To.Rank = parsed.ToRank

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Response encode error", http.StatusInternalServerError)
	}
}
