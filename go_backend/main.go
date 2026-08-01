// CM3070 FP code
// main.go - initiate the backend go server

package main

import (
	"go_backend/game/session"
	"go_backend/gamearchive"
	"go_backend/handlers"
	"log"
	"net/http"
)

// starts the go http server for the chess game backend
func main() {
	session.SetArchiveStore(gamearchive.NewJSONFileStore(gamearchive.ResolvePath()))

	// initialize handler and router
	h := handlers.NewHandler()
	handlers.StartAnalyzerWorker()
	mux := http.NewServeMux()
	registerRoutes(mux, h)

	// log startup status
	log.Println("server successfully loaded at http://localhost:8080")
	if err := http.ListenAndServe(":8080", withRequestLogging(mux)); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
