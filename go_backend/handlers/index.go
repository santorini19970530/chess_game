// CM3070 FP code
// index.go - game playing page and state response

package handlers

import (
	chessboard "go_backend/game/board"
	sessionpkg "go_backend/game/session"
	"html/template"
	"log"
	"net/http"
)

// gameStateResponse - status of the game that share with the frontend
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

// indexPageData - fields for the index html templates
type indexPageData struct {
	PageTitle      string
	BoardHTML      template.HTML
	WhiteTurnClass string
	BlackTurnClass string
}

// generateChessBoard - builds the chessboard html for the index page
func generateChessBoard() template.HTML {
	return DrawChessBoard(chessboard.NewChessBoard())
}

// Index - renders the main game page template
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	t, err := template.ParseFiles(
		"../frontend/index.html",
		"../frontend/html_puzzles/head.html",
		"../frontend/html_puzzles/header.html",
		"../frontend/html_puzzles/footer.html",
		"../frontend/html_puzzles/game_panel.html",
		"../frontend/html_puzzles/game_config.html",
		"../frontend/html_puzzles/game_info.html",
		"../frontend/html_puzzles/game_play.html",
	)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("index template parse error: %v", err)
		return
	}

	whiteTurnClass := "game_info_col_white"
	blackTurnClass := "game_info_col_black"
	if sessionpkg.CurrentTurnLabel() == "White" {
		whiteTurnClass += " game_info_col_active"
	} else {
		blackTurnClass += " game_info_col_active"
	}

	data := indexPageData{
		PageTitle:      "Chess Game",
		BoardHTML:      generateChessBoard(),
		WhiteTurnClass: whiteTurnClass,
		BlackTurnClass: blackTurnClass,
	}

	if err := t.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, "Template render error", http.StatusInternalServerError)
		log.Printf("index template execute error: %v", err)
		return
	}
}
