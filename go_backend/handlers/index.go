// CM3070 FP code
// index.go - game playing page
// index page is having chess board (handle in issue 1), chess pieces (handle in issue 2), and other elements (to be handled later)

package handlers

import (
	"go_backend/chessboard"
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
	mainHTMLCode.WriteString(`<label for="chess-command">Chess command</label>`)
	mainHTMLCode.WriteString(`<div class="command_row">`)
	mainHTMLCode.WriteString(`<input id="chess-command" type="text" placeholder="e2e4" />`)
	mainHTMLCode.WriteString(`<button type="button">Submit</button>`)
	mainHTMLCode.WriteString(`</div>`)
	mainHTMLCode.WriteString(`</div>`)

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
