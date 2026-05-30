// CM3070 FP code
// router.go - registers page routes and static resources

package main

import (
	"go_backend/cssbuild"
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
	tailwindPath := frontendPath("styles", "tailwindcss")
	commandScriptPath := frontendPath("scripts", "chess_command.js")
	iconPath := frontendPath("pic", "icon.png")
	picDir := frontendPath("pic/")
	soundDir := frontendPath("sounds")

	serveNoCache := func(path string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
			http.ServeFile(w, r, path)
		}
	}

	mux.HandleFunc("/styles/style.css", func(w http.ResponseWriter, r *http.Request) {
		if err := cssbuild.EnsureStyleCSS(inputCSSPath, styleCSSPath, tailwindPath); err != nil {
			http.Error(w, "Failed to build stylesheet", http.StatusInternalServerError)
			return
		}
		serveNoCache(styleCSSPath)(w, r)
	})
	mux.HandleFunc("/styles/input.css", serveNoCache(inputCSSPath))
	mux.HandleFunc("/scripts/chess_command.js", serveNoCache(commandScriptPath))

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
	mux.HandleFunc("/game/new", h.NewGame)
	mux.HandleFunc("/game/flag", h.FlagGame)
	mux.HandleFunc("/game/config", h.UpdateGameConfig)
	mux.HandleFunc("/game/legal-moves", h.GetLegalMoves)
	mux.HandleFunc("/game/analysis/latest", h.GetLatestAnalysis)
}
