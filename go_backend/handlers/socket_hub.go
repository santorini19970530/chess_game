package handlers

import (
	"encoding/json"
	"sync"
	"time"
)

type socketEventType string

const (
	socketEventMoveApplied       socketEventType = "move_applied"
	socketEventTurnChanged       socketEventType = "turn_changed"
	socketEventGameOutcome       socketEventType = "game_outcome"
	socketEventAnalysisStatus    socketEventType = "analysis_status_update"
	socketEventExplanationReady  socketEventType = "explanation_ready"

	// Simulation live progress events (issue0030 Option A)
	socketEventSimulationStarted socketEventType = "simulation_started"
	socketEventSimulationMove    socketEventType = "simulation_move"
	socketEventSimulationGameEnd socketEventType = "simulation_game_end"
	socketEventSimulationDone    socketEventType = "simulation_completed"
)

type socketEnvelope struct {
	Event        socketEventType `json:"event"`
	GameID       string          `json:"game_id"`
	TimestampUTC string          `json:"timestamp_utc"`
	Data         interface{}     `json:"data,omitempty"`
}

type socketHub struct {
	mu            sync.RWMutex
	clientsByGame map[string]map[string]chan<- []byte
}

func newSocketHub() *socketHub {
	return &socketHub{
		clientsByGame: make(map[string]map[string]chan<- []byte),
	}
}

func (h *socketHub) RegisterClient(gameID, clientID string, send chan<- []byte) {
	if h == nil || gameID == "" || clientID == "" || send == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clientsByGame[gameID]; !ok {
		h.clientsByGame[gameID] = make(map[string]chan<- []byte)
	}
	h.clientsByGame[gameID][clientID] = send
}

func (h *socketHub) UnregisterClient(gameID, clientID string) {
	if h == nil || gameID == "" || clientID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	clients, ok := h.clientsByGame[gameID]
	if !ok {
		return
	}
	delete(clients, clientID)
	if len(clients) == 0 {
		delete(h.clientsByGame, gameID)
	}
}

func (h *socketHub) ClientCount(gameID string) int {
	if h == nil || gameID == "" {
		return 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clientsByGame[gameID])
}

func (h *socketHub) Broadcast(gameID string, event socketEventType, data interface{}) int {
	if h == nil || gameID == "" {
		return 0
	}
	envelope := socketEnvelope{
		Event:        event,
		GameID:       gameID,
		TimestampUTC: time.Now().UTC().Format(time.RFC3339Nano),
		Data:         data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return 0
	}

	h.mu.RLock()
	clients := h.clientsByGame[gameID]
	sinks := make([]chan<- []byte, 0, len(clients))
	for _, sink := range clients {
		sinks = append(sinks, sink)
	}
	h.mu.RUnlock()

	delivered := 0
	for _, sink := range sinks {
		select {
		case sink <- payload:
			delivered++
		default:
			// Non-blocking broadcast: slow clients are skipped.
		}
	}
	return delivered
}

var gameSocketHub = newSocketHub()

// BroadcastGlobal sends a message to every connected WebSocket client
// regardless of which game they are viewing. Useful for simulation progress.
func (h *socketHub) BroadcastGlobal(event socketEventType, data interface{}) {
	if h == nil {
		return
	}
	envelope := socketEnvelope{
		Event:        event,
		TimestampUTC: time.Now().UTC().Format(time.RFC3339Nano),
		Data:         data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return
	}

	h.mu.RLock()
	var sinks []chan<- []byte
	for _, clients := range h.clientsByGame {
		for _, sink := range clients {
			sinks = append(sinks, sink)
		}
	}
	h.mu.RUnlock()

	for _, sink := range sinks {
		select {
		case sink <- payload:
		default:
		}
	}
}
