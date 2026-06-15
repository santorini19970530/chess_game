package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	sessionpkg "go_backend/game/session"

	"github.com/gorilla/websocket"
)

var (
	socketClientCounter uint64
	wsUpgrader          = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(_ *http.Request) bool {
			// Socket-first local development: allow same-host tooling/origin.
			return true
		},
	}
)

func nextSocketClientID() string {
	n := atomic.AddUint64(&socketClientCounter, 1)
	return fmt.Sprintf("ws-client-%d", n)
}

func (h *Handler) GameSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	gameID := strings.TrimSpace(r.URL.Query().Get("gameId"))
	if gameID == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing required query param: gameId")
		return
	}
	if _, err := sessionpkg.GetGameSessionByID(gameID); err != nil {
		writeJSONError(w, http.StatusNotFound, "Game session not found")
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("warning: websocket upgrade failed %s: %v", gameIDLabel(gameID), err)
		return
	}

	clientID := nextSocketClientID()
	send := make(chan []byte, 32)
	done := make(chan struct{})

	gameSocketHub.RegisterClient(gameID, clientID, send)
	log.Printf("websocket connected %s client=%s total=%d", gameIDLabel(gameID), clientID, gameSocketHub.ClientCount(gameID))

	go func() {
		for {
			select {
			case payload := <-send:
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	close(done)
	gameSocketHub.UnregisterClient(gameID, clientID)
	_ = conn.Close()
	log.Printf("websocket disconnected %s client=%s total=%d", gameIDLabel(gameID), clientID, gameSocketHub.ClientCount(gameID))
}
