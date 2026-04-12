package main

import (
	"fmt"
	chesspieces "go_backend/chesspiece"
	"go_backend/components"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// captures and stores the response status code
func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// writes response data and defaults status to 200 when missing
func (sr *statusRecorder) Write(data []byte) (int, error) {
	if sr.statusCode == 0 {
		sr.statusCode = http.StatusOK
	}
	return sr.ResponseWriter.Write(data)
}

// maps status codes to a short status report label
func statusReport(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "success"
	case code >= 300 && code < 400:
		return "redirect"
	case code >= 400 && code < 500:
		return "client error"
	case code >= 500:
		return "server error"
	default:
		return "unknown"
	}
}

// wraps handlers to log request path, status code, and report
func withRequestLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		next(recorder, r)
		if strings.HasPrefix(r.URL.Path, "/.well-known/") {
			return
		}
		if recorder.statusCode == 0 {
			recorder.statusCode = http.StatusOK
		}
		log.Printf("loading page: %s %s -> %d %s [%s]",
			r.Method,
			r.URL.Path,
			recorder.statusCode,
			http.StatusText(recorder.statusCode),
			statusReport(recorder.statusCode),
		)
	}
}

type Page struct {
	Title string
	Body  []byte
}

// saves a page body to a title-based text file
func (p *Page) save() error {
	filename := p.Title + ".txt"
	return os.WriteFile(filename, p.Body, 0600)
}

// loads a page body from a title-based text file
func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

// renders a local html template file for wiki pages
func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	t, err := template.ParseFiles(tmpl + ".html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Printf("template parse error for %s: %v", tmpl, err)
		return
	}
	if err := t.Execute(w, p); err != nil {
		http.Error(w, "Template render error", http.StatusInternalServerError)
		log.Printf("template execute error for %s: %v", tmpl, err)
	}
}

// handles viewing a page and redirects to edit when missing
func viewHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/view/"):]
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

// handles editing a page and initializes an empty page when missing
func editHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/edit/"):]
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

// handles saving posted page content and redirects to view
func saveHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/save/"):]
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	if err := p.save(); err != nil {
		http.Error(w, "Failed to save page", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

// routes and renders the index page template bundle
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

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

	var mainHTMLCode strings.Builder
	mainHTMLCode.WriteString(`<div class="game_panel">`)
	mainHTMLCode.WriteString(`<div class="game_panel_left">`)
	mainHTMLCode.WriteString(string(generateChessBoard()))
	mainHTMLCode.WriteString(`</div>`)
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

	data := struct {
		PageTitle   string
		MainContent template.HTML
	}{
		PageTitle:   "Chess Game",
		MainContent: template.HTML(mainHTMLCode.String()),
	}

	if err := t.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, "Template render error", http.StatusInternalServerError)
		log.Printf("index template execute error: %v", err)
		return
	}
}

func generateChessBoard() template.HTML {
	gameBoard := components.NewChessBoard()
	return gameBoard.Draw()
}

// configures routes and starts the http server
func main() {
	http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir("../frontend/styles"))))

	http.HandleFunc("/", withRequestLogging(indexHandler))
	http.HandleFunc("/view/", withRequestLogging(viewHandler))
	http.HandleFunc("/edit/", withRequestLogging(editHandler))
	http.HandleFunc("/save/", withRequestLogging(saveHandler))

	log.Println("server successfully loaded at http://localhost:8080")
	fmt.Printf("%v", chesspieces.ChessPieces)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
