// CM3070 FP code
// main.go - initiate the backend go server

package main

import (
	"go_backend/handlers"
	"log"
	"net/http"
)

func main() {
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
