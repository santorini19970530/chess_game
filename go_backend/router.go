// CM3070 FP code
// router.go - registers page routes and static resources

package main

import (
	"go_backend/handlers"
	"net/http"
	"os"
)

// registerRoutes registers all routes for the web app
func registerRoutes(mux *http.ServeMux, h *handlers.Handler) {

	// css files
	mux.HandleFunc("/styles/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/styles/style.css")
	})

	mux.HandleFunc("/styles/input.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../frontend/styles/input.css")
	})

	// favicon and icon routes
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat("../frontend/pic/icon.png"); err == nil {
			http.ServeFile(w, r, "../frontend/pic/icon.png")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/icon.png", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat("../frontend/pic/icon.png"); err == nil {
			http.ServeFile(w, r, "../frontend/pic/icon.png")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// page routes
	mux.HandleFunc("/", h.Index) // index
	// below routings come from GO wiki tutorial page, to be updated in future development
	mux.HandleFunc("/view/", h.View)
	mux.HandleFunc("/edit/", h.Edit)
	mux.HandleFunc("/save/", h.Save)
}
