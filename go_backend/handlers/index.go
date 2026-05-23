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

	// left panel
	mainHTMLCode.WriteString(`<div class="game_panel">`)

	mainHTMLCode.WriteString(`<div class="game_panel_left">`)
	mainHTMLCode.WriteString(string(generateChessBoard()))
	mainHTMLCode.WriteString(`</div>`)

	// right panel
	mainHTMLCode.WriteString(`<div class="game_panel_right_top">`)
	mainHTMLCode.WriteString(`<h2>Game Information</h2>`)
	mainHTMLCode.WriteString(`<ul>`)
	mainHTMLCode.WriteString(`<li>Status: waiting for first move</li>`)
	mainHTMLCode.WriteString(`<li>Current turn: White</li>`)
	mainHTMLCode.WriteString(`<li>Win probability: to be developed</li>`)
	mainHTMLCode.WriteString(`</ul>`)
	mainHTMLCode.WriteString(`</div>`)

	mainHTMLCode.WriteString(`<div class="game_panel_right_bottom">`)
	mainHTMLCode.WriteString(`<label for="chess_command">Chess command</label>`)
	mainHTMLCode.WriteString(`<div class="command_row">`)
	mainHTMLCode.WriteString(`<input id="chess_command" type="text" placeholder="e2e4" />`)
	mainHTMLCode.WriteString(`<button id="chess_command_submit" type="button">Submit</button>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<p id="chess_command_status" class="command_status" role="status" aria-live="polite"></p>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_section">`)
	mainHTMLCode.WriteString(`<h3 class="chess_move_history_title">Move history</h3>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panels">`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panel">`)
	mainHTMLCode.WriteString(`<h4 class="chess_move_history_side_title">White</h4>`)
	mainHTMLCode.WriteString(`<ol id="chess_move_history_white" class="chess_move_history_list">`)
	mainHTMLCode.WriteString(`<li class="chess_move_history_placeholder">No moves yet.</li>`)
	mainHTMLCode.WriteString(`</ol>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`<div class="chess_move_history_panel">`)
	mainHTMLCode.WriteString(`<h4 class="chess_move_history_side_title">Black</h4>`)
	mainHTMLCode.WriteString(`<ol id="chess_move_history_black" class="chess_move_history_list">`)
	mainHTMLCode.WriteString(`<li class="chess_move_history_placeholder">No moves yet.</li>`)
	mainHTMLCode.WriteString(`</ol>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)
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

	expectedColor := sessionpkg.CurrentTurnColor()
	parsed, err := commandpkg.ParseCommandForColor(commandText, expectedColor)
	if err != nil {
		log.Printf("warning: invalid chess command format: %q", commandText)
		http.Error(w, "Invalid command format (use e2e4 or ng1f3)", http.StatusBadRequest)
		return
	}
	if err := commandpkg.ParseAndLogCommandForColor(commandText, expectedColor); err != nil {
		http.Error(w, "Invalid command format (use e2e4 or ng1f3)", http.StatusBadRequest)
		return
	}

	normalizedMove, err := sessionpkg.ApplyMoveByCommand(commandText)
	if err != nil {
		log.Printf("warning: failed to apply command %q: %v", commandText, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := struct {
		Command string `json:"command"`
		From    struct {
			File string `json:"file"`
			Rank int    `json:"rank"`
		} `json:"from"`
		To struct {
			File string `json:"file"`
			Rank int    `json:"rank"`
		} `json:"to"`
		History []string               `json:"history"`
		State   []sessionpkg.PieceState `json:"state"`
	}{
		Command: normalizedMove,
		History: sessionpkg.GetMoveHistory(),
		State:   sessionpkg.GetBoardState(),
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
