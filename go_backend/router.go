// CM3070 FP code
// router.go - registers page routes and static resources

package main

import (
	"go_backend/handlers"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func frontendPath(parts ...string) string {
	_, thisFile, _, ok := runtime.Caller(0)
	baseDir := "."
	if ok {
		baseDir = filepath.Dir(thisFile)
	}

	pathParts := append([]string{baseDir, "..", "frontend"}, parts...)
	return filepath.Clean(filepath.Join(pathParts...))
}

// registerRoutes registers all routes for the web app
func registerRoutes(mux *http.ServeMux, h *handlers.Handler) {
	styleCSSPath := frontendPath("styles", "style.css")
	inputCSSPath := frontendPath("styles", "input.css")
	commandScriptPath := frontendPath("scripts", "chess_command.js")
	iconPath := frontendPath("pic", "icon.png")
	picDir := frontendPath("pic/")
	soundDir := frontendPath("sounds")

	// css files
	mux.HandleFunc("/styles/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, styleCSSPath)
	})

	mux.HandleFunc("/styles/input.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, inputCSSPath)
	})

	// command js script
	mux.HandleFunc("/scripts/chess_command.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, commandScriptPath)
	})

	// favicon and icon routes
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(iconPath); err == nil {
			http.ServeFile(w, r, iconPath)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/icon.png", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(iconPath); err == nil {
			http.ServeFile(w, r, iconPath)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// pieces pictures
	mux.Handle("/pic/", http.StripPrefix("/pic/", http.FileServer(http.Dir(picDir))))
	if _, err := os.Stat(picDir); err != nil {
		log.Printf("warning: piece image directory not found at %s: %v", picDir, err)
	}

	// sounds
	mux.Handle("/sounds/", http.StripPrefix("/sounds/", http.FileServer(http.Dir(soundDir))))
	if _, err := os.Stat(soundDir); err != nil {
		log.Printf("warning: sound directory not found at %s: %v", soundDir, err)
	}

	// page routes
	mux.HandleFunc("/", h.Index) // index
	mux.HandleFunc("/command", h.SubmitChessCommand)
}
