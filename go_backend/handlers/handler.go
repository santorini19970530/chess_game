// CM3070 FP code
// handler.go - defines shared handler types and helper functions

package handlers

import (
	"html/template"
	"log"
	"net/http"
)

// Handler is the root handler type
type Handler struct{}

// Page stores a page title and page body content
type Page struct {
	Title string
	Body  []byte
}

// NewHandler returns a Handler instance
func NewHandler() *Handler {
	return &Handler{}
}

// renderTemplate parses and executes a local html template
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
